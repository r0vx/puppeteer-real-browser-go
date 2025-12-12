#!/bin/bash

# 浏览器上下文和指纹浏览器示例运行脚本

echo "🎭 Puppeteer Real Browser Go - Examples"
echo "======================================"

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

# 创建临时目录
TEMP_DIR="temp_examples"
mkdir -p "$TEMP_DIR"

echo ""
echo "📋 Available Examples:"
echo "  1. Simple Context Test (Quick demo)"
echo "  2. Multi-Account Management Demo"
echo "  3. Fingerprint Browser Demo"
echo "  4. Chrome Extensions Demo"
echo "  5. Persistent Accounts Demo"
echo "  6. Pre-installed Extensions Demo"
echo "  7. Run All (Sequential)"
echo ""

read -p "Select example to run (1-7): " choice

case $choice in
    1)
        echo ""
        echo "🧪 Running Simple Context Test..."
        echo "================================="
        cd "$TEMP_DIR"
        cp ../simple_context_test.go ./main.go
        go mod init temp_simple_test 2>/dev/null || true
        
        echo "module temp_simple_test

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    2)
        echo ""
        echo "🔄 Running Multi-Account Management Demo..."
        echo "=========================================="
        cd "$TEMP_DIR"
        cp ../multi_account_demo.go ./main.go
        go mod init temp_multi_account 2>/dev/null || true
        go mod tidy 2>/dev/null || true
        
        # 添加依赖
        echo "module temp_multi_account

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    3)
        echo ""
        echo "🎭 Running Fingerprint Browser Demo..."
        echo "======================================"
        cd "$TEMP_DIR"
        cp ../fingerprint_browser_demo.go ./main.go
        go mod init temp_fingerprint 2>/dev/null || true
        go mod tidy 2>/dev/null || true
        
        # 添加依赖
        echo "module temp_fingerprint

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    4)
        echo ""
        echo "🧩 Running Chrome Extensions Demo..."
        echo "===================================="
        cd "$TEMP_DIR"
        cp ../extension_demo.go ./main.go
        go mod init temp_extension_demo 2>/dev/null || true
        
        echo "module temp_extension_demo

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    5)
        echo ""
        echo "💾 Running Persistent Accounts Demo..."
        echo "====================================="
        cd "$TEMP_DIR"
        cp ../persistent_accounts_demo.go ./main.go
        go mod init temp_persistent_accounts 2>/dev/null || true
        
        echo "module temp_persistent_accounts

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    6)
        echo ""
        echo "🧩 Running Pre-installed Extensions Demo..."
        echo "==========================================="
        cd "$TEMP_DIR"
        cp ../preinstalled_extensions_demo.go ./main.go
        go mod init temp_preinstalled_extensions 2>/dev/null || true
        
        echo "module temp_preinstalled_extensions

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    7)
        echo ""
        echo "🚀 Running All Examples Sequential..."
        echo "====================================="
        
        # 运行简单测试
        echo ""
        echo "🧪 Step 1: Simple Context Test"
        echo "------------------------------"
        cd "$TEMP_DIR"
        cp ../simple_context_test.go ./main.go
        go mod init temp_simple_test 2>/dev/null || true
        echo "module temp_simple_test

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        
        echo ""
        echo "⏳ Waiting 3 seconds before next demo..."
        sleep 3
        
        # 运行多账号管理演示
        echo ""
        echo "🔄 Step 2: Multi-Account Management Demo"
        echo "----------------------------------------"
        cd "$TEMP_DIR"
        cp ../multi_account_demo.go ./main.go
        go mod init temp_multi_account 2>/dev/null || true
        echo "module temp_multi_account

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        
        echo ""
        echo "⏳ Waiting 5 seconds before next demo..."
        sleep 5
        
        # 运行指纹浏览器演示
        echo ""
        echo "🎭 Step 3: Fingerprint Browser Demo"
        echo "----------------------------------"
        cd "$TEMP_DIR"
        rm -f main.go go.mod go.sum
        cp ../fingerprint_browser_demo.go ./main.go
        go mod init temp_fingerprint 2>/dev/null || true
        echo "module temp_fingerprint

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        
        echo ""
        echo "⏳ Waiting 5 seconds before next demo..."
        sleep 5
        
        # 运行插件演示
        echo ""
        echo "🧩 Step 4: Chrome Extensions Demo"
        echo "---------------------------------"
        cd "$TEMP_DIR"
        rm -f main.go go.mod go.sum
        cp ../extension_demo.go ./main.go
        go mod init temp_extension_demo 2>/dev/null || true
        echo "module temp_extension_demo

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        
        echo ""
        echo "⏳ Waiting 5 seconds before next demo..."
        sleep 5
        
        # 运行持久化账号演示
        echo ""
        echo "💾 Step 5: Persistent Accounts Demo"
        echo "----------------------------------"
        cd "$TEMP_DIR"
        rm -f main.go go.mod go.sum
        cp ../persistent_accounts_demo.go ./main.go
        go mod init temp_persistent_accounts 2>/dev/null || true
        echo "module temp_persistent_accounts

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        
        echo ""
        echo "⏳ Waiting 5 seconds before next demo..."
        sleep 5
        
        # 运行预装插件演示
        echo ""
        echo "🧩 Step 6: Pre-installed Extensions Demo"
        echo "---------------------------------------"
        cd "$TEMP_DIR"
        rm -f main.go go.mod go.sum
        cp ../preinstalled_extensions_demo.go ./main.go
        go mod init temp_preinstalled_extensions 2>/dev/null || true
        echo "module temp_preinstalled_extensions

go 1.23

replace github.com/HNRow/puppeteer-real-browser-go => ../..

require github.com/HNRow/puppeteer-real-browser-go v0.0.0-00010101000000-000000000000" > go.mod
        
        go run main.go
        cd ..
        ;;
    *)
        echo "❌ Invalid choice. Please select 1, 2, 3, 4, 5, 6, or 7."
        exit 1
        ;;
esac

# 清理临时文件
echo ""
echo "🧹 Cleaning up temporary files..."
rm -rf "$TEMP_DIR"

echo ""
echo "✅ Example execution completed!"
echo ""
echo "💡 Tips:"
echo "  - Check the browser windows that opened during the demo"
echo "  - Review the console output for detailed information"
echo "  - Modify the example files to test different scenarios"