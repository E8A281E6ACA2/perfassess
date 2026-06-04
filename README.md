# 高性能多终端自动化性能评估系统

一款用于自动化评估计算机和服务器性能的工具。支持 CPU、内存、磁盘、网络等多项性能测试，提供清晰的性能报告和综合评分。

## 项目文档

- 项目分析与优化路线图: [docs/project-analysis-and-roadmap.md](docs/project-analysis-and-roadmap.md)
- 发布前检查清单: [docs/release-checklist.md](docs/release-checklist.md)

## 当前状态

当前项目已经具备可运行的 VPS 测评脚本雏形，支持一把梭性能测试、系统信息采集、扩展检测、多种报告输出和报告对比。

现阶段的目标是从“可运行 MVP”继续升级为成熟 VPS 测评脚本，重点提升以下三点：

- 主流 VPS 测评口径对齐，例如 sysbench、fio、iperf3、speedtest 和 Geekbench
- 默认一把梭的稳定性、可解释性和无依赖可运行能力
- 发布前验证、报告 schema、快照测试和 CI 的回归保护

更完整的分析和后续优化路线见 `docs/project-analysis-and-roadmap.md`。

## 快速开始

### 方式一：一把梭完整检测（默认）

```bash
# 1. 构建程序
make build

# 2. 运行默认完整检测（等价于 -b all）
./build/perfassess

# 3. 如需保存报告
./build/perfassess -o report.txt
```

默认无参数会直接执行 CPU、内存、磁盘、网络基础检测，适合脚本化和快速评估。

### 方式二：命令行指定检测项

```bash
# 快速 CPU 测试
./build/perfassess -b cpu

# 完整测试 + Web 报告
./build/perfassess -b all --web

# 完整测试 + 所有功能
./build/perfassess -b all --route-trace --streaming --web -o report.txt
```

### 方式三：交互式菜单（可选）

```bash
./build/perfassess -i
```

## ✨ 功能亮点

### 🎯 三种使用方式，满足不同需求

1. **命令行模式** - 默认一把梭执行，适合自动化脚本
2. **交互式菜单** - 中文界面，纯数字选择，作为可选模式保留
3. **Web 报告** - 美观的 HTML 界面，可视化展示（新功能）

### 🚀 核心功能

- ✅ **多维度性能测试**
  - CPU 性能测试（单核/多核）
  - 内存读写速度测试
  - 磁盘 I/O 性能测试
  - 网络延迟和带宽测试
  - 网络质量矩阵（IPv4/IPv6 可用性、TCP connect 延迟、抖动、失败率）
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
  - VPS 测评摘要，压缩展示系统、CPU、内存、磁盘、网络和置信度核心结果
  - CPU/内存多轮采样，输出结果波动指标
  - 网络上传估算时限制网络评分上限，避免不完整测速拿满分

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
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/high-performance-multi-terminal-automated-performance-evaluation-system/main/install.sh | bash
```

或使用 wget:
```bash
wget -qO- https://raw.githubusercontent.com/E8A281E6ACA2/high-performance-multi-terminal-automated-performance-evaluation-system/main/install.sh | bash
```

**Windows (PowerShell)**:
```powershell
# 下载最新版本
Invoke-WebRequest -Uri "https://github.com/E8A281E6ACA2/high-performance-multi-terminal-automated-performance-evaluation-system/releases/latest/download/perfassess.exe" -OutFile "perfassess.exe"
```

### 📦 方式二：下载预编译二进制文件

从 [Releases 页面](https://github.com/E8A281E6ACA2/high-performance-multi-terminal-automated-performance-evaluation-system/releases) 下载对应平台的文件：

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

确保已安装 Go 1.25.3 或更高版本。

```bash
# 克隆仓库
git clone https://github.com/E8A281E6ACA2/high-performance-multi-terminal-automated-performance-evaluation-system.git
cd high-performance-multi-terminal-automated-performance-evaluation-system

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
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/high-performance-multi-terminal-automated-performance-evaluation-system/main/uninstall.sh | bash

# 或手动删除
sudo rm /usr/local/bin/perfassess
```

## 使用方法

### 🎯 默认一把梭模式

直接运行程序，无需任何参数，会使用默认配置执行完整基础检测：

```bash
# 默认完整检测，等价于 -b all
./build/perfassess
```

默认会执行 CPU、内存、磁盘、网络基础性能测试。网络基础测试会额外输出无外部依赖的 TCP connect 质量矩阵，用于判断 IPv4/IPv6 可用性、延迟、抖动和失败率。外部依赖不会自动安装；如需使用 `sysbench`、`geekbench6`、`fio`、`iperf3`、`speedtest` 或路由追踪，请先运行 `check-deps` 查看提示。

### 🧭 交互式模式（可选）

如需菜单引导，可以显式启用交互式模式：

```bash
# 进入交互式菜单
./build/perfassess -i

