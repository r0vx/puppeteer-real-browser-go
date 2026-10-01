//go:build windows

package browser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// ponytail: Windows 实现只保证能编译，未在实机验证；需要正常退出（profile 落盘）时应改为走 CDP Browser.close

// configureChromeCmd Windows 没有 SIGTERM，ctx 取消时由 exec 直接结束进程
func configureChromeCmd(cmd *exec.Cmd) {
	cmd.WaitDelay = 5 * time.Second
}

// terminateProcess Windows 无法向无窗口进程发送可捕获的终止信号，返回错误让调用方直接强杀
func terminateProcess(_ *os.Process) error {
	return errors.ErrUnsupported
}

// killProcessTree 用 taskkill /T 结束 Chrome 及其子进程，失败时退而只杀主进程
func (cp *ChromeProcess) killProcessTree() error {
	if cp.PID == 0 {
		return nil
	}
	if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cp.PID)).Run(); err != nil {
		if killErr := cp.Cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}
	return nil
}

// waitProcessTreeExit taskkill /T 已同步结束子进程，这里只留出文件句柄释放的时间
func (cp *ChromeProcess) waitProcessTreeExit(_ time.Duration) {
	time.Sleep(500 * time.Millisecond)
}
