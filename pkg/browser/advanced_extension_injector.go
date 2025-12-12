package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chromedp/chromedp"
)

// AdvancedExtensionInjector 使用CDP直接注入扩展到Chrome运行时
type AdvancedExtensionInjector struct {
	ctx context.Context
}

// NewAdvancedExtensionInjector 创建高级扩展注入器
func NewAdvancedExtensionInjector(ctx context.Context) *AdvancedExtensionInjector {
	return &AdvancedExtensionInjector{ctx: ctx}
}

// InjectExtensionRuntime 通过CDP直接将扩展注入到页面运行时
func (aei *AdvancedExtensionInjector) InjectExtensionRuntime(extensionPath string) error {
	// 1. 读取扩展的manifest.json
	manifest, err := aei.readManifest(extensionPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// 2. 如果有content_scripts，直接注入到页面
	if contentScripts, ok := manifest["content_scripts"].([]interface{}); ok {
		for _, scriptConfig := range contentScripts {
			if err := aei.injectContentScript(extensionPath, scriptConfig.(map[string]interface{})); err != nil {
				return fmt.Errorf("failed to inject content script: %w", err)
			}
		}
	}

	// 3. 如果有background script，创建虚拟background context
	if background, ok := manifest["background"].(map[string]interface{}); ok {
		if err := aei.createVirtualBackground(extensionPath, background); err != nil {
			return fmt.Errorf("failed to create background: %w", err)
		}
	}

	// 4. 注入Chrome APIs模拟
	return aei.InjectChromeAPIs(manifest)
}

// injectContentScript 直接将content script注入到页面
func (aei *AdvancedExtensionInjector) injectContentScript(extensionPath string, scriptConfig map[string]interface{}) error {
	// 获取要注入的脚本文件
	jsFiles, ok := scriptConfig["js"].([]interface{})
	if !ok {
		return nil
	}

	// 获取匹配模式
	matches, _ := scriptConfig["matches"].([]interface{})

	for _, jsFile := range jsFiles {
		scriptPath := filepath.Join(extensionPath, jsFile.(string))
		scriptContent, err := os.ReadFile(scriptPath)
		if err != nil {
			continue
		}

		// 将脚本包装在一个隔离的执行环境中
		wrappedScript := aei.wrapScriptInIsolatedEnvironment(string(scriptContent), matches)

		// 通过chromedp注入脚本
		err = chromedp.Run(aei.ctx, chromedp.Evaluate(wrappedScript, nil))
		if err != nil {
			return fmt.Errorf("failed to inject script: %w", err)
		}
	}

	return nil
}

// wrapScriptInIsolatedEnvironment 将扩展脚本包装在隔离环境中
func (aei *AdvancedExtensionInjector) wrapScriptInIsolatedEnvironment(script string, matches []interface{}) string {
	// 创建一个独立的执行作用域，模拟扩展的isolated world
	wrapper := fmt.Sprintf(`
		(function() {
			// 创建扩展的隔离作用域
			const extensionScope = {
				chrome: window.chrome || {},
				console: console,
				document: document,
				window: window
			};
			
			// 注入Chrome扩展API
			extensionScope.chrome.runtime = extensionScope.chrome.runtime || {
				id: 'injected-extension-' + Math.random().toString(36).substr(2, 9),
				sendMessage: function(message, callback) {
					console.log('Extension message:', message);
					if (callback) callback({success: true});
				},
				onMessage: {
					addListener: function(callback) {
						window.addEventListener('extension-message', callback);
					}
				}
			};
			
			extensionScope.chrome.storage = extensionScope.chrome.storage || {
				local: {
					get: function(keys, callback) {
						const stored = JSON.parse(localStorage.getItem('extension-storage') || '{}');
						callback(stored);
					},
					set: function(items, callback) {
						const stored = JSON.parse(localStorage.getItem('extension-storage') || '{}');
						Object.assign(stored, items);
						localStorage.setItem('extension-storage', JSON.stringify(stored));
						if (callback) callback();
					}
				}
			};
			
			// 在扩展作用域中执行脚本
			with (extensionScope) {
				try {
					%s
				} catch (e) {
					console.error('Extension script error:', e);
				}
			}
		})();
	`, script)

	return wrapper
}

// createVirtualBackground 创建虚拟的background script环境
func (aei *AdvancedExtensionInjector) createVirtualBackground(extensionPath string, background map[string]interface{}) error {
	var scriptPath string

	// 支持不同的background配置格式
	if serviceWorker, ok := background["service_worker"].(string); ok {
		scriptPath = filepath.Join(extensionPath, serviceWorker)
	} else if scripts, ok := background["scripts"].([]interface{}); ok && len(scripts) > 0 {
		scriptPath = filepath.Join(extensionPath, scripts[0].(string))
	}

	if scriptPath == "" {
		return nil
	}

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}

	// 创建一个持久的background执行环境
	backgroundWrapper := fmt.Sprintf(`
		// 创建background script的全局环境
		if (!window.extensionBackground) {
			window.extensionBackground = {
				chrome: {
					runtime: {
						onInstalled: { addListener: function(cb) { setTimeout(cb, 100); }},
						onStartup: { addListener: function(cb) { setTimeout(cb, 100); }},
						onMessage: { 
							addListener: function(cb) {
								window.addEventListener('extension-background-message', cb);
							}
						}
					},
					tabs: {
						query: function(queryInfo, callback) {
							callback([{id: 1, url: location.href, active: true}]);
						},
						sendMessage: function(tabId, message, callback) {
							window.dispatchEvent(new CustomEvent('extension-message', {detail: message}));
							if (callback) callback({success: true});
						}
					},
					storage: {
						local: {
							get: function(keys, callback) {
								const stored = JSON.parse(localStorage.getItem('extension-background-storage') || '{}');
								callback(stored);
							},
							set: function(items, callback) {
								const stored = JSON.parse(localStorage.getItem('extension-background-storage') || '{}');
								Object.assign(stored, items);
								localStorage.setItem('extension-background-storage', JSON.stringify(stored));
								if (callback) callback();
							}
						}
					}
				}
			};
			
			// 在background环境中执行脚本
			with (window.extensionBackground) {
				try {
					%s
				} catch (e) {
					console.error('Background script error:', e);
				}
			}
		}
	`, string(scriptContent))

	return chromedp.Run(aei.ctx, chromedp.Evaluate(backgroundWrapper, nil))
}

