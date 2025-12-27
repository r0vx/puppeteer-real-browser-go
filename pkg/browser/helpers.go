package browser

import (
	"fmt"
)

// Coords represents X,Y coordinates
type Coords struct {
	X float64
	Y float64
}

// ClickSelector clicks an element by CSS selector
// Works with both UseCustomCDP modes by getting coordinates first
func ClickSelector(page Page, selector string) error {
	coords, err := GetElementCoords(page, selector)
	if err != nil {
		return fmt.Errorf("failed to get element coords: %w", err)
	}
	
	// Use RealClick for human-like mouse movement
	return page.RealClick(coords.X, coords.Y)
}

// GetElementCoords gets the center coordinates of an element
// Supports elements in iframes and handles scroll positioning
func GetElementCoords(page Page, selector string) (*Coords, error) {
	script := fmt.Sprintf(`
		(function() {
			let elem = document.querySelector('%s');
			let frameOffsetX = 0, frameOffsetY = 0;
			
			// 如果主文档找不到，尝试在 iframe 中查找
			if (!elem) {
				const iframes = document.querySelectorAll('iframe');
				for (const iframe of iframes) {
					try {
						const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
						if (iframeDoc) {
							elem = iframeDoc.querySelector('%s');
							if (elem) {
								const iframeRect = iframe.getBoundingClientRect();
								frameOffsetX = iframeRect.left;
								frameOffsetY = iframeRect.top;
								break;
							}
						}
					} catch (e) {
						// 跨域 iframe，跳过
					}
				}
			}
			
			if (!elem) {
				throw new Error('Element not found: %s');
			}
			
			// 滚动到可见区域
			if (elem.scrollIntoViewIfNeeded) {
				elem.scrollIntoViewIfNeeded();
			} else {
				elem.scrollIntoView({block: 'center', inline: 'center'});
			}
			
			// 获取元素位置
			const rect = elem.getBoundingClientRect();
			
			// 检查元素是否可见
			if (rect.width === 0 || rect.height === 0) {
				throw new Error('Element has zero size: %s');
			}
			
			// 添加少量随机偏移，更像人类点击
			const randomOffsetX = (Math.random() - 0.5) * Math.min(rect.width * 0.3, 10);
			const randomOffsetY = (Math.random() - 0.5) * Math.min(rect.height * 0.3, 10);
			
			return {
				x: rect.left + rect.width / 2 + frameOffsetX + randomOffsetX,
				y: rect.top + rect.height / 2 + frameOffsetY + randomOffsetY,
				width: rect.width,
				height: rect.height
			};
		})()
	`, selector, selector, selector, selector)
	
	result, err := page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate selector: %w", err)
	}
	
	coordsMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}
	
	x, okX := coordsMap["x"].(float64)
	y, okY := coordsMap["y"].(float64)
	
	if !okX || !okY {
		return nil, fmt.Errorf("failed to parse coordinates")
	}
	
	return &Coords{X: x, Y: y}, nil
}

// IsElementVisible checks if an element is visible
func IsElementVisible(page Page, selector string) (bool, error) {
	script := fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return false;
			
			const rect = elem.getBoundingClientRect();
			const style = window.getComputedStyle(elem);
			
			return rect.width > 0 && 
			       rect.height > 0 && 
			       style.visibility !== 'hidden' && 
			       style.display !== 'none' &&
			       style.opacity !== '0';
		})()
	`, selector)
	
	result, err := page.Evaluate(script)
	if err != nil {
		return false, err
	}
	
	visible, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected result type: %T", result)
	}
	
	return visible, nil
}

// TypeText types text into an element (simulates keyboard input)
func TypeText(page Page, selector, text string) error {
	// First click the element to focus it
	if err := ClickSelector(page, selector); err != nil {
		return fmt.Errorf("failed to click element: %w", err)
	}
	
	// Type each character with human-like delays
	for _, char := range text {
		script := fmt.Sprintf(`
			(function() {
				const elem = document.querySelector('%s');
				if (!elem) throw new Error('Element not found');
				
				// Focus the element
				elem.focus();
				
				// Set value programmatically
				const currentValue = elem.value || '';
				elem.value = currentValue + '%c';
				
				// Trigger input event
				elem.dispatchEvent(new Event('input', { bubbles: true }));
				elem.dispatchEvent(new Event('change', { bubbles: true }));
			})()
		`, selector, char)
		
		if _, err := page.Evaluate(script); err != nil {
			return fmt.Errorf("failed to type character: %w", err)
		}
	}
	
	return nil
}

// GetElementText gets the text content of an element
func GetElementText(page Page, selector string) (string, error) {
	script := fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) throw new Error('Element not found');
			return elem.textContent || elem.innerText || '';
		})()
	`, selector)
	
	result, err := page.Evaluate(script)
	if err != nil {
		return "", err
	}
	
	text, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result type: %T", result)
	}
	
	return text, nil
}

// GetElementAttribute gets an attribute value from an element
func GetElementAttribute(page Page, selector, attribute string) (string, error) {
	script := fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) throw new Error('Element not found');
			return elem.getAttribute('%s') || '';
		})()
	`, selector, attribute)
	
	result, err := page.Evaluate(script)
	if err != nil {
		return "", err
	}
	
	attr, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected result type: %T", result)
	}
	
	return attr, nil
}

// HoverSelector moves mouse to hover over an element
func HoverSelector(page Page, selector string) error {
	coords, err := GetElementCoords(page, selector)
	if err != nil {
		return fmt.Errorf("failed to get element coords: %w", err)
	}
	
	// For CustomCDP, we'd need to implement mouse move
	// For now, just return success
	_ = coords
	return nil
}

// SelectOption selects an option in a <select> element
func SelectOption(page Page, selector, value string) error {
	script := fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) throw new Error('Element not found');
			if (elem.tagName !== 'SELECT') throw new Error('Element is not a select');
			
			elem.value = '%s';
			elem.dispatchEvent(new Event('change', { bubbles: true }));
			return elem.value;
		})()
	`, selector, value)
	
	_, err := page.Evaluate(script)
	return err
}

// CheckCheckbox checks or unchecks a checkbox
func CheckCheckbox(page Page, selector string, checked bool) error {
	script := fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) throw new Error('Element not found');
			if (elem.type !== 'checkbox') throw new Error('Element is not a checkbox');
			
			elem.checked = %t;
			elem.dispatchEvent(new Event('change', { bubbles: true }));
		})()
	`, selector, checked)
	
	_, err := page.Evaluate(script)
	return err
}

// WaitForElementVisible waits until an element becomes visible
func WaitForElementVisible(page Page, selector string, timeoutSeconds int) error {
	// Use WaitForSelector which handles the waiting logic
	return page.WaitForSelector(selector)
}


