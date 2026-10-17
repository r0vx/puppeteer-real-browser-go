// baseline_server 真机基准采集服务：提供采集页，接收手机上传的结果并存为 JSON 文件。
// 只监听本机地址，由 scripts/collect-baseline.sh 经 cloudflared 临时暴露为 https 地址
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// maxResultBytes 单份结果的大小上限
const maxResultBytes = 1 << 20

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run 解析参数并运行服务，收到中断信号后优雅退出
func run(ctx context.Context, args []string) error {
	fset := flag.NewFlagSet("baseline_server", flag.ContinueOnError)
	port := fset.Int("port", 8765, "监听端口（只监听 127.0.0.1）")
	out := fset.String("out", "test/baseline", "结果保存目录")
	page := fset.String("page", "test/baseline/collector.html", "采集页路径")
	if err := fset.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", *port),
		Handler:           newServer(*page, *out).routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Printf("采集服务已启动：http://%s ，结果保存到 %s", srv.Addr, *out)
	select {
	case err := <-errc:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// server 采集服务的依赖
type server struct {
	pagePath string // 采集页路径，每次请求重新读取，改页面不用重启
	outDir   string // 结果保存目录
}

// newServer 创建采集服务
func newServer(pagePath, outDir string) *server {
	return &server{pagePath: pagePath, outDir: outDir}
}

// routes 全部路由
func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handlePage())
	mux.HandleFunc("POST /result", s.handleResult())
	return mux
}

// handlePage 返回采集页
func (s *server) handlePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := os.ReadFile(s.pagePath)
		if err != nil {
			http.Error(w, "read collector page: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(page)
	}
}

// baselineSummary 结果中用于命名文件的字段
type baselineSummary struct {
	Auto *struct {
		Identity struct {
			UserAgent     string `json:"userAgent"`
			UserAgentData *struct {
				Model string `json:"model"`
			} `json:"userAgentData"`
		} `json:"identity"`
	} `json:"auto"`
}

// handleResult 接收一份结果：校验后格式化保存，返回文件名；不记录请求方地址
func (s *server) handleResult() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxResultBytes))
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				http.Error(w, "result larger than 1 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var sum baselineSummary
		if err := json.Unmarshal(body, &sum); err != nil || sum.Auto == nil {
			http.Error(w, "body is not a baseline result (JSON with an auto field)", http.StatusBadRequest)
			return
		}
		var pretty bytes.Buffer
		json.Indent(&pretty, body, "", "  ") // 上面已确认是合法 JSON，不会失败
		pretty.WriteByte('\n')

		id := sum.Auto.Identity
		model := ""
		if id.UserAgentData != nil {
			model = id.UserAgentData.Model
		}
		name, err := writeUnique(s.outDir, baselineName(id.UserAgent, model), pretty.Bytes())
		if err != nil {
			http.Error(w, "save result: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("已保存 %s（%s）", name, id.UserAgent)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"file": name})
	}
}

// otherChromium UA 里也带 Chrome/ 的其他浏览器：采集必须用正版 Chrome，这些记为 other，检查时能发现用错
var otherChromium = []string{"Edg", "OPR/", "MiuiBrowser", "XiaoMi/", "SamsungBrowser", "HeyTapBrowser", "UCBrowser",
	"MQQBrowser", "HuaweiBrowser", "VivoBrowser", "Quark", "baidu"}

// baselineName 由 UA 与机型推导结果文件的基本名：<系统>-<浏览器>-<机型>
func baselineName(ua, model string) string {
	osName := "other"
	switch {
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"):
		osName = "ios"
	case strings.Contains(ua, "Android"):
		osName = "android"
	}
	browser := "other"
	switch {
	case strings.Contains(ua, "CriOS/"):
		browser = "chrome"
	case strings.Contains(ua, "Chrome/"):
		browser = "chrome"
		for _, s := range otherChromium {
			if strings.Contains(ua, s) {
				browser = "other"
			}
		}
	case strings.Contains(ua, "Version/") && strings.Contains(ua, "Safari/"):
		browser = "safari"
	}
	if model == "" && osName == "ios" {
		model = "iphone"
	}
	return osName + "-" + browser + "-" + sanitizeName(model)
}

var nonNameChars = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeName 文件名片段只保留小写字母、数字和连字符；为空时返回 unknown
func sanitizeName(s string) string {
	s = strings.Trim(nonNameChars.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if s == "" {
		return "unknown"
	}
	return s
}

// writeUnique 把数据写到 dir/<base>.json；已存在时依次尝试 <base>-2.json、<base>-3.json……，返回实际文件名
func writeUnique(dir, base string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	for i := 1; ; i++ {
		name := base + ".json"
		if i > 1 {
			name = fmt.Sprintf("%s-%d.json", base, i)
		}
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create %s: %w", name, err)
		}
		_, werr := f.Write(data)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return "", fmt.Errorf("write %s: %w", name, werr)
		}
		return name, nil
	}
}
