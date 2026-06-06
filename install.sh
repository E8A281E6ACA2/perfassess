#!/bin/bash
# Perfassess - 一键安装脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
REPO="E8A281E6ACA2/perfassess"
VERSION="latest"
BINARY_NAME="perfassess"
INSTALL_DIR="/usr/local/bin"

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 检测操作系统和架构
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$OS" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            OS="windows"
            ;;
        *)
            print_error "不支持的操作系统: $OS"
            exit 1
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "不支持的架构: $ARCH"
            exit 1
            ;;
    esac
    
    print_info "检测到系统: $OS $ARCH"
}

# 下载二进制文件
download_binary() {
    print_info "正在下载 $BINARY_NAME..."
    
    # 构建下载URL
    if [ "$OS" = "windows" ]; then
        BINARY_FILE="${BINARY_NAME}.exe"
    else
        BINARY_FILE="${BINARY_NAME}_${OS}_${ARCH}"
    fi
    
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_FILE}"
    
    # 创建临时目录
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    # 下载文件
    if command -v curl &> /dev/null; then
        curl -fsSL -o "$BINARY_NAME" "$DOWNLOAD_URL"
    elif command -v wget &> /dev/null; then
        wget -q -O "$BINARY_NAME" "$DOWNLOAD_URL"
    else
        print_error "需要 curl 或 wget 来下载文件"
        exit 1
    fi
    
    if [ ! -f "$BINARY_NAME" ]; then
        print_error "下载失败"
        exit 1
    fi
    
    print_success "下载完成"
}

# 安装二进制文件
install_binary() {
    print_info "正在安装到 $INSTALL_DIR..."
    
    # 添加执行权限
    chmod +x "$BINARY_NAME"
    
    # 检查是否需要 sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        print_warning "需要管理员权限安装到 $INSTALL_DIR"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    fi
    
    print_success "安装完成"
}

# 验证安装
verify_installation() {
    print_info "验证安装..."
    
    if command -v "$BINARY_NAME" &> /dev/null; then
        VERSION_OUTPUT=$("$BINARY_NAME" --help | head -1)
        print_success "安装成功！"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  🎉 Perfassess已安装"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "快速开始："
        echo "  1. 运行交互式菜单:  $BINARY_NAME"
        echo "  2. 快速 CPU 测试:   $BINARY_NAME -b cpu"
        echo "  3. 完整测试 + Web:  $BINARY_NAME -b all --web"
        echo "  4. 查看帮助:        $BINARY_NAME --help"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    else
        print_error "安装验证失败"
        exit 1
    fi
}

# 清理临时文件
cleanup() {
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}

# 主函数
main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║          Perfassess                       ║"
    echo "║          Perfassess                         ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    
    # 设置清理陷阱
    trap cleanup EXIT
    
    # 执行安装步骤
    detect_platform
    download_binary
    install_binary
    verify_installation
}

# 运行主函数
main