// InjectChromeAPIs 注入完整的Chrome扩展API模拟
func (aei *AdvancedExtensionInjector) InjectChromeAPIs(manifest map[string]interface{}) error {
	// 获取扩展权限
	permissions := []string{}
	if perms, ok := manifest["permissions"].([]interface{}); ok {
		for _, perm := range perms {
			permissions = append(permissions, perm.(string))
		}
	}

	// 根据权限动态构建Chrome API
	apiScript := aei.buildChromeAPIScript(permissions, manifest)

	return chromedp.Run(aei.ctx, chromedp.Evaluate(apiScript, nil))
}

// buildChromeAPIScript 根据权限构建Chrome API脚本
func (aei *AdvancedExtensionInjector) buildChromeAPIScript(permissions []string, manifest map[string]interface{}) string {
	extensionId := aei.generateExtensionId(manifest)

	apiScript := fmt.Sprintf(`
		// 注入完整的Chrome扩展API
		if (!window.chrome) window.chrome = {};
		
		// Extension ID
		window.chrome.runtime = window.chrome.runtime || {};
		window.chrome.runtime.id = '%s';
		
		// Storage API
		if (%v) {
			window.chrome.storage = {
				local: {
					get: function(keys, callback) {
						const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
						callback(stored);
					},
					set: function(items, callback) {
						const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
						Object.assign(stored, items);
						localStorage.setItem('chrome-extension-storage', JSON.stringify(stored));
						if (callback) callback();
					},
					remove: function(keys, callback) {
						const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
						if (typeof keys === 'string') keys = [keys];
						keys.forEach(key => delete stored[key]);
						localStorage.setItem('chrome-extension-storage', JSON.stringify(stored));
						if (callback) callback();
					}
				},
				sync: {
					get: function(keys, callback) {
						const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
						callback(stored);
					},
					set: function(items, callback) {
						const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
						Object.assign(stored, items);
						localStorage.setItem('chrome-extension-storage', JSON.stringify(stored));
						if (callback) callback();
					}
				}
			};
		}
		
		// Tabs API
		if (%v) {
			window.chrome.tabs = {
				query: function(queryInfo, callback) {
					callback([{
						id: 1,
						url: location.href,
						title: document.title,
						active: true,
						windowId: 1
					}]);
				},
				sendMessage: function(tabId, message, options, callback) {
					if (typeof options === 'function') {
						callback = options;
					}
					setTimeout(() => callback({success: true}), 10);
				},
				create: function(createProperties, callback) {
					const newTab = {
						id: Math.floor(Math.random() * 1000),
						url: createProperties.url,
						active: createProperties.active !== false
					};
					if (callback) callback(newTab);
				}
			};
		}
		
		// Runtime messaging
		window.chrome.runtime.sendMessage = function(message, callback) {
			console.log('Extension runtime message:', message);
			if (callback) setTimeout(() => callback({received: true}), 10);
		};
		
		window.chrome.runtime.onMessage = {
			addListener: function(callback) {
				window.addEventListener('chrome-extension-message', function(event) {
					callback(event.detail.message, event.detail.sender, event.detail.sendResponse);
				});
			}
		};
		
		// 标记扩展已注入
		window.chrome.runtime.injected = true;
		console.log('Chrome Extension APIs injected successfully for extension:', '%s');
	`,
		extensionId,
		aei.hasPermission(permissions, "storage"),
		aei.hasPermission(permissions, "tabs"),
		extensionId,
	)

	return apiScript
}