# 或明确指定交互式模式
./build/perfassess --interactive
```

交互式菜单将引导您：
1. 选择检测项目（性能测试/路由追踪/流媒体检测/自定义组合）
2. 配置输出选项（详细模式/保存文件）
3. 确认配置后开始测试

### 📝 命令行模式（适合自动化）

命令行模式适合脚本自动化、CI/CD 集成或高级用户快速测试。

#### 基础用法

```bash
# 检查外部依赖（sysbench、geekbench6、fio、iperf3、speedtest、traceroute/tracert）
./build/perfassess check-deps

# 运行所有检测
./build/perfassess --benchmarks all
# 或使用简写
./build/perfassess -b all

# 或使用 make
make run
```

#### 预设模式

```bash
# 快速预设：CPU + 内存 + 磁盘内置测试，适合快速巡检
./build/perfassess --quick

# 完整预设：基础测试 + 路由追踪 + 流媒体 + AI 服务 + 安全体检
./build/perfassess --full

# VPS 测评预设：sysbench + fio + speedtest + VPS 评分基准 + 常用网络检查
./build/perfassess --vps-profile

# VPS 测评预设 + 自建 iperf3 服务端
./build/perfassess --vps-profile --iperf3-server 1.2.3.4:5201

# 完整预设 + 主流网络吞吐后端（需要 iperf3 服务端）
./build/perfassess --full --iperf3-server 1.2.3.4:5201

# 完整预设 + iperf3 多节点矩阵
./build/perfassess --full --iperf3-servers 1.2.3.4:5201,[2001:db8::1]:5201
```

`--vps-profile` 面向一把梭 VPS 测评，默认启用 `sysbench` CPU 后端、`sysbench` 内存后端、`fio` 磁盘后端、`speedtest` 网络后端、`vps` 评分基准、路由追踪和流媒体检测。它不会自动安装外部工具；建议先运行 `check-deps` 查看缺失项。使用 `speedtest` 时，报告会展示 Ookla 节点 ID、名称、地区、国家、Host、ISP、结果 URL、出口 IP 和 ping jitter。提供 `--iperf3-server`、`--iperf3-servers` 或 `--iperf3-server-file` 时，且未显式指定 `--network-backend`，会自动切换到 `iperf3` 网络后端。

`--full` 会优先使用 `sysbench` CPU 后端、`sysbench` 内存后端和 `fio` 磁盘后端；如果未安装依赖，程序会给出明确提示但不会自动安装。提供 `--iperf3-server`、`--iperf3-servers` 或 `--iperf3-server-file` 时，`--full` 会自动切换到 `iperf3` 网络后端。

#### 单项测试

```bash
# 只测试 CPU
./build/perfassess -b cpu

# 使用 sysbench 后端测试 CPU（需要预装 sysbench）
./build/perfassess -b cpu --cpu-backend sysbench

# 使用 Geekbench 6 后端测试 CPU（需要预装 geekbench6）
./build/perfassess -b cpu --cpu-backend geekbench

# 只测试内存
./build/perfassess -b memory

# 使用 sysbench 后端测试内存（需要预装 sysbench）
./build/perfassess -b memory --memory-backend sysbench

# 只测试磁盘
./build/perfassess -b disk

# 使用 fio 后端测试磁盘（需要预装 fio）
./build/perfassess -b disk --disk-backend fio

# 只测试网络
./build/perfassess -b network

# 默认网络测试会输出网络质量矩阵，无需安装额外工具

# 使用 iperf3 后端测试网络吞吐（需要预装 iperf3，并准备服务端）
./build/perfassess -b network --network-backend iperf3 --iperf3-server 1.2.3.4:5201

# 使用 iperf3 多节点矩阵测试网络吞吐（逗号分隔，支持 host:port 和 [IPv6]:port）
./build/perfassess -b network --network-backend iperf3 --iperf3-servers 1.2.3.4:5201,[2001:db8::1]:5201

# 使用 iperf3 节点文件测试网络吞吐（支持空行和 # 注释）
./build/perfassess -b network --network-backend iperf3 --iperf3-server-file docs/examples/iperf3-servers.txt

