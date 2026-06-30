# 报告 JSON Schema 说明

本文档记录 `--output-format json` 的核心字段契约。机器可读 JSON Schema 见 [docs/report.schema.json](report.schema.json)，CI 会校验示例报告和生成报告是否符合该 schema。

示例报告见 [docs/examples/report-json-sample.json](examples/report-json-sample.json)。

## 顶层结构

- `session_id`: 本次评估会话 ID
- `timestamp`: 报告生成时间
- `system_info`: 系统硬件、系统、虚拟化和 IP 信息
- `test_results`: CPU、内存、磁盘、网络测试结果
- `summary`: 综合评分、质量提示、评测档位和置信度

## 通用测试结果

每个测试结果使用同一结构：

- `test_name`: 测试名称
- `status`: `success`、`failed`、`skipped`、`degraded`
- `start_time`: 测试开始时间
- `end_time`: 测试结束时间
- `duration_seconds`: 测试耗时，单位秒
- `metrics`: 测试指标
- `error_message`: 失败或降级原因

当外部后端失败时，`metrics` 可包含统一错误分类：

- `error_category`: 错误类别，可能为 `missing_dependency`、`invalid_config`、`command_failed`、`permission_denied`、`resource_limited`、`network_unavailable`、`parse_failed`、`timeout` 或 `runtime_error`
- `error_stage`: 出错阶段，例如 `fio_write_run`、`speedtest_parse`、`iperf3_config`
- `error_hint`: 面向用户的处理建议

网络测试可能在延迟、下载、上传多个阶段分别失败，因此还可能包含 `network_error_<stage>_category`、`network_error_<stage>_stage` 和 `network_error_<stage>_hint`。

## CPU Metrics

- `backend`: `builtin`、`sysbench` 或 `geekbench`
- `single_core_source`: 单核指标来源
- `single_core_score`: 单核评分
- `single_core_raw_score`: Geekbench 单核原始分
- `single_core_events_per_sec`: sysbench 单核吞吐，单位 events/s
- `multi_core_source`: 多核指标来源
- `multi_core_score`: 多核评分
- `multi_core_raw_score`: Geekbench 多核原始分
- `multi_core_events_per_sec`: sysbench 多核吞吐，单位 events/s
- `total_score`: CPU 总分
- `cpu_cores`: CPU 逻辑核心数
- `*_samples`、`*_min`、`*_median`、`*_max`、`*_stddev`: 多轮采样统计

## Memory Metrics

- `backend`: `builtin` 或 `sysbench`
- `read_speed_mbps`: 内存读取吞吐，单位 MiB/s
- `read_speed_source`: 读取指标来源
- `write_speed_mbps`: 内存写入吞吐，单位 MiB/s
- `write_speed_source`: 写入指标来源
- `score`: 内存评分
- `test_size_mb`: 测试数据规模，单位 MiB
- `*_samples`、`*_min`、`*_median`、`*_max`、`*_stddev`: 多轮采样统计

## Disk Metrics

