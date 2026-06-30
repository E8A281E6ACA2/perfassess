# Makefile for Perfassess

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
BINARY_LINUX=$(APP_NAME)_linux
BINARY_DARWIN=$(APP_NAME)_darwin
BINARY_WINDOWS=$(APP_NAME).exe

.PHONY: all build build-all build-linux build-darwin build-windows release-checksums clean test test-coverage deps install run run-verbose run-cpu run-all bootstrap auto fmt fmt-check lint schema-check cli-smoke json-smoke project-invariants-smoke candidate-baseline-split-smoke remote-runbook-smoke release-runbook-smoke release-workflow-smoke report-contract-smoke calibration-summary-smoke calibration-collect-smoke redact-report-smoke auto-running-summary-smoke bootstrap-prebuilt-smoke artifact-verify-smoke vps-acceptance-summary-smoke vps-acceptance-strict-preflight-smoke release-smoke release-checksums-smoke verify-release-assets-smoke vps-acceptance vps-acceptance-low-standard validate pre-commit release-check help

# 默认目标
all: clean deps build

# 构建当前平台的可执行文件
build:
	@echo "构建 $(APP_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 构建所有平台的可执行文件
build-all: build-linux build-darwin build-windows release-checksums
	@echo "所有平台构建完成"

# 构建 Linux 版本
build-linux:
	@echo "构建 Linux 版本..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_LINUX)_amd64 $(MAIN_FILE)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_LINUX)_arm64 $(MAIN_FILE)
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

# 生成发布产物 SHA256 清单
release-checksums:
	@echo "生成发布产物 SHA256 清单..."
	@cd $(BUILD_DIR) && sha256sum $(BINARY_LINUX)_amd64 $(BINARY_LINUX)_arm64 $(BINARY_DARWIN)_amd64 $(BINARY_DARWIN)_arm64 $(BINARY_WINDOWS) > checksums.txt
	@echo "SHA256 清单已生成: $(BUILD_DIR)/checksums.txt"

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
	$(GOTEST) ./... -count=1

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

# 一键非交互测评与验收
auto:
	@echo "运行一键非交互测评与验收..."
	@scripts/perfassess-auto.sh

# 新服务器一键准备环境、构建、测试与验收
bootstrap:
	@echo "准备环境并运行一键测评..."
	@scripts/bootstrap.sh

# 格式化代码
fmt:
	@echo "格式化代码..."
	$(GOCMD) fmt ./...

# 检查 Go 代码格式
fmt-check:
	@echo "检查 Go 代码格式..."
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './build/*'))" || (echo "存在未格式化的 Go 文件:" && gofmt -l $$(find . -name '*.go' -not -path './build/*') && exit 1)

# 代码检查
lint:
	@echo "运行代码检查..."
	@which golangci-lint > /dev/null || (echo "请先安装 golangci-lint" && exit 1)
	golangci-lint run ./...

# 检查 JSON 文档格式
schema-check:
	@echo "检查 JSON schema 和示例报告..."
	@python3 -m json.tool docs/report.schema.json >/dev/null
	@python3 -m json.tool docs/examples/report-json-sample.json >/dev/null

# CLI 基础冒烟测试
cli-smoke:
	@echo "运行 CLI 冒烟测试..."
	@$(GOCMD) run ./cmd --help >/dev/null
	@$(GOCMD) run ./cmd check-deps >/dev/null

# 生成一份快速 JSON 报告并校验格式
json-smoke:
	@echo "运行快速 JSON 报告冒烟测试..."
	@$(GOCMD) run ./cmd --quick --output-format json -o /tmp/perfassess-release-smoke.json >/tmp/perfassess-release-smoke.stdout
	@python3 -m json.tool /tmp/perfassess-release-smoke.json >/dev/null

# 校验项目级不变量，防止旧命名、隐私边界和已移除能力回归
project-invariants-smoke:
	@echo "运行项目级不变量检查..."
	@bash scripts/project-invariants-smoke.sh

# 校验候选基线拆分提交计划
candidate-baseline-split-smoke:
	@echo "运行候选基线拆分计划冒烟测试..."
	@bash scripts/candidate-baseline-split-smoke.sh

# 校验远程服务器测试手册，防止 bootstrap 使用文档漂移
remote-runbook-smoke:
	@echo "运行远程测试手册冒烟测试..."
	@bash scripts/remote-runbook-smoke.sh

# 校验 GitHub Release 发布手册
release-runbook-smoke:
	@echo "运行发布手册冒烟测试..."
	@bash scripts/release-runbook-smoke.sh

# 校验 GitHub Release workflow
release-workflow-smoke:
	@echo "运行发布 workflow 冒烟测试..."
	@bash scripts/release-workflow-smoke.sh

# 校验报告语义和 Web 渲染契约
report-contract-smoke: build
	@echo "运行报告和 Web 契约冒烟测试..."
	@scripts/report-contract-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 校验脱敏校准样本汇总工具
calibration-summary-smoke: build
	@echo "运行校准样本汇总冒烟测试..."
	@scripts/calibration-summary-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 校验脱敏校准样本收集助手
calibration-collect-smoke: build
	@echo "运行校准样本收集冒烟测试..."
	@scripts/calibration-collect-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 校验 JSON 报告脱敏工具
redact-report-smoke:
	@echo "运行报告脱敏工具冒烟测试..."
	@bash scripts/redact-report-smoke.sh

# 校验自动测评运行中和失败时 summary.md 可用
auto-running-summary-smoke:
	@echo "运行自动测评运行中摘要冒烟测试..."
	@bash scripts/auto-running-summary-smoke.sh