# 使用 Ookla Speedtest CLI 后端测试网络（需要预装 speedtest）
./build/perfassess -b network --network-backend speedtest
```

`speedtest` 后端会把 Ookla CLI 返回的测速节点和出口信息写入文本报告、Web 摘要、JSON `test_results.network_result.metrics` 和 `summary.vps_benchmark_summary.network`，便于判断本次网络结果来自哪个节点。

`--disk-backend fio` 会保留顺序读写与随机 IOPS 兼容字段，并额外输出 YABS 对齐的 4k / 64k / 512k / 1m mixed randrw 50/50 矩阵，便于和主流 VPS 测评结果横向对比。

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

# 输出机器可读 JSON
./build/perfassess -b all --output-format json -o report.json

# JSON summary.share_templates 会包含 plain_text 和 markdown 两种复制模板
# 文本报告也会输出“分享模板”段落，便于直接发帖或发给他人

# 对比两份 JSON 报告
./build/perfassess compare vps-a.json vps-b.json

# 输出机器可读对比结果
./build/perfassess compare vps-a.json vps-b.json --format json

# 把 JSON 报告加入本地历史库
./build/perfassess history add report.json

# 查看历史趋势
./build/perfassess history trend

# 批量排序目录中的 JSON 报告
./build/perfassess compare-dir ./reports --sort-by total

# 自定义综合评分权重
./build/perfassess -b all --score-weights cpu=0.4,memory=0.2,disk=0.2,network=0.2

# 选择评分基准档位（vps/server/workstation，默认 server）
./build/perfassess -b all --score-profile vps

# 详细输出模式
./build/perfassess -b all -v

# 组合使用
./build/perfassess -b all -v -o report.txt --web
```

### 命令行参数

```bash
perfassess [command] [flags]

Commands:
  check-deps               检查外部测试工具依赖
  compare                  对比两份 JSON 评估报告
  compare-dir              批量排序目录中的 JSON 评估报告
  history                  管理本地 JSON 报告历史库

Flags:
  -i, --interactive          启用交互式菜单模式
  -b, --benchmarks strings   指定要运行的检测项目 (cpu,memory,disk,network,all) (default [all])
      --quick                快速预设：只运行 CPU、内存、磁盘基础测试
      --full                 完整预设：运行基础测试并启用可选检查；提供 iperf3 服务端时使用 iperf3
      --vps-profile          VPS 测评预设：启用 sysbench/fio/speedtest、VPS 评分基准和常用网络检查
  -o, --output string        指定输出文件路径（不指定则只输出到控制台）
      --output-format string 指定输出格式 (text,json) (default "text")
      --score-weights string 综合评分权重，如 cpu=0.3,memory=0.2,disk=0.25,network=0.25
      --score-profile string 评分基准档位 (vps,server,workstation) (default "server")
  -v, --verbose              启用详细输出模式
      --log-level string     设置日志级别 (debug,info,warn,error) (default "info")
      --route-trace          启用路由追踪功能
      --streaming            启用流媒体解锁检测功能
      --ai-services          启用 AI 服务检测功能
      --stress               启用长时间压力测试
      --security             启用基础安全体检
      --network-backend string
                              网络测试后端 (builtin,iperf3,speedtest) (default "builtin")
      --iperf3-server string  iperf3 服务端地址（仅 network-backend=iperf3 时使用）
      --iperf3-servers strings
                              iperf3 多服务端地址列表，逗号分隔（仅 network-backend=iperf3 时使用）
      --iperf3-server-file string
                              iperf3 节点文件路径，支持空行和 # 注释（仅 network-backend=iperf3 时使用）
      --cpu-backend string    CPU 测试后端 (builtin,sysbench,geekbench) (default "builtin")
      --memory-backend string 内存测试后端 (builtin,sysbench) (default "builtin")
      --disk-backend string   磁盘测试后端 (builtin,fio) (default "builtin")
      --web                  启用 Web 报告服务器（新功能）
      --port int             Web 服务器端口 (default 8080)
  -h, --help                 显示帮助信息

注：--tests 参数仍然支持，但推荐使用 --benchmarks

提示：使用 `--route-trace` 时，请确保系统已安装 traceroute（Linux/macOS）或 tracert（Windows），否则将提示缺少依赖。
提示：使用 `--cpu-backend sysbench` 时，请提前安装 sysbench。程序只检测并提示，不会自动安装依赖。
提示：使用 `--cpu-backend geekbench` 时，请提前安装 Geekbench 6 并确认 `geekbench6` 可通过 PATH 访问。程序只检测并提示，不会自动安装依赖。
提示：使用 `--memory-backend sysbench` 时，请提前安装 sysbench。程序只检测并提示，不会自动安装依赖。
提示：使用 `--disk-backend fio` 时，请提前安装 fio。程序只检测并提示，不会自动安装依赖。
提示：使用 `--network-backend iperf3` 时，请提前安装 iperf3，并提供可访问的 `--iperf3-server`、`--iperf3-servers` 或 `--iperf3-server-file`。服务端可写为 `host`、`host:port` 或 `[IPv6]:port`，程序会把端口转换为 iperf3 的 `-p` 参数。节点文件支持空行、整行 `#` 注释和行尾注释。程序只检测并提示，不会自动安装依赖，也不会内置公共 iperf3 节点。
```

### 使用示例

#### 交互式模式示例

```bash
# 显式进入交互式菜单
./build/perfassess -i

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
#   8. AI 服务检测
#   9. 自定义组合
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
  网络质量矩阵: target | proto | available | avg ms | jitter ms | fail %

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

