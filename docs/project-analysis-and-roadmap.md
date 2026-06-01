# 项目分析与优化路线

## 文档目的

本文档用于沉淀当前项目的阶段性判断，统一后续优化方向，避免讨论内容只停留在即时沟通中。

## 当前项目定位

当前项目已经具备较完整的产品雏形，覆盖以下能力：

- 命令行一把梭执行与可选交互式菜单
- 系统信息采集
- CPU、内存、磁盘、网络基础性能测试
- 路由追踪、流媒体检测、AI 服务可达性检测
- 压力测试、安全体检
- 终端报告与 Web 报告

从结构上看，项目已经不是单一脚本，而是一个模块化的综合评估工具。

## 当前效果评估

### 优点

- 主流程清晰，入口、控制器、测试模块、报告模块分层明确
- 支持功能较多，扩展点已经形成
- `go build ./...` 与 `go test ./...` 当前可以通过
- CLI 默认完整检测、可选交互式模式、Web 报告模式都已具备基本可用性

### 当前主要问题

#### 1. 结果可信度不足

当前最大问题不是功能数量，而是测试结果的可信度和解释性不足。

- CPU 基准测试容易快速打满分，区分度不够
- 网络上传速度当前为估算值，不是真实测速
- 各项评分基准值缺少真实样本校准
- 流媒体、AI 服务检测主要基于启发式网页响应判断，适合展示，不适合作为强结论

#### 2. 指标定义与展示未完全对齐

测试模块、评分模块、终端报告、Web 报告之间存在字段不完全一致的问题。

直接影响包括：

- 测试执行成功，但报告中关键指标可能为空
- 报告展示分数与实际测试结果不一致
- Web 报告与终端报告可能出现不同口径

这类问题会直接削弱用户对报告结果的信任。

#### 3. 缺少测试保障

虽然 `go test ./...` 可以执行成功，但项目当前没有真正的单元测试或集成测试文件，实际含义更接近“可以编译”，而不是“行为已验证”。

#### 4. 评分系统还比较经验化

当前评分更像一套初版经验规则：

- 权重固定
- 基准值固定
- 没有设备类型区分
- 没有置信区间和波动度输出

这会导致总分具备展示价值，但不够严谨。

## 当前结论

当前项目可以定义为：

“一个功能较完整、可运行、可展示的综合评估工具，但在基础性能测试可信度、评分一致性、结果解释性方面仍处于第一阶段。”

换句话说：

- 适合继续演示、迭代和内部使用
- 还不适合把综合评分当作强结论输出

## 优化原则

后续优化不建议继续优先堆功能，建议按照以下原则推进：

1. 先修结果一致性
2. 再修测试可信度
3. 最后再扩大功能覆盖面

如果基础数据层和评分层不稳，新增功能只会增加维护成本和误导风险。

## 使用形态决策

项目主路径采用 CLI 一把梭执行：

- 无参数默认执行完整基础检测，等价于 `-b all`
- 交互式菜单通过 `-i` 或 `--interactive` 显式启用
- 外部依赖只做检测和提示，不在评测流程中静默安装

这样更符合服务器评测工具的常见使用方式，也更适合脚本化、批量执行和复现实验结果。

## 优先级路线图

### 第一阶段：修正结果一致性

目标：让“测试结果是什么”和“报告展示什么”完全一致。

建议事项：

- 统一所有测试模块的 `Metrics` 字段定义
- 明确每个指标的类型、单位、含义
- 统一终端报告与 Web 报告读取字段
- 统一评分器与测试模块的数据契约
- 对“估算值”“降级执行”“部分失败”增加显式标记

完成标志：

- 同一轮测试在原始结果、终端报告、Web 报告中的关键指标一致
- 不再出现成功测试却无指标、或指标有值但得分为 0 的情况

当前已落地的统一方向：

- CPU:
  - `backend`
  - `single_core_source`
  - `single_core_score`
  - `single_core_score_samples`
  - `single_core_score_min`
  - `single_core_score_median`
  - `single_core_score_max`
  - `single_core_score_stddev`
  - `single_core_events_per_sec`
  - `single_core_events_per_sec_samples`
  - `single_core_events_per_sec_min`
  - `single_core_events_per_sec_median`
  - `single_core_events_per_sec_max`
  - `single_core_events_per_sec_stddev`
  - `multi_core_source`
  - `multi_core_score`
  - `multi_core_score_samples`
  - `multi_core_score_min`
  - `multi_core_score_median`
  - `multi_core_score_max`
  - `multi_core_score_stddev`
  - `multi_core_events_per_sec`
  - `multi_core_events_per_sec_samples`
  - `multi_core_events_per_sec_min`
  - `multi_core_events_per_sec_median`
  - `multi_core_events_per_sec_max`
  - `multi_core_events_per_sec_stddev`
  - `total_score`
  - `cpu_cores`