- `backend`: `builtin` 或 `fio`
- `sequential_read_mbps`: 顺序读取吞吐，单位 MiB/s
- `sequential_read_source`: 顺序读取来源
- `sequential_read_iops`: fio 顺序读取 IOPS
- `sequential_read_latency_ms`: fio 顺序读取平均延迟，单位 ms
- `sequential_read_latency_p95_ms`: fio 顺序读取 P95 延迟，单位 ms
- `sequential_write_mbps`: 顺序写入吞吐，单位 MiB/s
- `sequential_write_source`: 顺序写入来源
- `sequential_write_iops`: fio 顺序写入 IOPS
- `sequential_write_latency_ms`: fio 顺序写入平均延迟，单位 ms
- `sequential_write_latency_p95_ms`: fio 顺序写入 P95 延迟，单位 ms
- `random_iops`: 随机读写总 IOPS
- `random_iops_source`: 随机 IOPS 来源
- `random_read_mbps`: fio 4k mixed 随机读吞吐，单位 MiB/s
- `random_write_mbps`: fio 4k mixed 随机写吞吐，单位 MiB/s
- `random_read_iops`: fio 随机读 IOPS
- `random_write_iops`: fio 随机写 IOPS
- `random_read_latency_ms`: fio 随机读平均延迟，单位 ms
- `random_write_latency_ms`: fio 随机写平均延迟，单位 ms
- `random_read_latency_p95_ms`: fio 随机读 P95 延迟，单位 ms
- `random_write_latency_p95_ms`: fio 随机写 P95 延迟，单位 ms
- `fio_mixed_profile`: fio mixed 矩阵口径，当前为 `yabs_randrw_50_50`
- `fio_mixed_block_sizes`: mixed 矩阵块大小列表，当前为 `4k`、`64k`、`512k`、`1m`
- `fio_mixed_<block>_read_mbps`: 指定块大小 mixed 随机读吞吐，单位 MiB/s
- `fio_mixed_<block>_write_mbps`: 指定块大小 mixed 随机写吞吐，单位 MiB/s
- `fio_mixed_<block>_total_mbps`: 指定块大小 mixed 随机总吞吐，单位 MiB/s
- `fio_mixed_<block>_read_iops`: 指定块大小 mixed 随机读 IOPS
- `fio_mixed_<block>_write_iops`: 指定块大小 mixed 随机写 IOPS
- `fio_mixed_<block>_total_iops`: 指定块大小 mixed 随机总 IOPS
- `score`: 磁盘评分

## Network Metrics

- `backend`: `builtin`、`iperf3` 或 `speedtest`
- `backend_server`: 外部网络后端服务端信息；`iperf3` 为用户指定服务端、服务端列表或节点文件路径，`speedtest` 为 CLI 自动选择的测速节点
- `latency_ms`: 延迟，单位 ms
- `average_latency_ms`: 平均延迟，单位 ms
- `latency_source`: 延迟来源，当前主要为 `tcp_connect`
- `download_speed_mbps`: 下载吞吐，单位 Mbps
- `download_speed_source`: 下载测速来源；`iperf3` 后端为 `iperf3_download`，`speedtest` 后端为 `speedtest_download`，内置后端成功时记录实际 HTTP 下载源 URL，失败时为 `http_download_failed`
- `upload_speed_mbps`: 上传吞吐，单位 Mbps
- `upload_speed_source`: 上传测速来源
- `upload_speed_estimated`: 上传是否为估算值
- `network_error`: 网络部分失败说明
- `network_quality_profile`: 网络质量矩阵口径，当前为 `tcp_connect_matrix`
- `network_quality_target_count`: 网络质量目标数量
- `network_quality_ipv4_available`: IPv4 TCP connect 至少一个目标成功
- `network_quality_ipv6_available`: IPv6 TCP connect 至少一个目标成功
- `network_quality_ipv4_failure_rate`: IPv4 目标整体失败率，范围 0-1
- `network_quality_ipv6_failure_rate`: IPv6 目标整体失败率，范围 0-1
- `network_quality_failure_rate`: 全部质量目标整体失败率，范围 0-1
- `network_quality_avg_latency_ms`: 全部成功样本加权平均 TCP connect 延迟，单位 ms
- `network_quality_jitter_ms`: 全部成功样本加权平均抖动，单位 ms
- `network_quality_<n>_target`: 第 n 个质量目标名称
- `network_quality_<n>_address`: 第 n 个质量目标地址
- `network_quality_<n>_protocol`: 第 n 个质量目标协议族，`ipv4` 或 `ipv6`
- `network_quality_<n>_available`: 第 n 个质量目标是否有成功样本
- `network_quality_<n>_success_count`: 第 n 个质量目标成功采样次数
- `network_quality_<n>_failure_count`: 第 n 个质量目标失败采样次数
- `network_quality_<n>_failure_rate`: 第 n 个质量目标失败率，范围 0-1
- `network_quality_<n>_avg_latency_ms`: 第 n 个质量目标平均 TCP connect 延迟，单位 ms
- `network_quality_<n>_min_latency_ms`: 第 n 个质量目标最小 TCP connect 延迟，单位 ms
- `network_quality_<n>_max_latency_ms`: 第 n 个质量目标最大 TCP connect 延迟，单位 ms
- `network_quality_<n>_jitter_ms`: 第 n 个质量目标 TCP connect 抖动，单位 ms
- `iperf3_matrix_profile`: iperf3 多节点矩阵口径，当前为 `multi_server`
- `iperf3_matrix_server_count`: iperf3 矩阵服务端数量
- `iperf3_matrix_success_count`: 至少完成下载或上传的服务端数量
- `iperf3_matrix_avg_download_mbps`: iperf3 多节点平均下载吞吐
- `iperf3_matrix_avg_upload_mbps`: iperf3 多节点平均上传吞吐
- `iperf3_matrix_best_download_mbps`: iperf3 多节点最佳下载吞吐
- `iperf3_matrix_best_upload_mbps`: iperf3 多节点最佳上传吞吐
- `iperf3_matrix_<n>_server`: 第 n 个 iperf3 服务端原始配置
- `iperf3_matrix_<n>_name`: 第 n 个 iperf3 节点名称，来自节点文件 `name=` 元数据
- `iperf3_matrix_<n>_region`: 第 n 个 iperf3 节点区域，来自节点文件 `region=` 元数据
- `iperf3_matrix_<n>_provider`: 第 n 个 iperf3 节点提供方，来自节点文件 `provider=` 元数据
- `iperf3_matrix_<n>_authorization`: 第 n 个 iperf3 节点授权声明，来自节点文件 `auth=owned|authorized`
- `iperf3_matrix_<n>_protocol`: 第 n 个 iperf3 服务端协议族，`ipv4` 或 `ipv6`
- `iperf3_matrix_<n>_latency_ms`: 第 n 个 iperf3 服务端 TCP connect 延迟
- `iperf3_matrix_<n>_download_mbps`: 第 n 个 iperf3 服务端下载吞吐
- `iperf3_matrix_<n>_upload_mbps`: 第 n 个 iperf3 服务端上传吞吐
- `iperf3_matrix_<n>_error`: 第 n 个 iperf3 服务端失败说明
- `score`: 网络评分

