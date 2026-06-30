# 项目分析与优化路线

## 文档目的

本文档用于沉淀当前项目的阶段性判断，统一后续优化方向，避免讨论内容只停留在即时沟通中。

主流 VPS 测评工具的对标分析见 [主流 VPS 测评程序对标分析](mainstream-vps-benchmark-analysis.md)。后续路线应优先吸收其中的主流口径、报告结构和证据化输出原则。

## 当前项目定位

当前项目已经具备可运行的 VPS 测评脚本雏形，覆盖以下能力：

- 命令行一把梭执行与可选交互式菜单
- 系统信息采集
- CPU、内存、磁盘、网络基础性能测试
- sysbench、Geekbench、fio、iperf3、speedtest 等主流后端接入边界
- 网络质量矩阵、fio mixed 矩阵、iperf3 多节点矩阵
- VPS 测评摘要、JSON 报告、文本报告、报告对比和批量排序
- 路由追踪、流媒体检测、AI 服务可达性检测
- 压力测试、安全体检
- 终端报告与 Web 报告

从结构上看，项目已经不是单一脚本，而是一个模块化 VPS/服务器评估工具。下一阶段重点不是“能不能运行”，而是把输出质量、发布流程和主流测评对齐程度继续做成熟。

## 当前效果评估

### 优点

- 主流程清晰，入口、控制器、测试模块、报告模块分层明确
- 支持默认一把梭，外部依赖缺失时只提示不静默安装
- `go build ./...`、`go test ./...`、CLI smoke 和 JSON 契约校验已经固化
- CLI 默认完整检测、可选交互式模式、Web 报告模式都具备基本可用性
- 报告已包含 `benchmark_profile`、`confidence_level`、`score_breakdown` 和 `quality_notes`
- 已建立 CI、Release workflow、文本报告快照、JSON schema 和发布前检查清单

### 当前主要问题

#### 1. 主流 VPS 测评口径仍需继续增强

当前最大问题不是功能数量，而是需要进一步贴近成熟 VPS 测评脚本常见口径。

- 默认内置 CPU/内存/磁盘测试仍偏轻量，区分度不如外部主流基准
- 内置网络上传仍是估算值，真实上传需要 `iperf3` 或 `speedtest`
- `iperf3` 多节点能力已具备，但还缺默认推荐节点池和真实服务端验证策略
- 各项评分基准值仍需要更多真实 VPS 样本校准
- 流媒体、AI 服务检测主要基于启发式网页响应判断，适合展示，不适合作为强结论

#### 2. 输出契约已初步对齐，但仍需持续维护

测试模块、评分模块、终端报告、Web 报告之间已建立核心字段契约，但新增功能仍必须同步维护 schema、文档和快照。

- 新增报告字段必须同步 `docs/report.schema.json` 和 `docs/report-schema.md`
- 示例 JSON 和文本快照需要覆盖关键输出
- Web 报告和终端报告仍需要跟随后续矩阵能力继续增强

#### 3. 工程保障已经建立，但发布质量还要继续收紧

当前已经具备单元测试、报告契约测试、快照测试、CI 和 release workflow。后续需要继续补齐跨平台真实运行验证和外部依赖后端实测。

#### 4. 评分系统仍需要样本校准

当前评分已经支持 profile、权重归一化、降级提示和置信度，但基准线仍需要更多真实机器数据校准。

- VPS、通用服务器、工作站 profile 仍需要样本回测
- 主流后端与内置后端的分数映射仍需进一步校准
- 长时间稳定性、抖动、失败率尚未进入综合评分

这会导致总分已经具备横向参考价值，但还不能作为强排名结论。

## 当前结论

当前项目可以定义为：

“一个可运行、可发布、具备主流后端雏形的 VPS 测评脚本，已经适合内部使用和小范围试用；距离成熟公开测评脚本还需要继续增强主流口径、样本校准和真实环境验证。”

换句话说：

- 适合快速运行、生成报告、做同口径对比和趋势记录
- 默认无外部依赖路径可运行，外部后端用于提升可信度
- 综合评分可作为参考，但成熟版本还需要更多样本校准

