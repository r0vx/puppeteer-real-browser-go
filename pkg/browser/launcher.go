package browser

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/internal/config"
	"github.com/HNRow/puppeteer-real-browser-go/internal/utils"
)

// ChromeLauncher handles Chrome process launching
type ChromeLauncher struct{}

// NewChromeLauncher creates a new ChromeLauncher
func NewChromeLauncher() *ChromeLauncher {
	return &ChromeLauncher{}
}

// Launch starts a Chrome process with the given options
func (cl *ChromeLauncher) Launch(ctx context.Context, opts *ConnectOptions) (*ChromeProcess, error) {
	// Find Chrome executable
	chromePath, err := cl.findChromeExecutable(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find Chrome executable: %w", err)
	}

	// Find available port
	port, err := utils.FindFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find free port: %w", err)
	}

	// Build Chrome flags
	flags, err := cl.buildChromeFlags(opts, port)
	if err != nil {
		return nil, fmt.Errorf("failed to build Chrome flags: %w", err)
	}

	// DEBUG: 打印实际的Chrome启动参数 (可选)
	// fmt.Printf("🔧 Chrome启动路径: %s\n", chromePath)
	// fmt.Printf("🔧 Chrome启动参数:\n")
	// for i, flag := range flags {
	//	fmt.Printf("  [%d] %s\n", i, flag)
	// }

	// Create Chrome command
	cmd := exec.CommandContext(ctx, chromePath, flags...)

	// Set process group for proper cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start Chrome process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	chrome := &ChromeProcess{
		Cmd:   cmd,
		Port:  port,
		PID:   cmd.Process.Pid,
		Flags: flags,
	}

	// Wait for Chrome to be ready
	if err := cl.waitForChromeReady(ctx, port); err != nil {
		chrome.Kill()
		return nil, fmt.Errorf("Chrome failed to start properly: %w", err)
	}


	return chrome, nil
}

// findChromeExecutable finds the Chrome executable path
func (cl *ChromeLauncher) findChromeExecutable(opts *ConnectOptions) (string, error) {
	// Check if custom Chrome path is specified
	if opts.CustomConfig != nil {
		if chromePath, ok := opts.CustomConfig["chromePath"].(string); ok && chromePath != "" {
			if _, err := os.Stat(chromePath); err == nil {
				return chromePath, nil
			}
		}
	}

	// Use default Chrome path detection
	return utils.GetChromeExecutablePath()
}

// buildChromeFlags constructs the Chrome command line flags
func (cl *ChromeLauncher) buildChromeFlags(opts *ConnectOptions, port int) ([]string, error) {
	var flags []string

	if opts.IgnoreAllFlags {
		// Use minimal flags when ignoring defaults
		flags = append(flags, fmt.Sprintf("--remote-debugging-port=%d", port))
		flags = append(flags, opts.Args...)

		// Add headless flags if needed
		headlessFlags := config.GetHeadlessFlags(opts.Headless)
		flags = append(flags, headlessFlags...)

		// Add proxy flags if configured
		if opts.Proxy != nil {
			proxyFlags := config.GetProxyFlags(opts.Proxy.Host, opts.Proxy.Port)
			flags = append(flags, proxyFlags...)
		}

		// Add extension flags if configured
		extensions := opts.Extensions
		
		// 如果启用自动加载默认扩展，添加到临时加载列表
		if opts.AutoLoadDefaultExtensions {
			defaultExtensions := config.GetDefaultExtensionPaths()
			extensions = append(extensions, defaultExtensions...)
		}
		
		// 如果启用了自动加载默认扩展，需要确保添加--enable-extensions
		var extensionFlags []string
		if opts.AutoLoadDefaultExtensions || len(extensions) > 0 {
			extensionFlags = append(extensionFlags, "--enable-extensions")
			// 只有当有临时扩展路径时才添加--load-extension
			if len(extensions) > 0 {
				additionalFlags := config.GetExtensionFlags(extensions)
				extensionFlags = append(extensionFlags, additionalFlags...)
			}
		} else {
			extensionFlags = config.GetExtensionFlags(extensions)
		}
		flags = append(flags, extensionFlags...)
	} else {
		// Start with default flags
		defaultFlags := config.DefaultChromeFlags()

		// Add stealth flags to avoid detection
		stealthFlags := config.GetStealthFlags()

		// Modify disable-features flag to include AutomationControlled
		for i, flag := range defaultFlags {
			if strings.HasPrefix(flag, "--disable-features=") {
				defaultFlags[i] = flag + ",AutomationControlled"
				break
			}
		}

		// Remove disable-component-update flag (similar to original JS)
		filteredFlags := make([]string, 0, len(defaultFlags))
		for _, flag := range defaultFlags {
			if !strings.HasPrefix(flag, "--disable-component-update") {
				filteredFlags = append(filteredFlags, flag)
			}
		}

		// Add remote debugging port
		filteredFlags = append(filteredFlags, fmt.Sprintf("--remote-debugging-port=%d", port))

		// Add user data directory
		userDataDir, err := cl.getUserDataDir(opts)
		if err != nil {
			return nil, err
		}
		filteredFlags = append(filteredFlags, "--user-data-dir="+userDataDir)

		// 处理扩展
		extensions := opts.Extensions
		
		// 如果启用自动加载默认扩展，添加到临时加载列表
		if opts.AutoLoadDefaultExtensions {
			defaultExtensions := config.GetDefaultExtensionPaths()
			extensions = append(extensions, defaultExtensions...)
		}
		
		// 处理扩展标志
		var extensionFlags []string
		if opts.AutoLoadDefaultExtensions || len(extensions) > 0 {
			extensionFlags = append(extensionFlags, "--enable-extensions")
			// 只有当有临时扩展路径时才添加--load-extension
			if len(extensions) > 0 {
				additionalFlags := config.GetExtensionFlags(extensions)
				extensionFlags = append(extensionFlags, additionalFlags...)
			}
		} else {
			extensionFlags = config.GetExtensionFlags(extensions)
		}
		
		// Merge all flags
		flags = config.MergeFlags(
			filteredFlags,
			stealthFlags,
			opts.Args,
			config.GetHeadlessFlags(opts.Headless),
			extensionFlags,
		)

		// Add proxy flags if configured
		if opts.Proxy != nil {
			proxyFlags := config.GetProxyFlags(opts.Proxy.Host, opts.Proxy.Port)
			flags = append(flags, proxyFlags...)
		}
	}

	return flags, nil
}

