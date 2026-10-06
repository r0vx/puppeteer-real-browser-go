//go:build !windows

package browser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// configureChromeCmd 把 Chrome 放进独立进程组以便整组清理。
// ctx 取消（含 BrowserInstance.Close）时由 Launch 设置的 cmd.Cancel 先 SIGTERM 让 Chrome 正常退出
// （经 sigtermOnce 与 Kill 共用，只发一次），5s 内未退出再由 exec 强杀
func configureChromeCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 5 * time.Second
}

// terminateProcess 请求进程正常退出
func terminateProcess(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

// killProcessTree kills the process and all its children.
// 启动时设置了 Setpgid，Chrome 及其子进程同属一个进程组，负 PID 表示整组
func (cp *ChromeProcess) killProcessTree() error {
	if cp.PID == 0 {
		return nil
	}
	if err := syscall.Kill(-cp.PID, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		// 进程组杀失败时退而只杀主进程
		if killErr := cp.Cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}
	return nil
}

// waitProcessTreeExit 等进程组内所有进程退出，超时则整组强杀。
// ponytail: 宿主以 PID 1 运行且不回收孤儿时，僵尸辅助进程会让这里每次等满超时；容器请加 --init
func (cp *ChromeProcess) waitProcessTreeExit(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for syscall.Kill(-cp.PID, 0) == nil { // 进程组内仍有进程
		if time.Now().After(deadline) {
			syscall.Kill(-cp.PID, syscall.SIGKILL)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