- Memory:
  - `backend`
  - `read_speed_mbps`
  - `read_speed_source`
  - `read_speed_mbps_samples`
  - `read_speed_mbps_min`
  - `read_speed_mbps_median`
  - `read_speed_mbps_max`
  - `read_speed_mbps_stddev`
  - `write_speed_mbps`
  - `write_speed_source`
  - `write_speed_mbps_samples`
  - `write_speed_mbps_min`
  - `write_speed_mbps_median`
  - `write_speed_mbps_max`
  - `write_speed_mbps_stddev`
  - `score`
  - `test_size_mb`
- Disk:
  - `backend`
  - `read_speed_mbps`
  - `write_speed_mbps`
  - `sequential_read_mbps`
  - `sequential_read_source`
  - `sequential_read_iops`
  - `sequential_read_latency_ms`
  - `sequential_read_latency_p95_ms`
  - `sequential_write_mbps`
  - `sequential_write_source`
  - `sequential_write_iops`
  - `sequential_write_latency_ms`
  - `sequential_write_latency_p95_ms`
  - `random_iops`
  - `random_iops_source`
  - `random_read_iops`
  - `random_write_iops`
  - `random_read_latency_ms`
  - `random_write_latency_ms`
  - `random_read_latency_p95_ms`
  - `random_write_latency_p95_ms`
  - `score`
- Network:
  - `latency_ms`
  - `average_latency_ms`
  - `latency_source`
  - `download_speed_mbps`
  - `download_speed_source`
  - `upload_speed_mbps`
  - `upload_speed_source`
  - `upload_speed_estimated`
  - `score`

说明：

- 报告层和评分层应优先读取统一字段，不再各自散落解析 `Metrics`
- 如存在兼容字段，应只作为过渡读取逻辑，而不应继续扩散
- CPU 和内存当前使用 3 轮采样，中位数作为兼容主指标，`stddev` 用于解释结果波动
- 当前网络测试中：
  - 延迟为 TCP connect 近似值，使用 `latency_source=tcp_connect` 标记
  - 下载为 HTTP 下载测速，使用 `download_speed_source=http_download` 标记
  - 上传在缺乏可靠公共端点时使用估算值，使用 `upload_speed_estimated=true` 和 `upload_speed_source=estimated_from_download` 标记
  - 估算上传值不应与真实上传测速等价对待，评分时不参与真实上传分，且网络评分上限为 85

### 第二阶段：提升测试可信度

目标：让基础性能测试结果更稳定、更可复现。

建议事项：

- CPU 测试改为更稳定的吞吐型指标，并增加多轮采样
- 内存测试增加预热、多轮执行、块大小控制
- 磁盘测试区分顺序读写与随机读写，并降低 page cache 干扰
- 网络测试明确真实测速与估算模式，取消写死上传值
- 输出 `min / median / max / stddev` 等统计信息

完成标志：

- 同一机器多次运行结果波动可控
- 报告中可以解释结果稳定性，而非只给单次数值

### 第三阶段：改进评分系统

目标：让评分从“经验分”升级为“可解释分”。

建议事项：

- 原始指标与评分并列展示
- 对不同设备类型引入不同基准线
- 允许评分权重配置化
- 增加评分说明和计算依据
- 为“未完成全部测试”的场景设计更明确的评分策略

完成标志：

- 用户可以理解某项得分为何高或低
- 总分与分项表现之间逻辑一致

当前已落地的评分解释方向：

- 新增 `score_weights` 配置和 `--score-weights` CLI 参数
- 权重包括 `cpu`、`memory`、`disk`、`network`，未执行或失败的测试不参与总分归一化
- 报告摘要新增 `score_breakdown`
- `score_breakdown` 记录每个分项的原始指标、基准线、子项分、权重和计算说明
- 文本报告会展示简明评分说明，JSON 报告保留结构化计算依据

### 第四阶段：补充工程保障

目标：让项目具备持续迭代能力。

建议事项：

- 为评分器、报告生成器、配置校验增加单元测试
- 为基础测试流程增加集成测试或 mock 测试
- 建立示例报告快照，防止输出格式回归
- 引入 lint、test、build 的 CI 流程

完成标志：

- 关键模块修改后可以自动验证
- 文档、输出、行为具备基本回归保护

当前已开始的保障工作：

