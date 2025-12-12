// +build !linux

package browser

import "fmt"

// XvfbManager is a stub for non-Linux platforms
type XvfbManager struct{}

// NewXvfbManager creates a new Xvfb manager (stub for non-Linux)
func NewXvfbManager() *XvfbManager {
	return &XvfbManager{}
}

// Start is a no-op on non-Linux platforms
func (xm *XvfbManager) Start() error {
	// Xvfb is only needed on Linux
	return nil
}

// Stop is a no-op on non-Linux platforms
func (xm *XvfbManager) Stop() error {
	return nil
}

// IsRunning always returns false on non-Linux platforms
func (xm *XvfbManager) IsRunning() bool {
	return false
}

// GetDisplay returns empty string on non-Linux platforms
func (xm *XvfbManager) GetDisplay() string {
	return ""
}

// IsXvfbInstalled returns false on non-Linux platforms
func IsXvfbInstalled() bool {
	return false
}

// GetXvfbInstallCommand returns empty string on non-Linux platforms
func GetXvfbInstallCommand() string {
	return ""
}

// GetXvfbWarningMessage returns empty message on non-Linux platforms
func GetXvfbWarningMessage() string {
	return ""
}

// RunWithXvfb just runs the function directly on non-Linux platforms
func RunWithXvfb(fn func() error) error {
	return fn()
}

// NeedsXvfb checks if the current platform needs Xvfb
func NeedsXvfb() bool {
	return false
}

// SetupXvfbForBrowser prints info about Xvfb requirement (stub)
func SetupXvfbForBrowser() {
	fmt.Println("Xvfb is not required on this platform")
}