# 日常开发验证：格式、schema、测试、构建、CLI 冒烟
make validate

# 发布前验证：validate、跨平台构建、快速 JSON 报告
make release-check

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

- **语言**: Go 1.25.3+
- **CLI 框架**: [Cobra](https://github.com/spf13/cobra)
- **日志库**: [Zap](https://github.com/uber-go/zap)
- **系统信息**: [gopsutil](https://github.com/shirou/gopsutil)
- **配置管理**: 内置配置结构与 CLI 参数绑定

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

### 评分基准档位

默认使用 `server` 基准。可以通过 `--score-profile vps|server|workstation` 切换不同设备类型的内存、磁盘和网络基准线，避免 VPS、通用服务器和工作站使用同一套阈值导致评分解释失真。

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

默认网络测试已包含 TCP connect 质量矩阵，可直接观察 IPv4/IPv6 可用性、目标失败率和抖动；如需真实上传吞吐或多节点吞吐对比，再使用 `--network-backend iperf3` 或 `--network-backend speedtest`。

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
| 默认完整测试 | `./build/perfassess` |
| 交互式菜单 | `./build/perfassess -i` |
| 快速 CPU 测试 | `./build/perfassess -b cpu` |
| 完整测试 | `./build/perfassess -b all` |
| VPS 测评预设 | `./build/perfassess --vps-profile` |
| Web 报告 | `./build/perfassess -b all --web` |
| 保存报告 | `./build/perfassess -b all -o report.txt` |
| 报告对比 | `./build/perfassess compare vps-a.json vps-b.json` |
| 批量排序 | `./build/perfassess compare-dir ./reports --sort-by total` |
| 加入历史 | `./build/perfassess history add report.json` |
| 趋势分析 | `./build/perfassess history trend` |
| 路由追踪 | `./build/perfassess --route-trace`（需预装 traceroute/tracert） |
| 流媒体检测 | `./build/perfassess --streaming` |
| AI 服务检测 | `./build/perfassess --ai-services` |
| 压力测试 | `./build/perfassess --stress` |
| 安全体检 | `./build/perfassess --security` |
| 完整功能 | `./build/perfassess -b all --route-trace --streaming --web -o report.txt` |

### 参数速查表

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--interactive` | `-i` | 交互式菜单 | `-i` |
| `--benchmarks` | `-b` | 检测项目 | `-b cpu,memory` |
| `--quick` | - | 快速预设 | `--quick` |
| `--full` | - | 完整预设 | `--full` |
| `--vps-profile` | - | VPS 测评预设 | `--vps-profile` |
| `--output` | `-o` | 输出文件 | `-o report.txt` |
| `--output-format` | - | 输出格式 | `--output-format json` |
| `--verbose` | `-v` | 详细输出 | `-v` |
| `--web` | - | Web 报告 | `--web` |
| `--port` | - | Web 端口 | `--port 9090` |
| `--score-weights` | - | 综合评分权重 | `--score-weights cpu=0.4,memory=0.2,disk=0.2,network=0.2` |
| `--score-profile` | - | 评分基准档位 | `--score-profile vps` |
| `--cpu-backend` | - | CPU 测试后端 | `--cpu-backend geekbench` |
| `--memory-backend` | - | 内存测试后端 | `--memory-backend sysbench` |
| `--disk-backend` | - | 磁盘测试后端 | `--disk-backend fio` |
| `--network-backend` | - | 网络测试后端 | `--network-backend iperf3` |
| `--iperf3-server-file` | - | iperf3 节点文件 | `--iperf3-server-file docs/examples/iperf3-servers.txt` |
| `--store` | - | history 子命令历史库路径 | `history list --store ./history.jsonl` |
| `--sort-by` | - | compare-dir 排序字段 | `compare-dir ./reports --sort-by cpu` |
| `--route-trace` | - | 路由追踪 | `--route-trace` |
| `--streaming` | - | 流媒体检测 | `--streaming` |
| `--ai-services` | - | AI 服务检测 | `--ai-services` |
| `--stress` | - | 长时间压力测试 | `--stress` |
| `--security` | - | 安全体检 | `--security` |

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