// hasPermission 检查是否有特定权限
func (aei *AdvancedExtensionInjector) hasPermission(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// generateExtensionId 生成扩展ID
func (aei *AdvancedExtensionInjector) generateExtensionId(manifest map[string]interface{}) string {
	name, _ := manifest["name"].(string)
	version, _ := manifest["version"].(string)
	return fmt.Sprintf("injected-%s-%s", strings.ToLower(strings.ReplaceAll(name, " ", "-")), version)
}

// readManifest 读取manifest.json
func (aei *AdvancedExtensionInjector) readManifest(extensionPath string) (map[string]interface{}, error) {
	manifestPath := filepath.Join(extensionPath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// InjectAllExtensions 批量注入多个扩展
func (aei *AdvancedExtensionInjector) InjectAllExtensions(extensionPaths []string) error {
	for i, path := range extensionPaths {
		fmt.Printf("Injecting extension %d/%d: %s\n", i+1, len(extensionPaths), filepath.Base(path))

		if err := aei.InjectExtensionRuntime(path); err != nil {
			fmt.Printf("Warning: Failed to inject %s: %v\n", path, err)
			continue
		}

		fmt.Printf("✅ Successfully injected: %s\n", filepath.Base(path))
	}

	return nil
}

// VerifyInjection 验证注入是否成功
func (aei *AdvancedExtensionInjector) VerifyInjection() (bool, error) {
	var result bool
	err := chromedp.Run(aei.ctx,
		chromedp.Evaluate(`
			// 检查Chrome API是否已注入
			!!(window.chrome && 
			   window.chrome.runtime && 
			   window.chrome.runtime.injected && 
			   window.chrome.storage)
		`, &result),
	)

	return result, err
}
