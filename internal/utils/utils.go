package utils

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"
)

// GetRandomInt returns a random integer between min and max (inclusive)
func GetRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min+1) + min
}

// FindFreePort finds an available port on the system
func FindFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetChromeExecutablePath returns the path to Chrome executable
func GetChromeExecutablePath() (string, error) {
	var paths []string
	
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	case "windows":
		paths = []string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Users\\" + os.Getenv("USERNAME") + "\\AppData\\Local\\Google\\Chrome\\Application\\chrome.exe",
		}
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	return "", fmt.Errorf("Chrome executable not found")
}

// WaitWithTimeout waits for a condition with timeout
func WaitWithTimeout(condition func() bool, timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(interval)
	}
	
	return fmt.Errorf("timeout waiting for condition")
}

// IsLinux checks if the current OS is Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsMacOS checks if the current OS is macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsWindows checks if the current OS is Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// GetUserDataDir returns a suitable user data directory for Chrome
func GetUserDataDir() (string, error) {
	tmpDir := os.TempDir()
	userDataDir := fmt.Sprintf("%s/puppeteer-real-browser-go-%d", tmpDir, time.Now().UnixNano())
	
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create user data directory: %w", err)
	}
	
	return userDataDir, nil
}
