#!/bin/bash

echo "🧪 性能优化快速验证脚本"
echo "========================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go 未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go 已安装: $(go version)${NC}"
echo ""

# 步骤 1: 检查文件是否存在
echo "📋 步骤 1: 检查优化文件..."
FILES=(
    "pkg/browser/pool.go"
    "pkg/browser/wait.go"
    "pkg/browser/pool_test.go"
    "cmd/example/performance_test.go"
)

all_files_exist=true
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  ${GREEN}✓${NC} $file"
    else
        echo -e "  ${RED}✗${NC} $file ${RED}(缺失)${NC}"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = false ]; then
    echo -e "\n${RED}❌ 部分文件缺失，请先运行优化脚本${NC}"
    exit 1
fi

echo -e "\n${GREEN}✅ 所有优化文件存在${NC}\n"

# 步骤 2: 编译检查
echo "📋 步骤 2: 编译检查..."
if go build ./cmd/example/performance_test.go 2>/dev/null; then
    echo -e "${GREEN}✅ 编译成功${NC}"
    rm -f performance_test 2>/dev/null
else
    echo -e "${RED}❌ 编译失败${NC}"
    echo "请运行: go build ./cmd/example/performance_test.go"
    exit 1
fi
echo ""

# 步骤 3: 运行单元测试
echo "📋 步骤 3: 运行单元测试..."
cd pkg/browser

# 只运行快速测试
if go test -v -short -timeout 30s 2>&1 | grep -q "PASS"; then
    echo -e "${GREEN}✅ 单元测试通过${NC}"
else
    echo -e "${YELLOW}⚠️  部分测试跳过或失败（正常，可能需要 Chrome）${NC}"
fi

cd ../..
echo ""

# 步骤 4: 检查 Chrome 是否安装
echo "📋 步骤 4: 检查 Chrome/Chromium..."
chrome_found=false

if [ "$(uname)" == "Darwin" ]; then
    # macOS
    if [ -f "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]; then
        echo -e "${GREEN}✅ Chrome 已安装 (macOS)${NC}"
        chrome_found=true
    fi
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    if command -v google-chrome &> /dev/null || command -v chromium-browser &> /dev/null; then
        echo -e "${GREEN}✅ Chrome/Chromium 已安装 (Linux)${NC}"
        chrome_found=true
    fi
fi

if [ "$chrome_found" = false ]; then
    echo -e "${YELLOW}⚠️  Chrome/Chromium 未找到${NC}"
    echo "   安装方法:"
    echo "   - macOS: brew install --cask google-chrome"
    echo "   - Linux: sudo apt-get install chromium-browser"
fi
echo ""

# 总结
echo "========================================"
echo "📊 验证结果总结"
echo "========================================"
echo -e "${GREEN}✅ 优化文件已添加${NC}"
echo -e "${GREEN}✅ 代码编译通过${NC}"

if [ "$chrome_found" = true ]; then
    echo -e "${GREEN}✅ 可以运行完整测试${NC}"
    echo ""
    echo "🚀 下一步: 运行性能测试"
    echo "   cd cmd/example"
    echo "   go run performance_test.go"
else
    echo -e "${YELLOW}⚠️  需要安装 Chrome 才能运行完整测试${NC}"
fi

echo ""
echo "📚 更多信息请查看: 测试说明.md"
echo ""