## 优化原则

后续优化不建议无序堆功能，建议按照以下原则推进：

1. 保持默认一把梭稳定可运行
2. 优先对齐主流 VPS 测评口径
3. 每个新增指标必须同步 schema、文档和测试
4. 每个版本必须通过发布前检查

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
- 新增 `score_profile` 配置和 `--score-profile` CLI 参数，当前支持 `vps`、`server`、`workstation`
- 权重包括 `cpu`、`memory`、`disk`、`network`，未执行或失败的测试不参与总分归一化
- 评分基准线从硬编码迁移到 profile，内存、磁盘、网络会按不同设备场景使用不同基准
- 报告摘要新增 `score_calibration`，记录校准版本、当前档位基准线、完整 profile 基准线和等级阈值
- 报告摘要新增 `score_breakdown`
- `score_breakdown` 记录每个分项的原始指标、基准线、子项分、权重、校准版本和计算说明
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

当前已落地的保障工作：

- 已为 `internal/reporter` 补充第一批单元测试
- 已为 `internal/tests/network.go` 补充第一批行为测试
- 已增加 GitHub Actions CI，覆盖 `go test ./...` 与 `go build ./...`
- CI 已补充 CLI help、依赖检查和 JSON 示例报告契约 smoke
- 已新增 `make validate`，统一本地日常验证入口
- 已新增 `make release-check`，统一发布前验证入口并覆盖发布二进制冒烟、跨平台构建和 JSON 报告契约
- 已新增发布二进制端到端冒烟脚本，覆盖版本命令、help、依赖检查、快速 JSON 报告、报告对比和目录批量排序链路
- Release workflow 会在上传产物前对 Linux amd64 发布二进制执行端到端冒烟测试
- 已新增 [发布前检查清单](release-checklist.md)
- 已建立文本报告快照测试，防止核心输出格式无意回归
- 已建立 JSON 示例报告契约测试，保护 `benchmark_profile`、`confidence_level` 和 `score_breakdown`
- 已新增机器可读 `docs/report.schema.json`，并用测试校验示例报告与生成报告
- CI 和 Release workflow 已改为读取 `go.mod` 中的 Go 版本，避免发布构建环境与源码版本脱节
- README 与安装脚本已修正为当前 GitHub 仓库地址，避免一键安装继续指向旧仓库或占位地址
- 报告摘要已增加 `quality_notes`，用于记录估算、降级、未执行和波动风险
- 输出层已支持 `--output-format json`，便于自动化采集和批量对比
- 已增加 JSON 示例契约测试和文本报告快照测试，防止报告字段、评分说明和质量提示静默回归
- 已新增报告/Web 契约 smoke，校验测评结论、关键证据、模块可信度、分享模板和 Web 首屏决策面板
- 已新增校准样本汇总 smoke，校验脱敏样本汇总、筛选条件、candidate/formal 数据集准入失败路径
- 已新增报告产物清单校验 smoke，校验报告目录、压缩包内容、目录篡改和压缩包内部篡改均能被 SHA256 发现
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
  - 两份 JSON 报告的分项对比与评分基准一致性检查

## 推荐近期执行顺序

当前收尾批次目标：

1. 增加真实 VPS 验收脚本，保留验收产物并检查核心 JSON 报告契约
2. 增加真实 VPS 验收文档，覆盖必过命令、可选依赖、iperf3 授权节点和失败处理
3. 增加脱敏归档约定，避免真实 VPS 样本泄露公网 IP、ISP 和地理位置
4. 将验收流程接入 README 和发布前检查清单

下一批增强建议：

1. 收集更多真实 VPS 样本，基于 `score_calibration.version` 做下一轮阈值回测
2. 使用 `python3 scripts/calibration-summary.py` 汇总脱敏样本，按 `docs/calibration-dataset-policy.md` 检查数据集准入状态，再基于 P50/P75/P90 调整下一版评分校准阈值

当前已落地的 VPS 测评摘要边界：