## Summary

- `overall_score`: 综合评分对象
- `total_score`: 总分
- `grade`: 等级
- `cpu_score`: CPU 分项分
- `memory_score`: 内存分项分
- `disk_score`: 磁盘分项分
- `network_score`: 网络分项分
- `tests_success`: 成功测试数量
- `tests_failed`: 失败测试数量
- `tests_skipped`: 跳过测试数量
- `tests_degraded`: 降级测试数量，例如网络只完成延迟但吞吐失败
- `performance_note`: 未完成全部核心测试时的说明
- `quality_notes`: 质量提示列表
- `assessment_conclusion`: 测评结论，包含适用场景、关键证据、短板排序、限制和建议
- `module_assessments`: 模块级可信度，统一描述 CPU、内存、磁盘、网络、路由追踪、IP 质量、流媒体和 AI 服务的状态、置信度、证据、限制和建议
- `benchmark_profile`: 本次评测档位和实际后端组合
- `confidence_level`: 本次报告置信等级和原因
- `vps_benchmark_summary`: 面向 VPS 测评分享的核心摘要
- `share_templates`: 可直接复制分享的文本和 Markdown 模板
- `score_profile`: 评分基准档位，当前为 `vps`、`server` 或 `workstation`
- `score_calibration`: 本次评分使用的校准版本、基准线和等级阈值
- `score_breakdown`: 评分计算依据和分项权重

## IP Quality Report

`summary.ip_quality_report` 表示 IP 节点分析报告。

- `public_ip`、`ip_version`、`isp`、`country`、`city`、`asn`、`organization`、`reverse_dns`: 基础 IP、ASN 和反向 DNS 信息
- `ip_type`: IP 类型推断，例如 `datacenter_likely` 或 `residential_or_isp_likely`
- `risk_level`、`risk_score`: 风险等级和 0-100 风险分
- `risk_sources`: 风险来源摘要，包含 Team Cymru ASN、Reverse DNS、DNSBL、本地关键词启发式，以及默认禁用的商业风险 API 说明
- `risk_factors`: 风险因子明细，例如代理、VPN、Tor、机房/托管和滥用线索
- `blacklist_summary`、`blacklist_checks`: DNSBL 黑名单汇总和逐项结果
- `mail_summary`、`mail_checks`: 邮件服务商出站连通性汇总和逐项结果
- `network_stack`: 当前 IP 质量检测所基于的网络栈视角
- `verdict`、`evidence`、`recommendations`、`notes`: 面向人工阅读的结论、证据、建议和说明

