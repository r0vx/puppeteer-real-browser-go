package browser

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// AccountManager manages multiple isolated browser instances for different accounts
type AccountManager struct {
	accounts    map[string]*AccountInstance
	baseOptions *ConnectOptions
	mutex       sync.RWMutex
}

// AccountInstance represents a browser instance for a specific account
type AccountInstance struct {
	ProfileName string
	Options     *ConnectOptions
	Instance    *BrowserInstance
	UserDataDir string
}

// NewAccountManager creates a new account manager
func NewAccountManager(baseOptions *ConnectOptions) *AccountManager {
	if baseOptions == nil {
		baseOptions = &ConnectOptions{
			Headless:       false,
			UseCustomCDP:   true,
			PersistProfile: true,
		}
	}

	return &AccountManager{
		accounts:    make(map[string]*AccountInstance),
		baseOptions: baseOptions,
	}
}

// CreateAccount creates a new isolated browser instance for an account
func (am *AccountManager) CreateAccount(ctx context.Context, profileName string, options *ConnectOptions) (*AccountInstance, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if account already exists
	if existing, exists := am.accounts[profileName]; exists {
		return existing, nil
	}

	// Merge base options with account-specific options
	accountOptions := am.mergeOptions(am.baseOptions, options)
	accountOptions.ProfileName = profileName
	accountOptions.PersistProfile = true

	// 注释掉预安装系统，专注于 --load-extension 方式
	// Pre-install extensions before creating browser instance
	// if err := am.preInstallExtensions(profileName, accountOptions); err != nil {
	//	return nil, fmt.Errorf("failed to pre-install extensions for account %s: %w", profileName, err)
	// }

	// Create isolated browser instance for this account
	instance, err := Connect(ctx, accountOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser instance for account %s: %w", profileName, err)
	}

	accountInstance := &AccountInstance{
		ProfileName: profileName,
		Options:     accountOptions,
		Instance:    instance,
		UserDataDir: "", // Will be set by the launcher
	}

	am.accounts[profileName] = accountInstance
	return accountInstance, nil
}

// GetAccount retrieves an existing account instance
func (am *AccountManager) GetAccount(profileName string) (*AccountInstance, bool) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	account, exists := am.accounts[profileName]
	return account, exists
}

// ListAccounts returns all account names
func (am *AccountManager) ListAccounts() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var names []string
	for name := range am.accounts {
		names = append(names, name)
	}
	return names
}

// RemoveAccount removes an account and closes its browser instance
func (am *AccountManager) RemoveAccount(profileName string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[profileName]
	if !exists {
		return fmt.Errorf("account %s not found", profileName)
	}

	// Close the browser instance
	if account.Instance != nil {
		if err := account.Instance.Close(); err != nil {
			return fmt.Errorf("failed to close browser instance for account %s: %w", profileName, err)
		}
	}

	delete(am.accounts, profileName)
	return nil
}

// CloseAll closes all account instances
func (am *AccountManager) CloseAll() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	var errors []error
	for profileName, account := range am.accounts {
		if account.Instance != nil {
			if err := account.Instance.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close account %s: %w", profileName, err))
			}
		}
	}

	// Clear all accounts
	am.accounts = make(map[string]*AccountInstance)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing accounts: %v", errors)
	}
	return nil
}

// GetAccountCount returns the number of active accounts
func (am *AccountManager) GetAccountCount() int {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return len(am.accounts)
}

// mergeOptions merges base options with account-specific options
func (am *AccountManager) mergeOptions(base *ConnectOptions, account *ConnectOptions) *ConnectOptions {
	// Start with a copy of base options
	merged := &ConnectOptions{
		Headless:       base.Headless,
		Args:           make([]string, len(base.Args)),
		CustomConfig:   make(map[string]interface{}),
		Proxy:          base.Proxy,
		Turnstile:      base.Turnstile,
		ConnectOption:  base.ConnectOption,
		DisableXvfb:    base.DisableXvfb,
		IgnoreAllFlags: base.IgnoreAllFlags,
		Plugins:        base.Plugins,
		UseCustomCDP:   base.UseCustomCDP,
		Extensions:     make([]string, len(base.Extensions)),
		PersistProfile: base.PersistProfile,
	}

	// Copy slices and maps
	copy(merged.Args, base.Args)
	copy(merged.Extensions, base.Extensions)
	for k, v := range base.CustomConfig {
		merged.CustomConfig[k] = v
	}

	// Override with account-specific options if provided
	if account != nil {
		if account.Headless != nil {
			merged.Headless = account.Headless
		}
		if len(account.Args) > 0 {
			merged.Args = append(merged.Args, account.Args...)
		}
		if account.Proxy != nil {
			merged.Proxy = account.Proxy
		}
		if len(account.Extensions) > 0 {
			merged.Extensions = append(merged.Extensions, account.Extensions...)
		}
		if account.CustomConfig != nil {
			for k, v := range account.CustomConfig {
				merged.CustomConfig[k] = v
			}
		}
		// Override boolean fields if they're explicitly set
		if account.UseCustomCDP {
			merged.UseCustomCDP = account.UseCustomCDP
		}
		if account.Turnstile {
			merged.Turnstile = account.Turnstile
		}
		if account.PersistProfile {
			merged.PersistProfile = account.PersistProfile
		}
	}

	return merged
}

// preInstallExtensions pre-installs extensions to the account's user data directory
func (am *AccountManager) preInstallExtensions(profileName string, options *ConnectOptions) error {
	if len(options.Extensions) == 0 {
		return nil
	}

	// Get the user data directory that will be used
	var userDataDir string

	if options.CustomConfig != nil {
		if dir, ok := options.CustomConfig["userDataDir"].(string); ok && dir != "" {
			userDataDir = dir
		}
	}

	if userDataDir == "" && options.PersistProfile && options.ProfileName != "" {
		// Use the same logic as launcher.go
		if homeDir, err := os.UserHomeDir(); err == nil {
			userDataDir = fmt.Sprintf("%s/.puppeteer-real-browser-go/profiles/%s", homeDir, options.ProfileName)
		} else {
			userDataDir = fmt.Sprintf("%s/puppeteer-real-browser-go/profiles/%s", os.TempDir(), options.ProfileName)
		}
	}

	if userDataDir == "" {
		// Skip pre-installation for temporary directories
		return nil
	}

	// Ensure user data directory exists
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create user data directory: %w", err)
	}

	// Create extension installer
	installer := NewExtensionInstaller(userDataDir)

	// Pre-install extensions
	fmt.Printf("🧩 Pre-installing %d extensions for account '%s'...\n", len(options.Extensions), profileName)

	if err := installer.PreInstallExtensions(options.Extensions); err != nil {
		return fmt.Errorf("failed to pre-install extensions: %w", err)
	}

	// Create extensions preferences
	if err := installer.CreateExtensionsPreferences(options.Extensions); err != nil {
		return fmt.Errorf("failed to create extension preferences: %w", err)
	}

	return nil
}