- 新增 `--vps-profile`，一键启用 `all` 测试、`vps` 评分基准、`sysbench` CPU、`sysbench` 内存、`fio` 磁盘、`speedtest` 网络、路由追踪和流媒体检测
- `--vps-profile` 不启用 AI 服务检测、安全体检和长时间压力测试，避免默认 VPS 测评包含耗时或偏运维巡检的项目
- `--vps-profile` 提供 `--iperf3-server`、`--iperf3-servers` 或 `--iperf3-server-file` 时会自动切换到 `iperf3` 网络后端；显式传入 `--network-backend` 时以用户参数为准
- `speedtest` 后端会输出 Ookla 节点 ID、名称、地区、国家、Host、ISP、结果 URL、出口 IP 和 ping jitter
- 文本报告顶部新增“VPS测评摘要”，集中展示系统、CPU、内存、磁盘、网络、网络质量、总分、等级、置信度和评分基准
- JSON `summary` 新增 `vps_benchmark_summary`，用于脚本快速读取核心测评结果
- JSON `summary` 新增 `share_templates`，提供 `plain_text` 和 `markdown` 两种复制模板
- 文本报告新增“分享模板”，直接输出适合复制的多行纯文本摘要
- `vps_benchmark_summary` 保留 `system`、`cpu`、`memory`、`disk`、`network`、`scores`、`confidence` 七个核心分组
- 该摘要只压缩常用结果，不替代完整 `test_results` 和原始 metrics

当前已落地的报告对比边界：

- 新增 `perfassess compare <report-a.json> <report-b.json>` 子命令
- 支持 `--format text|json`
- 输出总分、CPU、内存、磁盘、网络分项差异
- 输出差值、百分比变化和胜出方
- 如果两份报告 `score_profile` 不一致，会标记为不可直接比较
- 如果至少一份报告未完成，会输出质量提示
- 新增 `perfassess compare-dir <reports-dir>` 子命令
- `compare-dir` 支持按 `total`、`cpu`、`memory`、`disk`、`network` 排序
- `compare-dir` 支持 `--format text|json`

当前已收敛的历史追踪边界：

- 已移除本地持久历史库和 `history add/list/trend` 子命令，避免默认工具写入长期追踪数据
- 保留 `compare-dir` 作为一次性目录内排序能力，不写入 `~/.perfassess` 或其他历史库
- 后续若需要趋势能力，应优先交给外部平台或用户显式管理的数据目录，而不是 CLI 默认内置追踪

当前已落地的网络后端边界：

- `NetworkMetrics` 已作为网络测试的 typed model
- `NetworkBenchmarkBackend` 已作为网络评测后端接口
- `BuiltinNetworkBackend` 继续承载当前内置测试逻辑
- `Iperf3NetworkBackend` 已具备最小命令执行与 JSON 结果解析骨架
- `SpeedtestNetworkBackend` 已接入 Ookla Speedtest CLI JSON 输出，用于更接近主流 VPS 脚本的公网测速口径
- 配置层已预留 `network_backend`、`iperf3_server`、`iperf3_servers` 与 `iperf3_server_file`
- CLI 已支持 `--network-backend builtin|iperf3|speedtest`、`--iperf3-server`、`--iperf3-servers` 与 `--iperf3-server-file`
- `iperf3` 后端会检测本机是否安装 `iperf3`，缺失时返回安装提示，不自动安装
- `iperf3` 后端支持多服务端矩阵和节点文件，输出每个节点的协议族、延迟、下载、上传和错误信息
- `speedtest` 后端会检测本机是否安装 Ookla Speedtest CLI，缺失时返回安装提示，不自动安装
- 网络测试失败原因会写入 `network_error` 并展示在终端/Web 报告中
- 网络报告会展示 `backend`，并在 `iperf3` 场景展示服务端地址
- 网络测速来源字段会跟随 backend，例如 `iperf3_download`、`iperf3_upload`、`speedtest_download` 与 `speedtest_upload`
- 内置下载测速已支持多个 HTTP 下载源顺序重试，成功时 `download_speed_source` 记录实际 URL，避免单个公共测速源故障直接导致网络降级
- 默认网络测试已增加 TCP connect 网络质量矩阵，输出 IPv4/IPv6 可用性、目标失败率、平均延迟和抖动
- 网络质量矩阵作为诊断指标写入 `network_quality_*`，不会因为 IPv6 不通而单独拉低网络评分或测试状态
- `iperf3` 后端只负责吞吐测试，延迟继续回退到内置 TCP connect 测量并使用 `latency_source=tcp_connect`
- `iperf3` 不内置公共节点；用户需要提供自有或授权节点，避免依赖不稳定或未授权的公共服务

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
- 用户需要提供 `--iperf3-server <host>`、`--iperf3-server <host:port>`、`--iperf3-servers <host:port,[IPv6]:port>` 或 `--iperf3-server-file <path>`
- 如果服务端包含端口，程序会转换为 `iperf3 -c <host> -p <port>`，避免把 `host:port` 错传给 `-c`
- 节点文件每行一个服务端，支持空行、整行 `#` 注释和行尾注释
- 被测机器需要预先安装 `iperf3`
- 程序只负责检测并提示安装方式，不静默修改系统环境

