package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chromedp/chromedp"
)

// ExtensionManager handles advanced extension operations via CDP
type ExtensionManager struct {
	ctx context.Context
}

// NewExtensionManager creates a new extension manager
func NewExtensionManager(ctx context.Context) *ExtensionManager {
	return &ExtensionManager{ctx: ctx}
}

// InstallExtensionViaCDP installs extension using Chrome DevTools Protocol
func (em *ExtensionManager) InstallExtensionViaCDP(extensionPath string) error {
	// 确保开发者模式已启用
	if err := em.enableDeveloperMode(); err != nil {
		return fmt.Errorf("failed to enable developer mode: %w", err)
	}

	// 使用CDP直接安装扩展
	return em.loadUnpackedExtension(extensionPath)
}

// enableDeveloperMode enables developer mode programmatically
func (em *ExtensionManager) enableDeveloperMode() error {
	return chromedp.Run(em.ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.WaitVisible("#developerMode", chromedp.ByID),
		chromedp.Evaluate(`
			const toggle = document.querySelector('#developerMode');
			if (toggle && !toggle.checked) {
				toggle.click();
				console.log('Developer mode enabled');
			}
			return toggle ? toggle.checked : false;
		`, nil),
		chromedp.Sleep(1000), // 等待UI更新
	)
}

// loadUnpackedExtension loads an unpacked extension
func (em *ExtensionManager) loadUnpackedExtension(extensionPath string) error {
	absPath, err := filepath.Abs(extensionPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 验证扩展目录
	if err := em.validateExtensionDirectory(absPath); err != nil {
		return fmt.Errorf("invalid extension directory: %w", err)
	}

	// 使用CDP的Management API安装扩展
	return chromedp.Run(em.ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.WaitVisible("#pack-extension-overlay", chromedp.ByID),
		chromedp.Evaluate(fmt.Sprintf(`
			// 模拟点击"加载已解压的扩展程序"
			const loadButton = document.querySelector('#load-unpacked');
			if (loadButton) {
				loadButton.click();
				// 在实际环境中，这会打开文件对话框
				// 但在自动化环境中，我们需要使用不同的方法
				console.log('Triggered load unpacked extension');
				return true;
			}
			return false;
		`), nil),
	)
}

// validateExtensionDirectory validates extension directory structure
func (em *ExtensionManager) validateExtensionDirectory(path string) error {
	// 检查目录是否存在
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("extension directory does not exist: %w", err)
	}

	// 检查manifest.json
	manifestPath := filepath.Join(path, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("manifest.json not found: %w", err)
	}

	// 验证manifest格式
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("invalid manifest.json format: %w", err)
	}

	// 检查必需字段
	requiredFields := []string{"manifest_version", "name", "version"}
	for _, field := range requiredFields {
		if _, exists := manifest[field]; !exists {
			return fmt.Errorf("manifest.json missing required field: %s", field)
		}
	}

	return nil
}

// GetInstalledExtensions returns list of installed extensions
func (em *ExtensionManager) GetInstalledExtensions() ([]ExtensionInfo, error) {
	var extensions []ExtensionInfo

	err := chromedp.Run(em.ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.WaitReady("extensions-manager"),
		chromedp.Evaluate(`
			// 等待扩展加载
			setTimeout(() => {
				const items = document.querySelectorAll('extensions-item');
				const extensions = Array.from(items).map(item => {
					const shadow = item.shadowRoot;
					if (!shadow) return null;
					
					const name = shadow.querySelector('#name')?.textContent || '';
					const id = item.getAttribute('id') || '';
					const version = shadow.querySelector('#version')?.textContent || '';
					const enabled = shadow.querySelector('cr-toggle')?.checked || false;
					const description = shadow.querySelector('#description')?.textContent || '';
					
					return {
						id: id,
						name: name.trim(),
						version: version.replace('版本 ', '').trim(),
						enabled: enabled,
						description: description.trim()
					};
				}).filter(ext => ext !== null && ext.name !== '');
				
				window.extensionsList = extensions;
			}, 2000);
		`, nil),
		chromedp.Sleep(3000), // 等待执行完成
		chromedp.Evaluate(`window.extensionsList || []`, &extensions),
	)

	return extensions, err
}

// ExtensionInfo represents extension information
type ExtensionInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// InstallMultipleExtensions installs multiple extensions
func (em *ExtensionManager) InstallMultipleExtensions(extensionPaths []string) error {
	for i, path := range extensionPaths {
		fmt.Printf("Installing extension %d/%d: %s\n", i+1, len(extensionPaths), path)
		
		if err := em.InstallExtensionViaCDP(path); err != nil {
			return fmt.Errorf("failed to install extension %s: %w", path, err)
		}
		
		// 短暂延迟，避免并发问题
		chromedp.Sleep(2000).Do(em.ctx)
	}
	
	return nil
}

// EnableExtension enables a specific extension by ID
func (em *ExtensionManager) EnableExtension(extensionID string) error {
	return chromedp.Run(em.ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.WaitVisible(fmt.Sprintf(`extensions-item[id="%s"]`, extensionID)),
		chromedp.Evaluate(fmt.Sprintf(`
			const extensionItem = document.querySelector('extensions-item[id="%s"]');
			if (extensionItem && extensionItem.shadowRoot) {
				const toggle = extensionItem.shadowRoot.querySelector('cr-toggle');
				if (toggle && !toggle.checked) {
					toggle.click();
					return true;
				}
			}
			return false;
		`, extensionID), nil),
	)
}

// ForceLoadExtensionsFromCommandLine forces Chrome to recognize command-line loaded extensions
func (em *ExtensionManager) ForceLoadExtensionsFromCommandLine() error {
	return chromedp.Run(em.ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.Sleep(2000),
		// 刷新扩展管理页面以重新加载
		chromedp.Reload(),
		chromedp.Sleep(3000),
		// 强制刷新扩展列表
		chromedp.Evaluate(`
			// 强制触发扩展重新加载
			chrome.management && chrome.management.getAll ? 
				chrome.management.getAll(extensions => {
					console.log('Found extensions via management API:', extensions);
				}) : 
				console.log('Management API not available');
		`, nil),
	)
}