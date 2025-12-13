#!/bin/bash

# 清理浏览器Profile脚本
# 用于清理测试时产生的临时Profile，避免"恢复页面"提示

echo "🧹 清理浏览器Profile工具"
echo "================================"
echo ""

# 默认Profile目录（根据操作系统不同）
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    PROFILE_DIR="$HOME/Library/Application Support/Chrome/puppeteer-real-browser"
    echo "📍 检测到 macOS 系统"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    PROFILE_DIR="$HOME/.config/google-chrome/puppeteer-real-browser"
    echo "📍 检测到 Linux 系统"
else
    # Windows (Git Bash)
    PROFILE_DIR="$APPDATA/Google/Chrome/User Data/puppeteer-real-browser"
    echo "📍 检测到 Windows 系统"
fi

echo "📂 Profile目录: $PROFILE_DIR"
echo ""

# 列出所有Profile
if [ -d "$PROFILE_DIR" ]; then
    echo "📋 发现的Profile:"
    find "$PROFILE_DIR" -maxdepth 1 -type d | tail -n +2 | while read -r dir; do
        size=$(du -sh "$dir" 2>/dev/null | cut -f1)
        echo "   • $(basename "$dir") ($size)"
    done
    echo ""
    
    # 询问是否删除
    read -p "❓ 是否删除所有Profile? [y/N] " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🗑️  正在删除..."
        rm -rf "$PROFILE_DIR"/*
        echo "✅ 清理完成！"
    else
        echo "❌ 已取消清理"
    fi
else
    echo "ℹ️  没有找到Profile目录，可能还没有运行过程序"
fi

echo ""
echo "💡 提示："
echo "   • 清理后首次运行可能会慢一些"
echo "   • 清理后将不会有'恢复页面'的提示"
echo "   • Cookie和登录状态会丢失"
echo ""

