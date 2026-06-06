#!/bin/bash
# Perfassess - 卸载脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BINARY_NAME="perfassess"
INSTALL_DIR="/usr/local/bin"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║          卸载Perfassess                                     ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# 检查是否已安装
if ! command -v "$BINARY_NAME" &> /dev/null; then
    print_error "$BINARY_NAME 未安装"
    exit 1
fi

# 确认卸载
read -p "确定要卸载 $BINARY_NAME 吗？[y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "取消卸载"
    exit 0
fi

print_info "正在卸载..."

# 删除二进制文件
BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"
if [ -f "$BINARY_PATH" ]; then
    if [ -w "$INSTALL_DIR" ]; then
        rm "$BINARY_PATH"
    else
        sudo rm "$BINARY_PATH"
    fi
    print_success "已删除 $BINARY_PATH"
fi

# 删除日志文件（可选）
read -p "是否删除日志文件？[y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -d "logs" ]; then
        rm -rf logs
        print_success "已删除日志文件"
    fi
fi

print_success "卸载完成"