默认网络质量矩阵使用约定：

- 默认随网络测试执行，不需要安装额外工具
- 使用 TCP connect 采样，不依赖 ICMP 权限或 ping 工具
- 默认覆盖 IPv4 与 IPv6 公共目标，每个目标采样多次
- 输出字段包括 `network_quality_profile`、`network_quality_target_count`、`network_quality_ipv4_available`、`network_quality_ipv6_available`、`network_quality_failure_rate`、`network_quality_avg_latency_ms`、`network_quality_jitter_ms` 和 `network_quality_<n>_*`
- 该矩阵用于辅助判断连通性、抖动和失败率，不替代 `iperf3` 或 `speedtest` 的真实吞吐测试

## 下一批对标设计：主流基准后端增强

目标：把 CPU 和磁盘从“项目内置经验测试”进一步对齐到主流服务器测评常用工具链。

### CPU 后端设计

- 新增 `CPUBackend` 配置，可选值为 `builtin`、`sysbench` 与 `geekbench`
- 默认继续使用 `builtin`，确保无依赖一把梭可运行
- `--full` 预设优先使用 `sysbench`，用于更接近主流测评的 CPU 吞吐结果
- `sysbench` 后端执行两类测试：
  - 单线程：`sysbench cpu --threads=1 --time=10 run`
  - 多线程：`sysbench cpu --threads=<runtime.NumCPU()> --time=10 run`
- 解析 `events per second` 作为原始吞吐指标
- `geekbench` 后端执行 `geekbench6 --no-upload --export-json <path>`，一次运行同时解析单核与多核原始分
- `geekbench` 后端会缓存一次运行结果，避免多轮采样把 Geekbench 重复执行多次
- `geekbench` 原始分保存在 `single_core_raw_score` 与 `multi_core_raw_score`，`single_core_score` 与 `multi_core_score` 继续保持 0-100 归一化评分
- `geekbench6` 属于可选外部依赖，程序只检测并提示安装，不自动安装
- 继续输出兼容字段：
  - `single_core_score`
  - `multi_core_score`
  - `total_score`
