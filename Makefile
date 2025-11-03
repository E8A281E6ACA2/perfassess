# Makefile for Performance Assessment System

# 变量定义
APP_NAME=perfassess
VERSION=1.0.0
BUILD_DIR=build
CMD_DIR=cmd
MAIN_FILE=$(CMD_DIR)/main.go

# Go 相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# 构建标志
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -s -w"

# 平台相关
BINARY_UNIX=$(APP_NAME)_unix
BINARY_DARWIN=$(APP_NAME)_darwin
BINARY_WINDOWS=$(APP_NAME).exe

.PHONY: all build clean test deps help install run

# 默认目标
all: clean deps build

# 构建当前平台的可执行文件
build:
	@echo "构建 $(APP_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 构建所有平台的可执行文件
build-all: build-linux build-darwin build-windows
	@echo "所有平台构建完成"

# 构建 Linux 版本
build-linux:
	@echo "构建 Linux 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX)_amd64 $(MAIN_FILE)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX)_arm64 $(MAIN_FILE)
	@echo "Linux 版本构建完成"

# 构建 macOS 版本
build-darwin:
	@echo "构建 macOS 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_DARWIN)_amd64 $(MAIN_FILE)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_DARWIN)_arm64 $(MAIN_FILE)
	@echo "macOS 版本构建完成"

# 构建 Windows 版本
build-windows:
	@echo "构建 Windows 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_WINDOWS) $(MAIN_FILE)
	@echo "Windows 版本构建完成"

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf $(BUILD_DIR)
	@rm -rf logs/*.log
	$(GOCLEAN)
	@echo "清理完成"

# 运行测试
test:
	@echo "运行测试..."
	$(GOTEST) -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 下载依赖
deps:
	@echo "下载依赖..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "依赖下载完成"

# 安装到系统
install: build
	@echo "安装 $(APP_NAME)..."
	@cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "安装完成: /usr/local/bin/$(APP_NAME)"

# 运行程序
run: build
	@echo "运行 $(APP_NAME)..."
	@./$(BUILD_DIR)/$(APP_NAME)

# 运行程序（详细模式）
run-verbose: build
	@echo "运行 $(APP_NAME) (详细模式)..."
	@./$(BUILD_DIR)/$(APP_NAME) --verbose

# 运行 CPU 测试
run-cpu: build
	@echo "运行 CPU 测试..."
	@./$(BUILD_DIR)/$(APP_NAME) --tests cpu

# 运行所有测试
run-all: build
	@echo "运行所有测试..."
	@./$(BUILD_DIR)/$(APP_NAME) --tests all

# 格式化代码
fmt:
	@echo "格式化代码..."
	$(GOCMD) fmt ./...

# 代码检查
lint:
	@echo "运行代码检查..."
	@which golangci-lint > /dev/null || (echo "请先安装 golangci-lint" && exit 1)
	golangci-lint run ./...

# 显示帮助信息
help:
	@echo "Performance Assessment System - Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  make build          - 构建当前平台的可执行文件"
	@echo "  make build-all      - 构建所有平台的可执行文件"
	@echo "  make build-linux    - 构建 Linux 版本"
	@echo "  make build-darwin   - 构建 macOS 版本"
	@echo "  make build-windows  - 构建 Windows 版本"
	@echo "  make clean          - 清理构建产物"
	@echo "  make test           - 运行测试"
	@echo "  make test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  make deps           - 下载依赖"
	@echo "  make install        - 安装到系统"
	@echo "  make run            - 运行程序"
	@echo "  make run-verbose    - 运行程序（详细模式）"
	@echo "  make run-cpu        - 运行 CPU 测试"
	@echo "  make run-all        - 运行所有测试"
	@echo "  make fmt            - 格式化代码"
	@echo "  make lint           - 运行代码检查"
	@echo "  make help           - 显示此帮助信息"