- 已为 `internal/reporter` 补充第一批单元测试
- 已为 `internal/tests/network.go` 补充第一批行为测试
- 已增加 GitHub Actions CI，覆盖 `go test ./...` 与 `go build ./...`
- CI 已补充 CLI help、依赖检查和 JSON 示例报告契约 smoke
- 已建立文本报告快照测试，防止核心输出格式无意回归
- 已建立 JSON 示例报告契约测试，保护 `benchmark_profile`、`confidence_level` 和 `score_breakdown`
- 报告摘要已增加 `quality_notes`，用于记录估算、降级、未执行和波动风险
- 输出层已支持 `--output-format json`，便于自动化采集和批量对比
- 已增加 JSON 示例契约测试和文本报告快照测试，防止报告字段、评分说明和质量提示静默回归
- 首批覆盖重点为：
  - 网络评分在“真实上传”和“估算上传”场景下的差异
  - 终端报告对网络来源和估算标记的展示
  - Web 报告关键指标对估算上传的提示
  - 综合评分在部分测试执行场景下的权重归一化
  - 摘要与完整报告在“未完成”场景下的输出语义
  - 网络测试中上传估算分支与评分降权逻辑
  - `NetworkTest.Execute()` 在成功与降级场景下的结果字段写入
  - JSON 报告输出结构
  - 报告质量提示生成逻辑
  - JSON 示例报告关键字段契约
  - 文本报告完整输出快照

## 推荐近期执行顺序

建议按以下顺序推进：

1. 统一 `Metrics` schema，并修复报告字段对齐问题
2. 修复基础测试结果展示不全的问题
3. 为网络测试补充真实上传/降级标识机制
4. 重构 CPU、内存、磁盘基准逻辑
5. 为评分和报告增加测试覆盖

当前正在推进的下一阶段方向：

- 将 network 结果从通用 `Metrics map` 逐步收敛到 typed model
- 保留对现有报告层的兼容映射，先完成“结构化上游，兼容下游”
- 后续为接入 `iperf3` 等主流网络基准预留统一入口
- CLI 侧保留一把梭默认路径，同时提供 `--quick` 和 `--full` 预设，方便批量巡检与完整评测

当前已落地的网络后端边界：

- `NetworkMetrics` 已作为网络测试的 typed model
- `NetworkBenchmarkBackend` 已作为网络评测后端接口
- `BuiltinNetworkBackend` 继续承载当前内置测试逻辑
- `Iperf3NetworkBackend` 已具备最小命令执行与 JSON 结果解析骨架
- 配置层已预留 `network_backend` 与 `iperf3_server`
- CLI 已预留 `--network-backend` 与 `--iperf3-server`
- `iperf3` 后端会检测本机是否安装 `iperf3`，缺失时返回安装提示，不自动安装
- 网络测试失败原因会写入 `network_error` 并展示在终端/Web 报告中
- 网络报告会展示 `backend`，并在 `iperf3` 场景展示服务端地址
- 网络测速来源字段会跟随 backend，例如 `iperf3_download` 与 `iperf3_upload`
- `iperf3` 后端只负责吞吐测试，延迟继续回退到内置 TCP connect 测量并使用 `latency_source=tcp_connect`
- 后续真正启用 `iperf3` 前，需要补充真实服务端验证和报告展示

当前已落地的磁盘后端边界：

- `DiskBenchmarkBackend` 已作为磁盘评测后端接口
- `BuiltinDiskBackend` 继续承载当前内置文件读写与随机 IOPS 测试逻辑
- `FioDiskBackend` 已具备最小命令执行与 JSON 结果解析骨架
- 配置层已预留 `disk_backend`
- CLI 已预留 `--disk-backend builtin|fio`
- `fio` 后端会检测本机是否安装 `fio`，缺失时返回安装提示，不自动安装
- 磁盘报告会展示 `backend`，并继续输出顺序读写和随机 IOPS
- 磁盘测速来源字段会跟随 backend，例如 `fio`

`fio` 使用约定：

- 用户需要显式选择 `--disk-backend fio`
- 被测机器需要预先安装 `fio`
- 程序只负责检测并提示安装方式，不静默修改系统环境
- 默认仍使用内置 `builtin` 后端，避免无依赖场景下破坏一把梭执行

`iperf3` 使用约定：

- 用户需要显式选择 `--network-backend iperf3`
- 用户需要提供 `--iperf3-server <host>` 或 `--iperf3-server <host:port>`
- 如果服务端包含端口，程序会转换为 `iperf3 -c <host> -p <port>`，避免把 `host:port` 错传给 `-c`
- 被测机器需要预先安装 `iperf3`
- 程序只负责检测并提示安装方式，不静默修改系统环境

