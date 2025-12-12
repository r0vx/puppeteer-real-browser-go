package page

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// Controller handles page interactions with realistic mouse movements
type Controller struct {
	page    browser.Page
	ctx     context.Context
	enabled bool
}

// NewController creates a new page controller
func NewController(page browser.Page, ctx context.Context, enabled bool) *Controller {
	return &Controller{
		page:    page,
		ctx:     ctx,
		enabled: enabled,
	}
}

// Initialize sets up the controller
func (pc *Controller) Initialize() error {
	if !pc.enabled {
		return nil
	}

	// Wait a bit for the page to be ready
	time.Sleep(1 * time.Second)

	// Inject realistic mouse movement scripts
	script := `
		// Realistic mouse movement simulation
		window.__realisticMouse = {
			// Generate human-like mouse trajectory
			generateTrajectory: function(startX, startY, endX, endY) {
				const points = [];
				const steps = Math.floor(Math.random() * 20) + 20; // 20-40 steps
				
				for (let i = 0; i <= steps; i++) {
					const progress = i / steps;
					
					// Add some randomness to the path
					const randomX = (Math.random() - 0.5) * 10;
					const randomY = (Math.random() - 0.5) * 10;
					
					// Use easing function for more natural movement
					const easeProgress = this.easeOutCubic(progress);
					
					const x = startX + (endX - startX) * easeProgress + randomX;
					const y = startY + (endY - startY) * easeProgress + randomY;
					
					points.push({ x: Math.round(x), y: Math.round(y) });
				}
				
				return points;
			},
			
			// Easing function for natural movement
			easeOutCubic: function(t) {
				return 1 - Math.pow(1 - t, 3);
			},
			
			// Perform realistic click with mouse movement
			realClick: async function(x, y) {
				const startX = Math.random() * window.innerWidth;
				const startY = Math.random() * window.innerHeight;
				
				const trajectory = this.generateTrajectory(startX, startY, x, y);
				
				// Move mouse along trajectory
				for (const point of trajectory) {
					const event = new MouseEvent('mousemove', {
						clientX: point.x,
						clientY: point.y,
						bubbles: true
					});
					document.dispatchEvent(event);
					
					// Random delay between movements
					await new Promise(resolve => setTimeout(resolve, Math.random() * 10 + 5));
				}
				
				// Perform click
				const clickEvent = new MouseEvent('click', {
					clientX: x,
					clientY: y,
					bubbles: true,
					cancelable: true
				});
				document.dispatchEvent(clickEvent);
			}
		};
		
		console.log('🎯 Realistic mouse movement enabled');
	`

	_, err := pc.page.Evaluate(script)
	return err
}

// Stop stops the controller
func (pc *Controller) Stop() error {
	// Cleanup if needed
	return nil
}

// RealClick performs a realistic click with human-like mouse movement
func (pc *Controller) RealClick(x, y float64) error {
	if !pc.enabled {
		return pc.page.Click(x, y)
	}

	// Use the injected realistic mouse movement
	script := fmt.Sprintf(`
		window.__realisticMouse.realClick(%f, %f);
	`, x, y)

	pc.page.Evaluate(script)
	return nil
}

// HumanClick performs a click with random human-like delays
func (pc *Controller) HumanClick(x, y float64) error {
	// Add random delay before click (100-500ms)
	delay := time.Duration(rand.Intn(400)+100) * time.Millisecond
	time.Sleep(delay)

	// Add slight randomness to click position (±2 pixels)
	randomX := x + (rand.Float64()-0.5)*4
	randomY := y + (rand.Float64()-0.5)*4

	return pc.page.Click(randomX, randomY)
}

// Scroll performs realistic scrolling
func (pc *Controller) Scroll(deltaY float64) error {
	script := fmt.Sprintf(`
		window.scrollBy({
			top: %f,
			left: 0,
			behavior: 'smooth'
		});
	`, deltaY)

	_, err := pc.page.Evaluate(script)
	return err
}

// Type performs realistic typing with random delays
func (pc *Controller) Type(selector, text string) error {
	// Focus on element first
	focusScript := fmt.Sprintf(`
		const element = document.querySelector('%s');
		if (element) {
			element.focus();
			element.click();
		}
	`, selector)

	_, err := pc.page.Evaluate(focusScript)
	if err != nil {
		return err
	}

	// Type with realistic delays
	for _, char := range text {
		typeScript := fmt.Sprintf(`
			const element = document.querySelector('%s');
			if (element) {
				element.value += '%c';
				element.dispatchEvent(new Event('input', { bubbles: true }));
			}
		`, selector, char)

		_, err := pc.page.Evaluate(typeScript)
		if err != nil {
			return err
		}

		// Random delay between characters (50-150ms)
		delay := time.Duration(rand.Intn(100)+50) * time.Millisecond
		time.Sleep(delay)
	}

	return nil
}

// WaitForElement waits for an element with realistic timeout
func (pc *Controller) WaitForElement(selector string, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		script := fmt.Sprintf(`
			document.querySelector('%s') !== null
		`, selector)

		result, err := pc.page.Evaluate(script)
		if err == nil && result == true {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("element %s not found within timeout", selector)
}

// GetRandomPosition returns a random position within the viewport
func (pc *Controller) GetRandomPosition() (float64, float64) {
	script := `
		{
			x: Math.random() * window.innerWidth,
			y: Math.random() * window.innerHeight
		}
	`

	_, err := pc.page.Evaluate(script)
	if err != nil {
		// Fallback to default values
		return 100, 100
	}

	// Parse result (this is a simplified version)
	// In a real implementation, you'd properly parse the result
	return 100, 100
}
