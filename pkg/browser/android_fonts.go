package browser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// androidFontRole 一类安卓字体。配置里统一改成内部族名，只能经由通用族与别名访问：
// 安卓上 Roboto、Noto 这些名字不能按名访问（真机基准里一个都检测不到）
type androidFontRole struct {
	Name    string         // 改名后的族名，也是链接文件名的前缀
	Pattern *regexp.Regexp // 匹配字体文件名
}

// androidFontRoles 安卓 fonts.xml 的通用族（Debian 包里的对应字体）：sans-serif = Roboto，serif = Noto Serif，
// monospace = Noto Sans Mono（安卓用 Droid Sans Mono，Debian 不提供），中文与表情回退
var androidFontRoles = []androidFontRole{
	{"android-sans", regexp.MustCompile(`^Roboto-[A-Za-z]+\.ttf$`)},
	{"android-serif", regexp.MustCompile(`^NotoSerif-[A-Za-z]+\.ttf$`)},
	{"android-mono", regexp.MustCompile(`^Noto(Sans)?Mono-[A-Za-z]+\.ttf$`)},
	{"android-cjk", regexp.MustCompile(`^NotoSansCJK-[A-Za-z]+\.ttc$`)},
	{"android-emoji", regexp.MustCompile(`^NotoColorEmoji\.ttf$`)},
}

// androidFontAliases 通用族与安卓 fonts.xml 里的别名（真机基准检测到的 9 个，加上 times）→ 字体类
var androidFontAliases = []struct{ Family, Role string }{
	{"sans-serif", "android-sans"}, {"serif", "android-serif"}, {"monospace", "android-mono"},
	{"Arial", "android-sans"}, {"Helvetica", "android-sans"}, {"Tahoma", "android-sans"}, {"Verdana", "android-sans"},
	{"Times", "android-serif"}, {"Times New Roman", "android-serif"}, {"Georgia", "android-serif"},
	{"Courier", "android-mono"}, {"Courier New", "android-mono"}, {"Monaco", "android-mono"},
}

// androidFontRoots 生产与测试镜像里字体所在的目录
var androidFontRoots = []string{"/usr/share/fonts", "/usr/local/share/fonts"}

// writeAndroidFontConfig 在 dir 下生成手机身份的 fontconfig 配置：把 roots 里各类安卓字体的文件链接进 dir/fonts（文件名加类前缀），
// 扫描时按前缀改族名，再配通用族与别名；不包含系统配置，其他字体对该进程不可见。返回配置路径与缺失的字体类
func writeAndroidFontConfig(dir string, roots []string) (string, []string, error) {
	fontsDir := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create %s: %w", fontsDir, err)
	}
	found := map[string]bool{}
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil // 目录不存在或读不了：跳过
			}
			for _, r := range androidFontRoles {
				if !r.Pattern.MatchString(d.Name()) {
					continue
				}
				link := filepath.Join(fontsDir, r.Name+"--"+d.Name())
				if _, err := os.Lstat(link); err == nil {
					continue // 同名文件已链接（另一个根目录里的重复副本）
				}
				if err := os.Symlink(path, link); err == nil {
					found[r.Name] = true
				}
			}
			return nil
		})
	}
	var missing []string
	for _, r := range androidFontRoles {
		if !found[r.Name] {
			missing = append(missing, r.Name)
		}
	}

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"urn:fontconfig:fonts.dtd\">\n<fontconfig>\n")
	fmt.Fprintf(&b, "  <dir>%s</dir>\n  <cachedir>%s</cachedir>\n", fontsDir, filepath.Join(dir, "cache"))
	for _, r := range androidFontRoles {
		fmt.Fprintf(&b, "  <match target=\"scan\"><test name=\"file\" compare=\"contains\"><string>/%s--</string></test>"+
			"<edit name=\"family\" mode=\"assign_replace\"><string>%s</string></edit></match>\n", r.Name, r.Name)
	}
	for _, a := range androidFontAliases {
		fmt.Fprintf(&b, "  <alias binding=\"same\"><family>%s</family><prefer><family>%s</family><family>android-cjk</family><family>android-emoji</family></prefer></alias>\n", a.Family, a.Role)
	}
	b.WriteString("</fontconfig>\n")
	conf := filepath.Join(dir, "fonts.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", nil, fmt.Errorf("write %s: %w", conf, err)
	}
	return conf, missing, nil
}
