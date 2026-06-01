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

## CPU Metrics

- `backend`: `builtin` 或 `sysbench`
- `single_core_source`: 单核指标来源
- `single_core_score`: 单核评分
- `single_core_events_per_sec`: sysbench 单核吞吐，单位 events/s
- `multi_core_source`: 多核指标来源
- `multi_core_score`: 多核评分
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
- `random_read_iops`: fio 随机读 IOPS
- `random_write_iops`: fio 随机写 IOPS
- `random_read_latency_ms`: fio 随机读平均延迟，单位 ms
- `random_write_latency_ms`: fio 随机写平均延迟，单位 ms
- `random_read_latency_p95_ms`: fio 随机读 P95 延迟，单位 ms
- `random_write_latency_p95_ms`: fio 随机写 P95 延迟，单位 ms
- `score`: 磁盘评分

## Network Metrics

- `backend`: `builtin` 或 `iperf3`
- `backend_server`: iperf3 服务端，仅 iperf3 后端使用
- `latency_ms`: 延迟，单位 ms
- `average_latency_ms`: 平均延迟，单位 ms
- `latency_source`: 延迟来源，当前主要为 `tcp_connect`
- `download_speed_mbps`: 下载吞吐，单位 Mbps
- `download_speed_source`: 下载测速来源
- `upload_speed_mbps`: 上传吞吐，单位 Mbps
- `upload_speed_source`: 上传测速来源
- `upload_speed_estimated`: 上传是否为估算值
- `network_error`: 网络部分失败说明
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
- `benchmark_profile`: 本次评测档位和实际后端组合
- `confidence_level`: 本次报告置信等级和原因
- `score_profile`: 评分基准档位，当前为 `vps`、`server` 或 `workstation`
- `score_breakdown`: 评分计算依据和分项权重

## Benchmark Profile

- `name`: `quick`、`default`、`full`、`full_iperf3` 或 `custom`
- `cpu_backend`: CPU 实际后端
- `memory_backend`: 内存实际后端
- `disk_backend`: 磁盘实际后端
- `network_backend`: 网络实际后端
- `mainstream_count`: 使用主流外部基准后端的核心测试数量

## Confidence Level

- `level`: `high`、`medium`、`low`
- `reasons`: 置信等级原因

判断原则：

- 核心测试未执行或失败时为 `low`
- 使用内置后端或估算上传时至少降为 `medium`
- 核心测试均成功且未发现明显降级或估算路径时为 `high`

## Score Breakdown

- `score_profile`: 本次评分使用的基准档位
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

## History Entry

`perfassess history add <report.json>` 会把单次 JSON 报告提取为一行 JSONL 历史索引，默认路径为 `~/.perfassess/history.jsonl`，可通过 `--store` 指定。

核心字段：

- `report_path`: 原始报告路径
- `session_id`: 评测会话 ID
- `timestamp`: 报告时间
- `host_id`: 从系统信息派生的主机标识
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

`perfassess history trend --format json` 输出趋势对象：

- `host_id`: 趋势对应主机
- `count`: 样本数量
- `first`: 第一条历史记录
- `last`: 最后一条历史记录
- `total_delta`: 总分变化
- `cpu_delta`: CPU 分变化
- `memory_delta`: 内存分变化
- `disk_delta`: 磁盘分变化
- `network_delta`: 网络分变化
- `entries`: 参与趋势计算的历史记录

`perfassess compare-dir <reports-dir> --format json` 输出按指定分数字段排序后的 `History Entry` 数组，`--sort-by` 支持 `total`、`cpu`、`memory`、`disk`、`network`。