## VPS Benchmark Summary

`vps_benchmark_summary` 是对完整报告的压缩摘要，便于复制分享和脚本快速读取。它不替代完整 `test_results`，只保留 VPS 测评最常用的核心指标。

- `benchmark_profile`: 本次后端组合档位，例如 `full_iperf3`
- `mainstream_backend_count`: 使用主流后端数量
- `system`: CPU 型号、核心线程、内存、磁盘、OS、虚拟化、IP/ISP/位置等系统摘要
- `cpu`: CPU 后端、状态、单核/多核/总分、sysbench events/s 或 Geekbench 原始分
- `memory`: 内存后端、状态、读写吞吐和评分
- `disk`: 磁盘后端、状态、顺序读写、随机 IOPS、P95 延迟和 fio mixed 关键值
- `network`: 网络后端、状态、延迟、下载、上传、上传估算标记、speedtest 节点信息、iperf3 聚合值、IPv4/IPv6 可用性、质量矩阵失败率和抖动
- `scores`: 总分、等级、分项分和评分基准
- `confidence`: 置信等级、原因和质量提示

## Share Templates

`share_templates` 基于 `vps_benchmark_summary` 生成，便于直接复制到论坛、工单或 IM。

- `plain_text`: 多行纯文本模板，适合终端、聊天工具和不支持 Markdown 的平台
- `markdown`: Markdown 表格模板，适合 README、Issue、论坛和支持 Markdown 的平台

## Benchmark Profile

- `name`: `quick`、`default`、`full`、`full_iperf3`、`full_speedtest` 或 `custom`
- `cpu_backend`: CPU 实际后端
- `memory_backend`: 内存实际后端
- `disk_backend`: 磁盘实际后端
- `network_backend`: 网络实际后端
- `mainstream_count`: 使用主流外部基准后端的核心测试数量

## Network Metrics

网络测试会按后端追加可选指标：

- `speedtest_*`: Ookla Speedtest CLI 返回的节点信息，包括节点 ID、名称、地区、国家、Host、ISP、结果 URL、出口 IP、网卡名称、VPN 标记和 ping jitter
- `iperf3_matrix_*`: iperf3 多节点矩阵，包括节点数、成功数、平均/最佳上下行，以及每个节点的 server、协议、端口、延迟、上下行和错误信息
- `network_quality_*`: 内置 TCP connect 质量矩阵，包括 IPv4/IPv6 可用性、失败率、平均延迟和抖动

## Streaming Results

`streaming_results` 是流媒体解锁检测结果，键为平台名称。每个平台结果包含：

- `platform`: 平台名称，例如 `Netflix`、`Disney+`
- `available`: 是否可访问
- `region`: 判定区域；无法从响应确认时可能使用平台主要服务区域作为提示
- `message`: 检测说明
- `category`: 平台分组，例如 `global`、`us`、`jp`、`kr`、`music`、`sports`
- `unlock_type`: 解锁类型，例如 `full`、`partial`、`blocked`、`login_required`、`available`
- `protocol`: 当前检测视角，默认 `default`
- `region_source`: 区域来源，例如 `response`、`platform_hint` 或 `unknown`

## AI Results

`ai_results` 是 AI 服务可访问性检测结果，键为服务名称。每个服务结果包含：

- `service`: 服务名称，例如 `ChatGPT`、`Claude`、`Gemini`
- `available`: 是否可访问
- `message`: 检测说明，例如需要登录、地区限制、需要额外验证
- `category`: 服务分组，例如 `chatbot`、`assistant`、`search`、`coding`
- `access_type`: 访问类型，例如 `full`、`login_required`、`verification_required`、`rate_limited`、`restricted`
- `region_hint`: 服务主要区域或区域策略提示

## Confidence Level

- `level`: `high`、`medium`、`low`
- `reasons`: 置信等级原因

判断原则：

- 核心测试未执行或失败时为 `low`
- 使用内置后端或估算上传时至少降为 `medium`
- 核心测试均成功且未发现明显降级或估算路径时为 `high`

