package turnstile

import (
	"context"
	"fmt"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

// Solver handles Cloudflare Turnstile captcha solving
type Solver struct {
	page      browser.Page
	ctx       context.Context
	isRunning bool
	stopChan  chan bool
}

// NewSolver creates a new Turnstile solver
func NewSolver(page browser.Page, ctx context.Context) *Solver {
	return &Solver{
		page:     page,
		ctx:      ctx,
		stopChan: make(chan bool, 1),
	}
}

// Start starts the Turnstile solver
func (s *Solver) Start() error {
	if s.isRunning {
		return nil
	}

	s.isRunning = true
	go s.run()
	return nil
}

// Stop stops the Turnstile solver
func (s *Solver) Stop() error {
	if !s.isRunning {
		return nil
	}

	s.isRunning = false
	s.stopChan <- true
	return nil
}

// IsRunning returns whether the solver is currently running
func (s *Solver) IsRunning() bool {
	return s.isRunning
}

// WaitForSolution waits for a Turnstile solution with timeout
func (s *Solver) WaitForSolution(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Turnstile solution")
		case <-ticker.C:
			if s.checkForSolution() {
				return nil
			}
		}
	}
}

// run is the main solver loop
func (s *Solver) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndSolve()
		}
	}
}

// checkAndSolve checks for and solves Turnstile captchas
func (s *Solver) checkAndSolve() {
	// Check for Turnstile elements
	turnstileElements := s.findTurnstileElements()
	if len(turnstileElements) == 0 {
		return
	}

	// Try to solve each found element
	for _, element := range turnstileElements {
		s.solveElement(element)
	}
}

// findTurnstileElements finds Turnstile elements on the page
func (s *Solver) findTurnstileElements() []map[string]interface{} {
	script := `
		(() => {
			const elements = [];
			
			// Look for Turnstile response elements
			const responseElements = document.querySelectorAll('[name="cf-turnstile-response"]');
			responseElements.forEach(el => {
				const parent = el.parentElement;
				if (parent) {
					const rect = parent.getBoundingClientRect();
					elements.push({
						type: 'response',
						x: rect.x + rect.width / 2,
						y: rect.y + rect.height / 2,
						width: rect.width,
						height: rect.height,
						element: el
					});
				}
			});
			
			// Look for Turnstile iframes
			const iframes = document.querySelectorAll('iframe[src*="turnstile"]');
			iframes.forEach(iframe => {
				const rect = iframe.getBoundingClientRect();
				elements.push({
					type: 'iframe',
					x: rect.x + rect.width / 2,
					y: rect.y + rect.height / 2,
					width: rect.width,
					height: rect.height,
					element: iframe
				});
			});
			
			// Look for Turnstile checkboxes (divs with specific characteristics)
			const divs = document.querySelectorAll('div');
			divs.forEach(div => {
				try {
					const rect = div.getBoundingClientRect();
					const style = window.getComputedStyle(div);
					
					// Check for Turnstile-like characteristics
					if (rect.width >= 290 && rect.width <= 310 && 
						rect.height >= 60 && rect.height <= 80 &&
						style.margin === "0px" && style.padding === "0px" &&
						!div.querySelector('*')) {
						elements.push({
							type: 'checkbox',
							x: rect.x + rect.width / 2,
							y: rect.y + rect.height / 2,
							width: rect.width,
							height: rect.height,
							element: div
						});
					}
				} catch (err) {
					// Ignore errors
				}
			});
			
			return elements;
		})();
	`

	result, err := s.page.Evaluate(script)
	if err != nil {
		return nil
	}

	if elements, ok := result.([]interface{}); ok {
		var turnstileElements []map[string]interface{}
		for _, element := range elements {
			if elementMap, ok := element.(map[string]interface{}); ok {
				turnstileElements = append(turnstileElements, elementMap)
			}
		}
		return turnstileElements
	}

	return nil
}

// solveElement attempts to solve a Turnstile element
func (s *Solver) solveElement(element map[string]interface{}) {
	elementType, _ := element["type"].(string)
	x, _ := element["x"].(float64)
	y, _ := element["y"].(float64)

	switch elementType {
	case "checkbox":
		// Click on the checkbox
		s.clickElement(x, y)
	case "iframe":
		// Try to interact with iframe content
		s.interactWithIframe(x, y)
	case "response":
		// This is usually a hidden input, try to find and click the checkbox
		s.clickElement(x, y)
	}
}

// clickElement clicks on an element with realistic mouse movement
func (s *Solver) clickElement(x, y float64) {
	// Add some randomness to the click position
	randomX := x + (float64(time.Now().UnixNano()%100)-50)/100
	randomY := y + (float64(time.Now().UnixNano()%100)-50)/100

	// Perform the click
	s.page.Click(randomX, randomY)

	// Wait a bit after clicking
	time.Sleep(500 * time.Millisecond)
}

// interactWithIframe attempts to interact with iframe content
func (s *Solver) interactWithIframe(x, y float64) {
	script := fmt.Sprintf(`
		(() => {
			const iframes = document.querySelectorAll('iframe[src*="turnstile"]');
			for (const iframe of iframes) {
				try {
					const rect = iframe.getBoundingClientRect();
					if (rect.x <= %f && rect.x + rect.width >= %f &&
						rect.y <= %f && rect.y + rect.height >= %f) {
						
						// Try to click inside the iframe
						const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
						if (iframeDoc) {
							const checkbox = iframeDoc.querySelector('input[type="checkbox"]');
							if (checkbox) {
								checkbox.click();
								return true;
							}
						}
					}
				} catch (err) {
					// Cross-origin iframe, can't access content
					continue;
				}
			}
			return false;
		})();
	`, x, x, y, y)

	s.page.Evaluate(script)
}

// checkForSolution checks if a Turnstile solution has been found
func (s *Solver) checkForSolution() bool {
	script := `
		(() => {
			// Check for filled response elements
			const responseElements = document.querySelectorAll('[name="cf-turnstile-response"]');
			for (const element of responseElements) {
				if (element.value && element.value.length > 0) {
					return true;
				}
			}
			
			// Check for success indicators
			const successIndicators = document.querySelectorAll('.cf-turnstile-success, .turnstile-success');
			if (successIndicators.length > 0) {
				return true;
			}
			
			// Check if challenge is no longer visible
			const challenges = document.querySelectorAll('iframe[src*="turnstile"]');
			let visibleChallenges = 0;
			for (const challenge of challenges) {
				const rect = challenge.getBoundingClientRect();
				if (rect.width > 0 && rect.height > 0) {
					visibleChallenges++;
				}
			}
			
			return visibleChallenges === 0;
		})();
	`

	result, err := s.page.Evaluate(script)
	if err != nil {
		return false
	}

	if solved, ok := result.(bool); ok {
		return solved
	}

	return false
}
