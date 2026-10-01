package utils

import (
	"sync"
	"testing"
)

// TestGetUserDataDirUnique 并发创建的临时 profile 目录必须互不相同：
// 两个 Chrome 共用 profile 时后启动的会交给前一个进程并退出，调试端口永远等不到
func TestGetUserDataDirUnique(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	const n = 64
	dirs := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			d, err := GetUserDataDir()
			if err != nil {
				t.Error(err)
				return
			}
			dirs[i] = d
		})
	}
	wg.Wait()

	seen := make(map[string]bool, n)
	for _, d := range dirs {
		if seen[d] {
			t.Errorf("duplicate user data dir %s", d)
		}
		seen[d] = true
	}
}
