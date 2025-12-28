package browser

import (
	"context"

	"github.com/chromedp/chromedp"
)

// GetAdvancedStealthScript returns an advanced anti-detection script
// Based on rebrowser-patches and other anti-detection techniques
func GetAdvancedStealthScript() string {
	return `
		// Advanced anti-detection script for Go puppeteer-real-browser
		(() => {
			'use strict';
			
			// Prevent multiple injections
			if (window.__stealthInjected) return;
			window.__stealthInjected = true;
			
			// 1. Fix MouseEvent screenX and screenY properties (Critical for Cloudflare)
			if (!MouseEvent.prototype.hasOwnProperty('_screenFixed')) {
				const originalScreenX = Object.getOwnPropertyDescriptor(MouseEvent.prototype, 'screenX');
				const originalScreenY = Object.getOwnPropertyDescriptor(MouseEvent.prototype, 'screenY');
				
				Object.defineProperty(MouseEvent.prototype, 'screenX', {
					get: function() {
						return this.clientX + (window.screenX || 0);
					},
					configurable: true
				});
				
				Object.defineProperty(MouseEvent.prototype, 'screenY', {
					get: function() {
						return this.clientY + (window.screenY || 0);
					},
					configurable: true
				});
				
				MouseEvent.prototype._screenFixed = true;
			}
			
			// 2. Hide webdriver property completely (Critical)
			if (navigator.webdriver !== undefined) {
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
					configurable: true
				});
			}
			
			// 3. Override automation-related properties
			// Make plugins look realistic
			Object.defineProperty(navigator, 'plugins', {
				get: () => {
					const plugins = [
						{
							name: 'Chrome PDF Plugin',
							filename: 'internal-pdf-viewer',
							description: 'Portable Document Format',
							length: 1
						},
						{
							name: 'Chrome PDF Viewer',
							filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai',
							description: '',
							length: 1
						},
						{
							name: 'Native Client',
							filename: 'internal-nacl-plugin',
							description: '',
							length: 2
						}
					];
					
					// Add array-like properties
					plugins.refresh = () => {};
					return plugins;
				},
				configurable: true
			});
			
			// 4. Override languages to be more realistic
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
				configurable: true
			});
			
			// 5. Fix permissions API
			if (navigator.permissions && navigator.permissions.query) {
				const originalQuery = navigator.permissions.query.bind(navigator.permissions);
				navigator.permissions.query = (parameters) => {
					if (parameters.name === 'notifications') {
						return Promise.resolve({ 
							state: Notification.permission,
							onchange: null
						});
					}
					return originalQuery(parameters);
				};
			}
			
			// 6. Hide automation control features and make chrome object realistic
			if (!window.chrome) {
				window.chrome = {};
			}
			
			if (!window.chrome.runtime) {
				window.chrome.runtime = {
					onConnect: undefined,
					onMessage: undefined,
					onConnectExternal: undefined,
					onMessageExternal: undefined
				};
			}
			
			// 7. Hide CDP and automation-related properties
			delete window.__nightmare;
			delete window._phantom;
			delete window.callPhantom;
			delete window.__webdriver_script_fn;
			delete window.__webdriver_evaluate;
			delete window.__selenium_unwrapped;
			delete window.__webdriver_unwrapped;
			delete window.__driver_evaluate;
			delete window.__webdriver_script_function;
			delete window.__fxdriver_evaluate;
			delete window.__driver_unwrapped;
			delete window.webdriver;
			delete window.domAutomation;
			delete window.domAutomationController;
			delete window.__lastWatirAlert;
			delete window.__lastWatirConfirm;
			delete window.__lastWatirPrompt;
			delete window._WEBDRIVER_ELEM_CACHE;
			
			// 8. Override console methods to hide automation traces
			const originalConsole = {
				debug: console.debug,
				log: console.log,
				warn: console.warn,
				error: console.error
			};
			
			const filterAutomationLogs = (method, args) => {
				const message = args.join(' ');
				// Filter out CDP and automation-related messages
				if (message.includes('DevTools') || 
					message.includes('Runtime.enable') ||
					message.includes('puppeteer') ||
					message.includes('chromedp') ||
					message.includes('automation')) {
					return;
				}
				return originalConsole[method].apply(console, args);
			};
			
			console.debug = (...args) => filterAutomationLogs('debug', args);
			console.log = (...args) => filterAutomationLogs('log', args);
			console.warn = (...args) => filterAutomationLogs('warn', args);
			console.error = (...args) => filterAutomationLogs('error', args);
			
			// 9. Override Error.stack to hide automation traces
			// DISABLED: This breaks some pages (e.g., Douyin QR code)
			// const originalPrepareStackTrace = Error.prepareStackTrace;
			// Error.prepareStackTrace = function(error, stack) { ... };
			
			// 10. Prevent detection through timing attacks
			const originalPerformanceNow = performance.now;
			let timeOffset = Math.random() * 10 - 5; // Random offset between -5 and 5ms
			performance.now = function() {
				return originalPerformanceNow.call(performance) + timeOffset;
			};
			
			// 11. Override document.createElement to hide automation iframes
			const originalCreateElement = document.createElement;
			document.createElement = function(tagName) {
				const element = originalCreateElement.call(document, tagName);
				if (tagName.toLowerCase() === 'iframe') {
					const originalSetAttribute = element.setAttribute;
					element.setAttribute = function(name, value) {
						if (name === 'src' && typeof value === 'string') {
							// Hide automation-related iframe sources
							if (value.includes('devtools') || 
								value.includes('chrome-extension') ||
								value.includes('moz-extension')) {
								return;
							}
						}
						return originalSetAttribute.call(element, name, value);
					};
				}
				return element;
			};
			
			// 12. Hide automation in user agent
			if (navigator.userAgent.includes('HeadlessChrome')) {
				Object.defineProperty(navigator, 'userAgent', {
					get: () => navigator.userAgent.replace('HeadlessChrome', 'Chrome'),
					configurable: true
				});
			}
			
			// 13. Fix vendor and product properties
			Object.defineProperty(navigator, 'vendor', {
				get: () => 'Google Inc.',
				configurable: true
			});
			
			Object.defineProperty(navigator, 'product', {
				get: () => 'Gecko',
				configurable: true
			});
			
			// 14. Add realistic hardwareConcurrency
			if (!navigator.hardwareConcurrency || navigator.hardwareConcurrency === 1) {
				Object.defineProperty(navigator, 'hardwareConcurrency', {
					get: () => 4,
					configurable: true
				});
			}
			
			// 15. Fix deviceMemory if it's suspicious
			if (navigator.deviceMemory && navigator.deviceMemory < 2) {
				Object.defineProperty(navigator, 'deviceMemory', {
					get: () => 8,
					configurable: true
				});
			}
			
			// 16. Override Notification.permission if needed
			if (Notification.permission === 'default') {
				Object.defineProperty(Notification, 'permission', {
					get: () => 'denied',
					configurable: true
				});
			}
			
			// 17. Hide automation in window.external
			if (window.external && window.external.toString().includes('Sequentum')) {
				delete window.external;
			}
			
			// 18. Add realistic connection info
			if (navigator.connection) {
				Object.defineProperty(navigator.connection, 'rtt', {
					get: () => 50 + Math.random() * 50,
					configurable: true
				});
			}
			
			// 19. Override toString methods to hide automation
			const originalToString = Function.prototype.toString;
			Function.prototype.toString = function() {
				const result = originalToString.call(this);
				if (result.includes('native code') && 
					(result.includes('puppeteer') || result.includes('chromedp'))) {
					return 'function () { [native code] }';
				}
				return result;
			};
			
			// 20. Fix automation detection in window.chrome
			if (window.chrome && window.chrome.runtime) {
				Object.defineProperty(window.chrome.runtime, 'onConnect', {
					get: () => undefined,
					configurable: true
				});
				Object.defineProperty(window.chrome.runtime, 'onMessage', {
					get: () => undefined,
					configurable: true
				});
			}
			
			// 21. Override navigator.maxTouchPoints
			Object.defineProperty(navigator, 'maxTouchPoints', {
				get: () => 0,
				configurable: true
			});
			
			// 22. Fix automation detection in window.outerHeight/outerWidth (Critical for Cloudflare)
			// This is a common detection vector - headless browsers often have outer dimensions of 0
			if (window.outerHeight === 0 || window.outerWidth === 0) {
				Object.defineProperty(window, 'outerHeight', {
					get: () => window.innerHeight + 120, // Typical browser chrome height
					configurable: true,
					enumerable: true
				});
				Object.defineProperty(window, 'outerWidth', {
					get: () => window.innerWidth + 16, // Typical browser chrome width
					configurable: true,
					enumerable: true
				});
			}
			
			// 23. Override navigator.onLine
			Object.defineProperty(navigator, 'onLine', {
				get: () => true,
				configurable: true
			});
			
			// 24. Fix automation detection in window.screen
			if (window.screen) {
				Object.defineProperty(window.screen, 'availLeft', {
					get: () => 0,
					configurable: true
				});
				Object.defineProperty(window.screen, 'availTop', {
					get: () => 0,
					configurable: true
				});
			}
			
			console.log('🛡️ Advanced stealth mode activated');
		})();
	`
}

// InjectAdvancedStealthScripts injects comprehensive anti-detection scripts
func InjectAdvancedStealthScripts() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		script := GetAdvancedStealthScript()
		return chromedp.Evaluate(script, nil).Do(ctx)
	})
}

// InjectStealthOnNewDocument injects stealth scripts on every new document
func InjectStealthOnNewDocument() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		script := GetAdvancedStealthScript()
		// Use Evaluate for now since AddScriptToEvaluateOnNewDocument might not be available
		return chromedp.Evaluate(script, nil).Do(ctx)
	})
}