## Score Calibration

`score_calibration` 用于解释当前分数按什么基准线计算，避免只看到 `score_profile` 但不知道阈值。

- `version`: 校准版本，当前为 `2026-06-v1`
- `active_profile`: 本次实际使用的评分档位，取值为 `vps`、`server` 或 `workstation`
- `active_baselines`: 当前档位的内存、磁盘和网络基准线
- `profiles`: 三套评分档位的完整基准线，便于横向审计
- `grade_thresholds`: 等级阈值，当前为优秀 90、良好 75、一般 60、较差 0
- `notes`: 评分校准说明，包括 CPU 归一化、未完成测试和后续真实样本回测提示

当前基准线：

| profile | memory read/write MB/s | disk read/write MB/s | disk IOPS | latency ms | down/up Mbps |
|---|---:|---:|---:|---:|---:|
| `vps` | 3000 / 2000 | 300 / 200 | 3000 | 80 | 50 / 25 |
| `server` | 5000 / 3000 | 500 / 300 | 5000 | 50 | 100 / 50 |
| `workstation` | 8000 / 6000 | 1500 / 1000 | 20000 | 30 | 300 / 100 |

## Score Breakdown

- `score_profile`: 本次评分使用的基准档位
- `calibration_version`: 本次评分使用的校准版本
- `cpu`: CPU 评分依据
- `memory`: 内存评分依据
- `disk`: 磁盘评分依据
- `network`: 网络评分依据
- `normalized_total`: 总分归一化说明

每个分项通常包含：

- `active`: 是否参与总分
- `status`: 测试状态
- `score`: 分项得分
- `weight`: 该分项权重
- `formula`: 计算说明
- 原始指标字段，例如 `read_speed_mbps`、`random_iops`、`download_speed_mbps`
- 基准线字段，例如 `read_base_mbps`、`sequential_read_base_mbps`、`download_base_mbps`

总分规则：

- `normalized_total.weighted_sum`: 成功测试分项的加权和
- `normalized_total.active_weight`: 成功测试分项的权重和
- `normalized_total.score`: `weighted_sum / active_weight`
- `normalized_total.calibration_version`: 与 `score_calibration.version` 保持一致
- 未执行或失败的核心测试不会参与权重归一化，但会降低 `confidence_level`

## Compare Result

`perfassess compare <report-a.json> <report-b.json>` 输出两份 JSON 报告的对比结果。文本格式用于人工阅读，`--format json` 用于脚本化处理。

核心字段：

- `report_a`: A 报告路径或标签
- `report_b`: B 报告路径或标签
- `winner`: 胜出方，可能为 `A`、`B`、`平局` 或 `不可直接比较`
- `score_profile_a`: A 报告评分基准
- `score_profile_b`: B 报告评分基准
- `comparable`: 两份报告是否可直接比较
- `warnings`: 不可直接比较或未完成报告的提示
- `total_difference`: 总分差异
- `metric_differences`: CPU、内存、磁盘、网络分项差异

差异字段：

- `name`: 指标名称
- `report_a`: A 报告数值
- `report_b`: B 报告数值
- `delta`: `B - A`
- `delta_percent`: 相对 A 的百分比变化
- `winner`: 该指标胜出方
- `higher_is_better`: 是否数值越高越好

## Compare Directory Entry

`perfassess compare-dir <reports-dir> --format json` 会读取目录中的 JSON 报告，并输出按指定分数字段排序后的报告摘要数组。它只做本次目录内排序，不写入本地历史库。

核心字段：

- `report_path`: 原始报告路径
- `session_id`: 评测会话 ID
- `timestamp`: 报告时间
- `host_label`: 从公网 IP、ISP 和 CPU 型号生成的展示标签
- `cpu_model`: CPU 型号
- `os`: 操作系统名称和版本
- `architecture`: 系统架构
- `score_profile`: 评分基准档位
- `benchmark_profile`: 本次评测后端组合
- `grade`: 等级
- `total_score`: 总分
- `cpu_score`: CPU 分项分
- `memory_score`: 内存分项分
- `disk_score`: 磁盘分项分
- `network_score`: 网络分项分

`--sort-by` 支持 `total`、`cpu`、`memory`、`disk`、`network`。