- 新增 CPU 来源字段：
  - `backend`
  - `single_core_raw_score`
  - `single_core_raw_score_samples`
  - `single_core_raw_score_min`
  - `single_core_raw_score_median`
  - `single_core_raw_score_max`
  - `single_core_raw_score_stddev`
  - `single_core_events_per_sec`
  - `single_core_events_per_sec_samples`
  - `single_core_events_per_sec_min`
  - `single_core_events_per_sec_median`
  - `single_core_events_per_sec_max`
  - `single_core_events_per_sec_stddev`
  - `multi_core_raw_score`
  - `multi_core_raw_score_samples`
  - `multi_core_raw_score_min`
  - `multi_core_raw_score_median`
  - `multi_core_raw_score_max`
  - `multi_core_raw_score_stddev`
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
- `FioDiskBackend` 新增 YABS 对齐 mixed 矩阵，块大小为 `4k`、`64k`、`512k`、`1m`，口径为 `randrw` 50/50
- 除现有兼容字段外，补充：
  - `sequential_read_iops`
  - `sequential_read_latency_ms`
  - `sequential_read_latency_p95_ms`
  - `sequential_write_iops`
  - `sequential_write_latency_ms`
  - `sequential_write_latency_p95_ms`
  - `random_read_iops`
  - `random_write_iops`
  - `random_read_mbps`
  - `random_write_mbps`
  - `random_read_latency_ms`
  - `random_write_latency_ms`
  - `random_read_latency_p95_ms`
  - `random_write_latency_p95_ms`
  - `fio_mixed_profile`
  - `fio_mixed_block_sizes`
  - `fio_mixed_<block>_read_mbps`
  - `fio_mixed_<block>_write_mbps`
  - `fio_mixed_<block>_total_mbps`
  - `fio_mixed_<block>_read_iops`
  - `fio_mixed_<block>_write_iops`
  - `fio_mixed_<block>_total_iops`
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
- `name` 当前包括 `quick`、`default`、`full`、`full_iperf3`、`full_speedtest` 和 `custom`
- 文本报告展示评测档位、后端组合和置信等级
- JSON schema 文档记录关键字段、单位、来源和估算语义

### 验证要求

- 新增配置校验与 CLI 参数测试
- 新增 `sysbench` 输出解析测试
- 新增 `geekbench6` JSON 解析、缺失依赖和缓存测试
- 新增 `fio` 详细 JSON 解析测试
- 日常开发必须通过 `make validate`
- 发布前必须通过 `make release-check`

## 文档维护约定

后续如有以下变化，应同步更新本文档：

- 新增或删除测试模块
- 调整评分模型
- 修改报告输出结构
- 重新定义项目阶段目标

建议在每完成一个阶段后，更新本文件中的“当前结论”和“优先级路线图”。

## 下一阶段设计：路由边界、证据化报告与发布收口

当前阶段不实现远端探针、回调服务或公共探针池。原因是这类能力需要额外节点授权、安全隔离、滥用防护和运维成本；如果没有这些前提，容易把本机出站路径误包装成“真实回程”，反而降低报告可信度。

后续路线改为：

- 当前 `--route-trace` 和国内方向参考只标记为本机出站路径，不称为真实回程。
- 自动报告保留 `backroute_trace.json` 兼容产物，但文件内必须说明它不是第三方真实回程。
- Web、Markdown、控制台报告都应把路由证据、限制和建议分开展示。
- 如果未来重新评估真实回程能力，应先完成安全设计和授权节点方案，再作为显式可选插件或外部集成，不进入默认一把梭。

当前证据化报告已经进入验证链路，下一步优先实施顺序调整为：

- 在真实 VPS 上执行 `PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh`，确认低配降级、标准完整报告和失败摘要都稳定。
- 收集多台服务器的 `calibration_sample.json`，使用 `python3 scripts/calibration-summary.py /path/to/samples --require-score-profiles vps,server,workstation --require-policy candidate` 做候选校准观察。
- 按 [真实 VPS 校准样本矩阵](calibration-sampling-matrix.md) 分批采集 low-memory、small-vps、standard-vps、high-vps、arm64 和 IPv6 网络样本，避免样本只代表单一机器类型。
- 继续把控制台、Markdown 和 Web 报告中的“结论 + 证据 + 限制 + 建议”保持为同一语义，避免某个报告入口信息缩水。
- full/mainstream 档位发布前必须额外验证 sysbench、fio、speedtest 或授权 iperf3 节点，不把缺少外部后端的结果包装成高置信结论。
- VPS 验收脚本应区分核心报告失败、可选后端成功和可选后端可解释跳过；非严格模式保留降级报告，严格模式阻断发布。

## 企业级推进路线

对标 YABS、IPQuality 和 RegionRestrictionCheck 后，后续推进顺序调整为“先可信，再丰富”。项目已经覆盖大量功能，继续堆模块的收益低于建立结果可信闭环。

### 当前最优先：发布基线收口