# 校验 bootstrap 可使用预构建二进制跳过源码构建
bootstrap-prebuilt-smoke:
	@echo "运行 bootstrap 预构建二进制冒烟测试..."
	@bash scripts/bootstrap-prebuilt-smoke.sh

# 校验自动测评报告产物清单
artifact-verify-smoke: build
	@echo "运行报告产物清单校验冒烟测试..."
	@bash scripts/artifact-verify-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 校验 VPS 验收摘要包含报告覆盖和校准适用性
vps-acceptance-summary-smoke: build
	@echo "运行 VPS 验收摘要冒烟测试..."
	@bash scripts/vps-acceptance-summary-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 校验严格 VPS 验收的资源预检会在报告生成前失败
vps-acceptance-strict-preflight-smoke: build
	@echo "运行严格 VPS 验收资源预检冒烟测试..."
	@bash scripts/vps-acceptance-strict-preflight-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 使用发布二进制执行端到端冒烟测试
release-smoke: build
	@echo "运行发布二进制端到端冒烟测试..."
	@scripts/release-smoke.sh $(BUILD_DIR)/$(APP_NAME)

# 校验发布产物 SHA256 清单
release-checksums-smoke:
	@echo "运行发布产物 SHA256 清单冒烟测试..."
	@bash scripts/release-checksums-smoke.sh $(BUILD_DIR)

# 校验发布后资产下载与 SHA256 校验脚本
verify-release-assets-smoke:
	@echo "运行 Release 资产校验脚本冒烟测试..."
	@bash scripts/verify-release-assets-smoke.sh

# 在真实 VPS 或测试机上执行验收
vps-acceptance: build
	@echo "运行真实 VPS 验收..."
	@scripts/vps-acceptance.sh

# 在真实 VPS 或测试机上执行推荐 low/standard 验收矩阵
vps-acceptance-low-standard: build
	@echo "运行真实 VPS low/standard 验收矩阵..."
	@PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh

# 日常验证入口
validate: fmt-check schema-check project-invariants-smoke candidate-baseline-split-smoke remote-runbook-smoke release-runbook-smoke release-workflow-smoke test build cli-smoke report-contract-smoke calibration-summary-smoke calibration-collect-smoke redact-report-smoke auto-running-summary-smoke bootstrap-prebuilt-smoke verify-release-assets-smoke artifact-verify-smoke vps-acceptance-summary-smoke vps-acceptance-strict-preflight-smoke
	@echo "日常验证通过"

# 提交前验证入口
pre-commit:
	@git status --short --branch
	@git diff --check
	@bash -n install.sh uninstall.sh scripts/*.sh
	@python3 -m py_compile scripts/perfassess-progress-server.py scripts/calibration-summary.py scripts/verify-artifacts.py scripts/redact-report.py
	@$(MAKE) validate
	@echo "提交前验证通过"

# 发布前验证入口
release-check: validate release-smoke build-all release-checksums-smoke
	@echo "发布前验证通过"

# 显示帮助信息
help:
	@echo "Perfassess - Makefile"
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
	@echo "  make bootstrap      - 准备环境并运行一键测评"
	@echo "  make auto           - 一键非交互测评与验收"
	@echo "  make fmt            - 格式化代码"
	@echo "  make fmt-check      - 检查 Go 代码格式"
	@echo "  make lint           - 运行代码检查"
	@echo "  make schema-check   - 检查 JSON schema 和示例报告"
	@echo "  make cli-smoke      - 运行 CLI 基础冒烟测试"
	@echo "  make json-smoke     - 生成快速 JSON 报告并校验格式"
	@echo "  make project-invariants-smoke - 校验项目级不变量"
	@echo "  make candidate-baseline-split-smoke - 校验候选基线拆分提交计划"
	@echo "  make remote-runbook-smoke - 校验远程测试手册"
	@echo "  make release-runbook-smoke - 校验 GitHub Release 发布手册"
	@echo "  make release-workflow-smoke - 校验 GitHub Release workflow"
	@echo "  make report-contract-smoke - 校验报告语义和 Web 渲染契约"
	@echo "  make calibration-summary-smoke - 校验脱敏校准样本汇总工具"
	@echo "  make calibration-collect-smoke - 校验脱敏校准样本收集助手"
	@echo "  make redact-report-smoke - 校验 JSON 报告脱敏工具"
	@echo "  make auto-running-summary-smoke - 校验自动测评运行中摘要"
	@echo "  make bootstrap-prebuilt-smoke - 校验 bootstrap 预构建二进制路径"
	@echo "  make artifact-verify-smoke - 校验自动测评报告产物清单"
	@echo "  make vps-acceptance-summary-smoke - 校验 VPS 验收摘要和校准适用性"
	@echo "  make vps-acceptance-strict-preflight-smoke - 校验严格 VPS 验收资源预检"
	@echo "  make release-smoke  - 使用发布二进制执行端到端冒烟测试"
	@echo "  make release-checksums-smoke - 校验发布产物 SHA256 清单"
	@echo "  make verify-release-assets-smoke - 校验 Release 资产校验脚本"
	@echo "  make vps-acceptance - 在真实 VPS 或测试机上执行验收"
	@echo "  make vps-acceptance-low-standard - 在真实 VPS 上执行 low/standard 验收矩阵"
	@echo "  make validate       - 日常验证：格式、schema、项目不变量、测试、构建、CLI、报告契约、校准样本、产物清单和严格验收预检"
	@echo "  make pre-commit     - 提交前验证：状态、diff、脚本语法、Python 编译和 validate"
	@echo "  make release-check  - 发布前验证：validate、二进制冒烟、跨平台构建"
	@echo "  make help           - 显示此帮助信息"