## 下一批对标设计：主流基准后端增强

目标：把 CPU 和磁盘从“项目内置经验测试”进一步对齐到主流服务器测评常用工具链。

### CPU 后端设计

- 新增 `CPUBackend` 配置，可选值为 `builtin` 与 `sysbench`
- 默认继续使用 `builtin`，确保无依赖一把梭可运行
- `--full` 预设优先使用 `sysbench`，用于更接近主流测评的 CPU 吞吐结果
- `sysbench` 后端执行两类测试：
  - 单线程：`sysbench cpu --threads=1 --time=10 run`
  - 多线程：`sysbench cpu --threads=<runtime.NumCPU()> --time=10 run`
- 解析 `events per second` 作为原始吞吐指标
- 继续输出兼容字段：
  - `single_core_score`
  - `multi_core_score`
  - `total_score`
- 新增 CPU 来源字段：
  - `backend`
  - `single_core_events_per_sec`
  - `single_core_events_per_sec_samples`
  - `single_core_events_per_sec_min`
  - `single_core_events_per_sec_median`
  - `single_core_events_per_sec_max`
  - `single_core_events_per_sec_stddev`
  - `multi_core_events_per_sec`
  - `multi_core_events_per_sec_samples`
  - `multi_core_events_per_sec_min`
  - `multi_core_events_per_sec_median`
  - `multi_core_events_per_sec_max`
  - `multi_core_events_per_sec_stddev`
  - `single_core_source`
  - `multi_core_source`

### 磁盘 fio 详细指标设计

- 保持 `DiskBenchmarkBackend` 接口兼容，不破坏现有报告与评分
- `FioDiskBackend` 在执行顺序读、顺序写、随机读写时缓存完整解析结果
- 除现有兼容字段外，补充：
  - `sequential_read_iops`
  - `sequential_read_latency_ms`
  - `sequential_read_latency_p95_ms`
  - `sequential_write_iops`
  - `sequential_write_latency_ms`
  - `sequential_write_latency_p95_ms`
  - `random_read_iops`
  - `random_write_iops`
  - `random_read_latency_ms`
  - `random_write_latency_ms`
  - `random_read_latency_p95_ms`
  - `random_write_latency_p95_ms`
- 报告层优先展示兼容关键指标，质量提示中说明 fio 后端可提供更高可信度
- 解析失败时必须返回明确错误，不静默降级为 0

## 当前批次设计：主流评测闭环

目标：补齐内存主流基准、报告可信度总结和 JSON schema 文档，使报告不仅给出结果，也说明本次结果的可信程度。

### 内存后端设计

- 新增 `MemoryBackend` 配置，可选值为 `builtin` 与 `sysbench`
- 默认继续使用 `builtin`，保证无依赖默认一把梭可运行
- `--full` 预设使用 `sysbench` 内存后端
- `sysbench` 内存后端执行：
  - 读取：`sysbench memory --memory-oper=read --memory-block-size=1M --memory-total-size=<size>M run`
  - 写入：`sysbench memory --memory-oper=write --memory-block-size=1M --memory-total-size=<size>M run`
- 解析 `MiB/sec` 作为内存吞吐指标
- 输出字段：
  - `backend`
  - `read_speed_mbps`
  - `read_speed_source`
  - `write_speed_mbps`
  - `write_speed_source`
  - `score`
  - `test_size_mb`

### 可信度闭环设计

- 报告摘要新增 `benchmark_profile`
  - `name`
  - `cpu_backend`
  - `memory_backend`
  - `disk_backend`
  - `network_backend`
  - `mainstream_count`
- 报告摘要新增 `confidence_level`
  - `level`: `high`、`medium`、`low`
  - `reasons`: 降级、估算、缺失或未执行原因
- `name` 当前包括 `quick`、`default`、`full`、`full_iperf3` 和 `custom`
- 文本报告展示评测档位、后端组合和置信等级
- JSON schema 文档记录关键字段、单位、来源和估算语义

### 验证要求

- 新增配置校验与 CLI 参数测试
- 新增 `sysbench` 输出解析测试
- 新增 `fio` 详细 JSON 解析测试
- 必须通过：
  - `go test ./...`
  - `go build ./...`
  - `go run ./cmd --help`
  - `go run ./cmd check-deps`

## 文档维护约定

后续如有以下变化，应同步更新本文档：

- 新增或删除测试模块
- 调整评分模型
- 修改报告输出结构
- 重新定义项目阶段目标

建议在每完成一个阶段后，更新本文件中的“当前结论”和“优先级路线图”。