当前工作区包含报告证据摘要、JSON schema、自动化验收、校准汇总和发布门禁等大批改动。下一步应先完成验证并按主题提交，形成一个可发布基线。

当前基线建议作为一组可信度增强提交收口：

- `feat(report): 增强外部检测证据可信度`
- 覆盖范围包括 IP 质量、流媒体、AI、路由、主流外部后端错误分类、VPS 验收摘要、校准 profile 门禁和发布手册同步。
- 历史追踪和真实回程探针已经通过项目不变量保护，不应在本批次恢复。

提交前最低验证：

```bash
git status --short --branch
git diff --check
bash -n install.sh uninstall.sh scripts/*.sh
python3 -m py_compile scripts/perfassess-progress-server.py scripts/calibration-summary.py scripts/verify-artifacts.py
make project-invariants-smoke
make validate
make pre-commit
```

发布前额外验证：

```bash
make release-check
PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh
```

### 第二优先：真实 VPS 校准样本

评分系统已经具备 profile、权重、置信度和校准版本，但仍需要真实样本证明阈值合理。下一步应开始有计划地收集脱敏样本：

- 低配样本：512MB-1GB 内存、1C VPS，验证低资源降级和跳过解释。
- 常规样本：2C-4C、2GB-8GB 内存，作为 server/vps 基准主力。
- 高配样本：更多核心、更高磁盘和网络，用于观察评分上限。
- 网络样本：不同地区、IPv4/IPv6、不同运营商，验证网络质量和 IP 质量报告。

样本汇总命令：

```bash
python3 scripts/calibration-summary.py /path/to/samples \
  --require-score-profiles vps,server,workstation \
  --require-policy candidate \
  --min-confidence medium \
  --format markdown \
  -o calibration-summary-candidate.md
```

评分基准变更前必须达到 formal 数据集要求，并保留 `Dataset Policy`、`Collection Plan` 和 `Score Profile Policies` 汇总输出。多 profile 发布不能只看整体样本数，必须分别确认 `vps`、`server` 和 `workstation` 的 profile policy。
正式校准和发布前审计还必须追加 `--require-manifest`，确保每个 `calibration_sample.json` 都能通过样本目录 `manifest.json` 的 SHA256 校验。

### 第三优先：报告产品化

报告下一步重点不是更多字段，而是更强结构：

- 控制台：一屏总览 + 分节详情，适合截图和论坛复制。
- Markdown：保持短摘要，完整细节留给附件和 Web。
- JSON：保持 schema 稳定，所有降级和失败原因结构化。
- Web：Material Design 3 / Google 风格，左侧固定导航，右侧模块证据表。

每个模块都应包含：

- 结论：这一项适合/不适合说明什么。
- 指标：关键数值和单位。
- 证据：来源、状态、耗时、错误或跳过原因。
- 限制：估算、启发式、外部依赖或平台限制。
- 建议：用户下一步如何复测或修复。

### 第四优先：主流后端稳定化

主流口径不是简单调用外部工具，而是把外部工具的失败、跳过和环境限制解释清楚。

重点增强：

- `iperf3`: 授权节点文件、`auth=owned|authorized` 节点授权声明、节点标签、失败节点保留、上下行分开统计。
- `speedtest`: 节点信息、出口 IP、jitter、结果 URL、许可或网络失败分类；下一步用真实失败样本回归分类边界。
- `fio`: direct I/O、磁盘空间、权限、文件系统能力和低配机器跳过策略；下一步补充真实文件系统/低配样本。
- `sysbench`: CPU/内存时间预算、低内存跳过、结果波动解释；下一步补充低内存和旧版本 sysbench 样本。
- `Geekbench`: 只保留显式可选路径，不默认下载运行。

### 第五优先：平台化边界

以下能力暂不进入默认一把梭：

- 默认上传在线报告。
- 默认长期本地历史追踪。
- 内置真实回程探针或公共探针池。
- 未授权公共 iperf3 节点池。

如果后续要做平台化，应先实现显式上传到用户自有 endpoint、脱敏报告导入导出、schema 驱动排行和授权节点安全设计。
