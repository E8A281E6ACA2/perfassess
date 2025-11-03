# 高性能多终端自动化性能评估系统

一款用于自动化评估计算机和服务器性能的工具。支持 CPU、内存、磁盘、网络等多项性能测试，提供清晰的性能报告和综合评分。

## 快速开始

### 方式一：交互式菜单（推荐新手）

```bash
# 1. 构建程序
make build

# 2. 运行（自动进入交互式菜单）
./build/perfassess

# 3. 按照菜单提示选择测试项目，开始评估！
```

就这么简单！程序会引导您完成所有配置。

### 方式二：命令行模式（推荐高级用户）

```bash
# 快速 CPU 测试
./build/perfassess -b cpu

# 完整测试 + Web 报告
./build/perfassess -b all --web

# 完整测试 + 所有功能
./build/perfassess -b all --route-trace --streaming --web -o report.txt
```

## ✨ 功能亮点

### 🎯 三种使用方式，满足不同需求

1. **交互式菜单** - 中文界面，纯数字选择，新手友好
2. **命令行模式** - 快速执行，适合自动化脚本
3. **Web 报告** - 美观的 HTML 界面，可视化展示（新功能）

### 🚀 核心功能

- ✅ **多维度性能测试**
  - CPU 性能测试（单核/多核）
  - 内存读写速度测试
  - 磁盘 I/O 性能测试
  - 网络延迟和带宽测试
  - 路由追踪测试（可选）
  - 流媒体解锁检测（可选）

- ✅ **系统信息收集**
  - CPU 型号、核心数、频率
  - 内存容量和类型
  - 磁盘容量和类型
  - 操作系统信息
  - 虚拟化检测
  - IP 地理位置信息

- ✅ **智能评分系统**
  - 各项性能单独评分（0-100）
  - 综合性能评分（加权平均）
  - 性能等级评定（优秀/良好/一般/较差）

- ✅ **多平台支持**
  - Linux
  - macOS
  - Windows

- ✅ **友好的输出**
  - 彩色终端输出
  - 易读的报告格式
  - 支持保存到文件
  - 🌐 **Web 报告服务器**（新功能）
    - 美观的 HTML 界面
    - 响应式设计
    - 性能评分可视化
    - 进度条展示
    - 一键启动，浏览器查看

## 安装

### 🚀 方式一：一键安装（推荐）

**Linux / macOS**:
```bash
curl -fsSL https://raw.githubusercontent.com/BinaryResearcher/high-performance-multi-terminal-automated-performance-evaluation-system/main/install.sh | bash
```

或使用 wget:
```bash
wget -qO- https://raw.githubusercontent.com/你的用户名/项目名/main/install.sh | bash
```

**Windows (PowerShell)**:
```powershell
# 下载最新版本
Invoke-WebRequest -Uri "https://github.com/你的用户名/项目名/releases/latest/download/perfassess.exe" -OutFile "perfassess.exe"
```

### 📦 方式二：下载预编译二进制文件