// getUserDataDir gets or creates user data directory
func (cl *ChromeLauncher) getUserDataDir(opts *ConnectOptions) (string, error) {
	// 1. 优先使用自定义配置
	if opts.CustomConfig != nil {
		if userDataDir, ok := opts.CustomConfig["userDataDir"].(string); ok && userDataDir != "" {
			return userDataDir, nil
		}
	}

	// 2. 如果启用了持久化配置，使用持久化目录
	if opts.PersistProfile && opts.ProfileName != "" {
		return utils.GetPersistentUserDataDir(opts.ProfileName)
	}

	// 3. 默认使用临时目录
	return utils.GetUserDataDir()
}

// waitForChromeReady waits for Chrome to be ready for connections
func (cl *ChromeLauncher) waitForChromeReady(ctx context.Context, port int) error {
	timeout := 30 * time.Second
	interval := 500 * time.Millisecond

	return utils.WaitWithTimeout(func() bool {
		// Try to connect to the debug port to see if Chrome is ready
		return cl.isDebugPortReady(port)
	}, timeout, interval)
}

// isDebugPortReady checks if Chrome's debug port is ready
func (cl *ChromeLauncher) isDebugPortReady(port int) bool {
	// Try to make a simple HTTP request to the debug endpoint
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/json/version", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Kill terminates the Chrome process
func (cp *ChromeProcess) Kill() error {
	if cp.Cmd == nil || cp.Cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first
	if err := cp.Cmd.Process.Signal(syscall.SIGTERM); err == nil {
		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- cp.Cmd.Wait()
		}()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			// Graceful shutdown timeout, force kill
		}
	}

	// Force kill the process and its children
	if err := cp.killProcessTree(); err != nil {
		return fmt.Errorf("failed to kill Chrome process tree: %w", err)
	}

	return nil
}

// killProcessTree kills the process and all its children
func (cp *ChromeProcess) killProcessTree() error {
	if cp.PID == 0 {
		return nil
	}

	// Kill the main process using syscall
	if err := syscall.Kill(cp.PID, syscall.SIGKILL); err != nil {
		return err
	}

	return nil
}

// IsRunning checks if the Chrome process is still running
func (cp *ChromeProcess) IsRunning() bool {
	if cp.Cmd == nil || cp.Cmd.Process == nil {
		return false
	}

	// Check if process is still running by sending signal 0
	err := syscall.Kill(cp.PID, 0)
	return err == nil
}
