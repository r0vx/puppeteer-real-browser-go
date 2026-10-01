//go:build linux
// +build linux

package browser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// XvfbManager manages Xvfb virtual display sessions
type XvfbManager struct {
	cmd        *exec.Cmd
	exited     chan struct{} // Xvfb 退出且已被 Wait 回收时关闭
	display    string
	displayNum int
	isRunning  bool
	mu         sync.Mutex
}

// NewXvfbManager creates a new Xvfb manager
func NewXvfbManager() *XvfbManager {
	return &XvfbManager{}
}

// Start starts an Xvfb virtual display
func (xm *XvfbManager) Start() error {
	xm.mu.Lock()
	defer xm.mu.Unlock()

	if xm.isRunning {
		return nil
	}

	// Find an available display number
	displayNum, err := xm.findAvailableDisplay()
	if err != nil {
		return fmt.Errorf("failed to find available display: %w", err)
	}

	xm.displayNum = displayNum
	xm.display = fmt.Sprintf(":%d", displayNum)

	// Xvfb arguments matching original Node.js version
	// xvfb_args: ["-screen", "0", "1920x1080x24", "-ac"]
	args := []string{
		xm.display,
		"-screen", "0", "1920x1080x24",
		"-ac", // Disable access control
	}

	xm.cmd = exec.Command("Xvfb", args...)
	// DISPLAY 是进程级共享的，Xvfb 随宿主进程存活；宿主退出（含未调用 Stop）时由内核发 SIGTERM，避免残留。
	// ponytail: Pdeathsig 绑定创建它的 OS 线程，Go 运行时一般不退出线程；若出现残留改为宿主侧显式 Stop
	xm.cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}

	// Redirect stderr to /dev/null for silent mode
	xm.cmd.Stderr = nil
	xm.cmd.Stdout = nil

	if err := xm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Xvfb: %w. Install with: sudo apt-get install xvfb", err)
	}

	// 后台 Wait 回收进程，exited 关闭即表示已退出。
	// 不能用 signal 0 判活：未回收的僵尸进程同样收信号成功
	exited := make(chan struct{})
	go func(cmd *exec.Cmd) {
		cmd.Wait()
		close(exited)
	}(xm.cmd)
	xm.exited = exited

	// 启动窗口内退出（显示号被占用、参数错误等）视为失败
	select {
	case <-exited:
		return fmt.Errorf("Xvfb failed to start properly: exited during startup")
	case <-time.After(500 * time.Millisecond):
	}

	// Set DISPLAY environment variable
	os.Setenv("DISPLAY", xm.display)

	xm.isRunning = true
	return nil
}

// Stop stops the Xvfb virtual display
func (xm *XvfbManager) Stop() error {
	xm.mu.Lock()
	defer xm.mu.Unlock()

	if !xm.isRunning {
		return nil
	}

	if xm.cmd != nil && xm.cmd.Process != nil {
		if err := xm.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("failed to stop Xvfb: %w", err)
		}
		<-xm.exited
	}

	// Clean up lock file
	lockFile := fmt.Sprintf("/tmp/.X%d-lock", xm.displayNum)
	os.Remove(lockFile)

	xm.isRunning = false
	xm.cmd = nil
	return nil
}

// IsRunning returns whether Xvfb is running
func (xm *XvfbManager) IsRunning() bool {
	xm.mu.Lock()
	defer xm.mu.Unlock()
	return xm.isRunning
}

// GetDisplay returns the display string (e.g., ":99")
func (xm *XvfbManager) GetDisplay() string {
	return xm.display
}

// findAvailableDisplay finds an available display number
func (xm *XvfbManager) findAvailableDisplay() (int, error) {
	// Start from display 99 and work up (common convention)
	for displayNum := 99; displayNum < 200; displayNum++ {
		lockFile := fmt.Sprintf("/tmp/.X%d-lock", displayNum)
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			return displayNum, nil
		}
	}
	return 0, fmt.Errorf("no available display number found")
}

// IsXvfbInstalled checks if Xvfb is installed on the system
func IsXvfbInstalled() bool {
	_, err := exec.LookPath("Xvfb")
	return err == nil
}

// GetXvfbInstallCommand returns the install command for the current platform
func GetXvfbInstallCommand() string {
	// Check for apt (Debian/Ubuntu)
	if _, err := exec.LookPath("apt-get"); err == nil {
		return "sudo apt-get install xvfb"
	}
	// Check for yum (RHEL/CentOS)
	if _, err := exec.LookPath("yum"); err == nil {
		return "sudo yum install xorg-x11-server-Xvfb"
	}
	// Check for dnf (Fedora)
	if _, err := exec.LookPath("dnf"); err == nil {
		return "sudo dnf install xorg-x11-server-Xvfb"
	}
	// Check for pacman (Arch)
	if _, err := exec.LookPath("pacman"); err == nil {
		return "sudo pacman -S xorg-server-xvfb"
	}
	return "Please install Xvfb for your distribution"
}

// GetXvfbWarningMessage returns a warning message when Xvfb is not available
func GetXvfbWarningMessage() string {
	return fmt.Sprintf(
		"You are running on Linux but do not have Xvfb installed. "+
			"The browser can be captured. Please install it with:\n\n%s\n",
		GetXvfbInstallCommand(),
	)
}

// RunWithXvfb runs a function with Xvfb display set up
func RunWithXvfb(fn func() error) error {
	xvfb := NewXvfbManager()

	if err := xvfb.Start(); err != nil {
		// Print warning but continue without Xvfb
		fmt.Println(GetXvfbWarningMessage())
		return fn()
	}
	defer xvfb.Stop()

	return fn()
}

// parseDisplay parses a display string to get the display number
func parseDisplay(display string) (int, error) {
	display = strings.TrimPrefix(display, ":")
	parts := strings.Split(display, ".")
	if len(parts) > 0 {
		return strconv.Atoi(parts[0])
	}
	return 0, fmt.Errorf("invalid display format: %s", display)
}