从 [Releases 页面](https://github.com/你的用户名/项目名/releases) 下载对应平台的文件：

| 平台 | 文件名 |
|------|--------|
| Linux (x64) | `perfassess_linux_amd64` |
| Linux (ARM64) | `perfassess_linux_arm64` |
| macOS (Intel) | `perfassess_darwin_amd64` |
| macOS (Apple Silicon) | `perfassess_darwin_arm64` |
| Windows | `perfassess.exe` |

**安装步骤**：
```bash
# 1. 下载文件
# 2. 添加执行权限（Linux/macOS）
chmod +x perfassess_*

# 3. 移动到系统路径
sudo mv perfassess_* /usr/local/bin/perfassess

# 4. 验证安装
perfassess --help
```

### 🔧 方式三：从源码构建

确保已安装 Go 1.21 或更高版本。

```bash
# 克隆仓库
git clone https://github.com/你的用户名/项目名.git
cd 项目名

# 下载依赖
make deps

# 构建
make build

# 可执行文件位于 build/perfassess
```

**跨平台编译**：
```bash
# 构建所有平台版本
make build-all

# 或单独构建特定平台
make build-linux    # Linux (amd64, arm64)
make build-darwin   # macOS (amd64, arm64)
make build-windows  # Windows (amd64)
```

**安装到系统**：
```bash
# 安装到 /usr/local/bin
make install
```

### 🗑️ 卸载

```bash
# 使用卸载脚本
curl -fsSL https://raw.githubusercontent.com/你的用户名/项目名/main/uninstall.sh | bash

# 或手动删除
sudo rm /usr/local/bin/perfassess
```

## 使用方法

### 🎯 交互式模式（推荐新手）

直接运行程序，无需任何参数，即可进入友好的交互式菜单：

```bash
# 进入交互式菜单
./build/perfassess

# 或明确指定交互式模式
./build/perfassess --interactive
./build/perfassess -i
```

交互式菜单将引导您：
1. 选择检测项目（性能测试/路由追踪/流媒体检测/自定义组合）
2. 配置输出选项（详细模式/保存文件）
3. 确认配置后开始测试

### 📝 命令行模式（适合自动化）

命令行模式适合脚本自动化、CI/CD 集成或高级用户快速测试。

#### 基础用法

```bash
# 运行所有检测
./build/perfassess --benchmarks all
# 或使用简写
./build/perfassess -b all

# 或使用 make
make run
```

#### 单项测试

```bash
# 只测试 CPU
./build/perfassess -b cpu

# 只测试内存
./build/perfassess -b memory

# 只测试磁盘
./build/perfassess -b disk

# 只测试网络
./build/perfassess -b network
```

#### 组合测试

```bash
# CPU + 内存
./build/perfassess -b cpu,memory

# CPU + 内存 + 磁盘
./build/perfassess -b cpu,memory,disk
```

#### 扩展功能

```bash
# 启用路由追踪
./build/perfassess -b all --route-trace

# 启用流媒体检测
./build/perfassess -b all --streaming

# 启用所有扩展功能
./build/perfassess -b all --route-trace --streaming
```

#### 🌐 Web 报告（新功能）

```bash
# 启用 Web 报告服务器（默认端口 8080）
./build/perfassess -b all --web

# 自定义端口
./build/perfassess -b all --web --port 9090

# 完整功能 + Web 报告
./build/perfassess -b all --route-trace --streaming --web
```

**Web 报告特点**：
- 📊 美观的 HTML 界面
- 📈 可视化性能评分
- 💻 响应式设计，支持手机/平板
- 🎨 现代化 UI，渐变色背景
- 📋 详细的系统信息和测试结果

**使用流程**：
1. 运行带 `--web` 参数的命令
2. 等待测试完成
3. 看到提示：`🌐 Web 服务器已启动: http://localhost:8080`
4. 在浏览器打开该地址查看报告
5. 按 `Ctrl+C` 停止服务器

#### 输出选项

```bash
# 保存报告到文件
./build/perfassess -b all -o report.txt

# 详细输出模式
./build/perfassess -b all -v

# 组合使用
./build/perfassess -b all -v -o report.txt --web
```

### 命令行参数

```bash
perfassess [flags]

Flags:
  -i, --interactive          启用交互式菜单模式（推荐新手使用）
  -b, --benchmarks strings   指定要运行的检测项目 (cpu,memory,disk,network,all) (default [all])
  -o, --output string        指定输出文件路径（不指定则只输出到控制台）
  -v, --verbose              启用详细输出模式
      --log-level string     设置日志级别 (debug,info,warn,error) (default "info")
      --route-trace          启用路由追踪功能
      --streaming            启用流媒体解锁检测功能
      --web                  启用 Web 报告服务器（新功能）
      --port int             Web 服务器端口 (default 8080)
  -h, --help                 显示帮助信息

注：--tests 参数仍然支持，但推荐使用 --benchmarks
```

### 使用示例

#### 交互式模式示例

```bash
# 最简单的方式 - 直接运行
./build/perfassess

# 程序会显示友好的菜单：
# ╔════════════════════════════════════════════════════════════════╗
# ║          高性能多终端自动化性能评估系统                       ║
# ╚════════════════════════════════════════════════════════════════╝
# 
# 【步骤 1/2】选择检测项目
#   1. 完整检测（推荐）
#   2. CPU 性能测试
#   3. 内存性能测试
#   4. 磁盘性能测试
#   5. 网络性能测试
#   6. 路由追踪测试
#   7. 流媒体解锁检测
#   8. 自定义组合
#   ...
```

#### 命令行模式示例

```bash
# 只运行 CPU 检测
./build/perfassess --benchmarks cpu
# 或使用简写
./build/perfassess -b cpu

# 运行 CPU 和内存检测
./build/perfassess --benchmarks cpu,memory

# 运行所有检测并保存报告到文件
./build/perfassess --output report.txt

# 启用详细输出模式
./build/perfassess --verbose

# 启用路由追踪功能
./build/perfassess --route-trace

# 启用流媒体解锁检测
./build/perfassess --streaming

# 组合使用多个功能
./build/perfassess -b all --route-trace --streaming -o report.txt

# 启用 Web 报告服务器（新功能）
./build/perfassess -b all --web
# 输出: 🌐 Web 服务器已启动: http://localhost:8080

# 自定义 Web 服务器端口
./build/perfassess -b all --web --port 9090

# 使用 make 快捷命令
make run-cpu        # 运行 CPU 测试
make run-all        # 运行所有测试
make run-verbose    # 详细模式运行
```

## 输出示例

```
╔════════════════════════════════════════════════════════════════╗
║          高性能多终端自动化性能评估系统 - 评估报告            ║
╚════════════════════════════════════════════════════════════════╝

会话ID:         session_1762170027
报告时间:       2025-11-03 19:40:29

=== 系统信息 ===

CPU型号:        Apple M1
CPU核心数:      8 核心
CPU线程数:      8 线程
CPU频率:        3200.00 MHz

内存总量:       16384 MB
可用内存:       8192 MB

磁盘总量:       500.00 GB
可用空间:       250.00 GB
磁盘类型:       SSD

操作系统:       macOS
系统版本:       14.0
系统架构:       arm64

虚拟化类型:     物理机

公网IP:         xxx.xxx.xxx.xxx
地理位置:       中国, 北京
ISP:            China Telecom

=== 性能测试结果 ===

--- CPU性能测试 ---
状态:           success
耗时:           0.19 秒
测试指标:
  单核评分:     100.00
  多核评分:     100.00
  总体评分:     100.00
  CPU核心数:    8

--- 内存性能测试 ---
状态:           success
耗时:           2.50 秒
测试指标:
  读取速度:     15000.00 MB/s
  写入速度:     12000.00 MB/s

--- 磁盘性能测试 ---
状态:           success
耗时:           5.30 秒
测试指标:
  顺序读取:     2500.00 MB/s
  顺序写入:     1800.00 MB/s
  随机IOPS:     50000

--- 网络性能测试 ---
状态:           success
耗时:           3.20 秒
测试指标:
  平均延迟:     15.50 ms
  下载速度:     500.00 Mbps
  上传速度:     200.00 Mbps

=== 综合性能评分 ===

CPU评分:        100.00 / 100
内存评分:       90.00 / 100
磁盘评分:       95.00 / 100
网络评分:       85.00 / 100

总体评分:       92.50 / 100
性能等级:       优秀

=== 测试统计 ===

成功测试:       4
失败测试:       0
跳过测试:       0
```

## 开发

### 项目结构

```
.
├── cmd/                    # 主程序入口
│   └── main.go
├── internal/               # 内部包
│   ├── cli/               # 命令行界面
│   ├── collector/         # 信息收集器
│   ├── config/            # 配置管理
│   ├── controller/        # 核心控制器
│   ├── models/            # 数据模型
│   ├── platform/          # 平台抽象
│   ├── reporter/          # 报告生成
│   └── tests/             # 性能测试
├── pkg/                   # 公共包
│   ├── logger/            # 日志管理
│   └── utils/             # 工具函数
├── build/                 # 构建输出
├── logs/                  # 日志文件
├── Makefile              # 构建脚本
├── go.mod                # Go 模块定义
└── README.md             # 项目文档
```

### 运行测试

```bash
# 运行所有单元测试
make test

# 运行测试并生成覆盖率报告
make test-coverage
```

### 代码格式化

```bash
# 格式化代码
make fmt

# 运行代码检查（需要安装 golangci-lint）
make lint
```

## 技术栈

- **语言**: Go 1.21+
- **CLI 框架**: [Cobra](https://github.com/spf13/cobra)
- **日志库**: [Zap](https://github.com/uber-go/zap)
- **系统信息**: [gopsutil](https://github.com/shirou/gopsutil)
- **配置管理**: [Viper](https://github.com/spf13/viper)

## 性能评分算法

### 评分基准

- **CPU**: 基于单核和多核运算能力
  - 单核基准: 100万次操作 = 60分
  - 多核基准: 500万次操作 = 60分

- **内存**: 基于读写速度
  - 读取基准: 5000 MB/s = 60分
  - 写入基准: 3000 MB/s = 60分

- **磁盘**: 基于顺序读写和随机 IOPS
  - 顺序读取: 500 MB/s = 60分
  - 顺序写入: 300 MB/s = 60分
  - 随机 IOPS: 5000 = 60分

- **网络**: 基于延迟和带宽
  - 延迟: 50ms = 60分（反比例）
  - 下载速度: 100 Mbps = 60分
  - 上传速度: 50 Mbps = 60分

### 综合评分

使用加权平均算法计算总分：
- CPU: 30%
- 内存: 20%
- 磁盘: 25%
- 网络: 25%

### 性能等级

- **优秀**: 90分以上
- **良好**: 75-89分
- **一般**: 60-74分
- **较差**: 60分以下

## 使用场景示例

### 场景1：快速检测服务器性能

```bash
# 新购买的服务器，想快速了解性能
./build/perfassess -b all --web

# 在浏览器查看漂亮的报告
# http://localhost:8080
```

### 场景2：VPS 选购对比

```bash
# 测试 VPS A
./build/perfassess -b all --streaming -o vps_a.txt

# 测试 VPS B
./build/perfassess -b all --streaming -o vps_b.txt

# 对比两份报告，选择性价比更高的
```

### 场景3：网络质量检测

```bash
# 重点测试网络性能
./build/perfassess -b network --route-trace --streaming --web
```

### 场景4：CI/CD 集成

```bash
# 在 CI/CD 流程中自动测试
./build/perfassess -b all -v -o ci_report.txt
```

### 场景5：定期性能监控

```bash
# 每天定时运行，保存报告
./build/perfassess -b all -o "report_$(date +%Y%m%d).txt"
```

## 常见问题

### Q: 为什么某些测试被跳过？

A: 测试可能因以下原因被跳过：
- 磁盘空间不足（需要至少 1GB）
- 可用内存不足（需要至少 512MB）
- 网络连接不可用

### Q: 如何提高测试准确性？

A: 建议：
- 关闭其他占用资源的程序
- 在系统空闲时运行测试
- 多次运行取平均值

### Q: 支持哪些操作系统？

A: 目前支持：
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Q: 路由追踪和流媒体检测需要特殊权限吗？

A: 
- 路由追踪：在某些系统上可能需要管理员权限
- 流媒体检测：只需要网络连接，无需特殊权限

### Q: 流媒体检测支持哪些平台？

A: 目前支持：
- Netflix
- YouTube Premium
- Disney+
- HBO Max
- Amazon Prime Video

### Q: Web 报告服务器如何停止？

A: 按 `Ctrl+C` 即可优雅停止服务器

### Q: 可以同时保存文本报告和启用 Web 服务器吗？

A: 可以！使用：
```bash
./build/perfassess -b all -o report.txt --web
```

### Q: Web 服务器端口被占用怎么办？

A: 使用 `--port` 参数指定其他端口：
```bash
./build/perfassess -b all --web --port 9090
```

## 快速参考

### 常用命令速查表

| 需求 | 命令 |
|------|------|
| 交互式菜单 | `./build/perfassess` |
| 快速 CPU 测试 | `./build/perfassess -b cpu` |
| 完整测试 | `./build/perfassess -b all` |
| Web 报告 | `./build/perfassess -b all --web` |
| 保存报告 | `./build/perfassess -b all -o report.txt` |
| 路由追踪 | `./build/perfassess --route-trace` |
| 流媒体检测 | `./build/perfassess --streaming` |
| 完整功能 | `./build/perfassess -b all --route-trace --streaming --web -o report.txt` |

### 参数速查表

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--interactive` | `-i` | 交互式菜单 | `-i` |
| `--benchmarks` | `-b` | 检测项目 | `-b cpu,memory` |
| `--output` | `-o` | 输出文件 | `-o report.txt` |
| `--verbose` | `-v` | 详细输出 | `-v` |
| `--web` | - | Web 报告 | `--web` |
| `--port` | - | Web 端口 | `--port 9090` |
| `--route-trace` | - | 路由追踪 | `--route-trace` |
| `--streaming` | - | 流媒体检测 | `--streaming` |

### 检测项目选项

| 选项 | 说明 |
|------|------|
| `all` | 所有性能测试（默认） |
| `cpu` | CPU 性能测试 |
| `memory` | 内存性能测试 |
| `disk` | 磁盘性能测试 |
| `network` | 网络性能测试 |

## 许可证

[MIT License](LICENSE)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.0.0 (2025-11-03)
- ✅ 完整的性能测试功能
- ✅ 交互式菜单模式
- ✅ 路由追踪和流媒体检测
- ✅ Web 报告服务器
- ✅ 多平台支持

## 联系方式

如有问题或建议，请通过 Issue 联系我们。
