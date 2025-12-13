#!/bin/bash

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║       🔍 浏览器资源监控 & 稳定性测试 启动脚本                 ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 菜单
echo "请选择要运行的工具："
echo ""
echo "  ${GREEN}1${NC}. 📊 实时资源监控 (带模拟请求)"
echo "  ${GREEN}2${NC}. 🧪 稳定性测试 - 快速 (5分钟)"
echo "  ${GREEN}3${NC}. 🧪 稳定性测试 - 中期 (1小时)"
echo "  ${GREEN}4${NC}. 🧪 稳定性测试 - 长期 (6小时)"
echo "  ${GREEN}5${NC}. 🧪 稳定性测试 - 自定义"
echo "  ${GREEN}6${NC}. ❌ 退出"
echo ""

read -p "请输入选项 [1-6]: " choice

case $choice in
  1)
    echo ""
    echo "${YELLOW}🚀 启动实时资源监控...${NC}"
    echo "   - 每 2 秒刷新一次"
    echo "   - 会自动模拟 100 个请求"
    echo "   - 按 Ctrl+C 停止"
    echo ""
    sleep 2
    go run cmd/monitor/main.go
    ;;
    
  2)
    echo ""
    echo "${YELLOW}🚀 启动快速稳定性测试 (5分钟)...${NC}"
    echo "   - 测试时长: 5 分钟"
    echo "   - 并发数: 5"
    echo "   - 请求间隔: 2 秒"
    echo ""
    sleep 2
    go run cmd/stability_test/main.go -duration=5m -concurrency=5 -delay=2s
    ;;
    
  3)
    echo ""
    echo "${YELLOW}🚀 启动中期稳定性测试 (1小时)...${NC}"
    echo "   - 测试时长: 1 小时"
    echo "   - 并发数: 10"
    echo "   - 请求间隔: 2 秒"
    echo ""
    read -p "确认启动？(y/N): " confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
      go run cmd/stability_test/main.go -duration=1h -concurrency=10 -delay=2s
    else
      echo "已取消"
    fi
    ;;
    
  4)
    echo ""
    echo "${YELLOW}🚀 启动长期稳定性测试 (6小时)...${NC}"
    echo "   - 测试时长: 6 小时"
    echo "   - 并发数: 5"
    echo "   - 请求间隔: 3 秒"
    echo ""
    echo "⚠️  这将运行 6 小时，建议后台运行："
    echo "   nohup ./run_monitor.sh > test.log 2>&1 &"
    echo ""
    read -p "确认启动？(y/N): " confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
      go run cmd/stability_test/main.go -duration=6h -concurrency=5 -delay=3s
    else
      echo "已取消"
    fi
    ;;
    
  5)
    echo ""
    echo "${YELLOW}🎛️  自定义稳定性测试${NC}"
    echo ""
    read -p "测试时长 (如: 30m, 2h): " duration
    read -p "并发数 (1-20): " concurrency
    read -p "请求间隔 (如: 1s, 2s): " delay
    echo ""
    echo "将运行: $duration 测试，$concurrency 并发，$delay 间隔"
    echo ""
    read -p "确认启动？(y/N): " confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
      go run cmd/stability_test/main.go -duration=$duration -concurrency=$concurrency -delay=$delay
    else
      echo "已取消"
    fi
    ;;
    
  6)
    echo ""
    echo "👋 再见！"
    exit 0
    ;;
    
  *)
    echo ""
    echo "❌ 无效选项"
    exit 1
    ;;
esac

echo ""
echo "✅ 测试完成！"

