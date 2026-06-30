# VPS 真实环境验收

本文档用于在真实 VPS 或测试机上验证项目是否达到可运行、可解释、可复现的最低发布门槛。

## 目标

- 验证最终二进制可以直接运行
- 验证默认无可选依赖时仍能完成核心验收
- 验证 JSON 报告包含关键摘要、评分校准、分享模板和评分拆解
- 验证 JSON 报告包含测评结论、关键证据和模块级可信度
- 验证报告对比与目录批量排序链路可用
- 在已安装可选依赖时补充 sysbench、fio、speedtest、iperf3 后端验收

## 前置条件

必须具备：

- Linux VPS 或测试机
- 当前项目源码或发布二进制
- `python3`

可选依赖：

- `sysbench`: CPU 和内存主流后端
- `fio`: 磁盘主流后端
- `speedtest`: Ookla 网络测速后端
- `iperf3`: 自有或授权 iperf3 节点测速后端

程序只检测并提示依赖，不会自动安装。

## 快速验收

如果希望一条命令完成构建、依赖检查、默认测评、核心验收和摘要生成，使用：

```bash
scripts/perfassess-auto.sh
cat /tmp/perfassess-auto/summary.md
```

该脚本默认不运行可选外部后端，避免缺失依赖或外部服务导致自动化流程失败。如需按已安装工具自动加测可选后端：

```bash
PERFASSESS_AUTO_OPTIONAL=auto scripts/perfassess-auto.sh
```

从源码构建后执行：

```bash
go build -o build/perfassess cmd/main.go
scripts/vps-acceptance.sh
```

使用已发布二进制：

```bash
PERFASSESS_BINARY=/usr/local/bin/perfassess scripts/vps-acceptance.sh
```

指定输出目录：

```bash
PERFASSESS_ACCEPTANCE_DIR=/tmp/perfassess-acceptance scripts/vps-acceptance.sh
```

不运行可选依赖后端：

```bash
PERFASSESS_ACCEPTANCE_OPTIONAL=never scripts/vps-acceptance.sh
```

## 验收矩阵

默认 `scripts/vps-acceptance.sh` 只运行轻量 smoke，覆盖版本、依赖检查、quick、network、cpu、报告对比和目录排序，适合日常开发验证。

真实服务器建议按机器配置选择矩阵：

```bash
# 低配或 512MB 机器：允许内存测试因资源不足跳过，但报告必须说明原因和置信度
PERFASSESS_ACCEPTANCE_MATRIX=low scripts/vps-acceptance.sh

# 普通 VPS：运行 builtin 完整基础测评和路由、流媒体、AI、IP 质量、安全体检
PERFASSESS_ACCEPTANCE_MATRIX=standard scripts/vps-acceptance.sh

# 发布前主流口径：运行 full 预设，需要 sysbench 和 fio
PERFASSESS_ACCEPTANCE_MATRIX=full scripts/vps-acceptance.sh

# 一次性覆盖全部矩阵
PERFASSESS_ACCEPTANCE_MATRIX=all scripts/vps-acceptance.sh
```

多个矩阵可以逗号分隔：

```bash
PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh
```

如果从源码仓库中运行，也可以使用等价 Makefile 入口：

```bash
make vps-acceptance-low-standard
```

默认矩阵允许资源不足导致核心模块跳过，但必须在报告中输出 `confidence_level.reasons` 和 `performance_note`。如果要发布前强制所有核心测试成功，可以启用严格模式：

```bash
PERFASSESS_ACCEPTANCE_MATRIX=low,standard,full \
PERFASSESS_ACCEPTANCE_STRICT=1 \
scripts/vps-acceptance.sh
```

非严格模式下，可选外部后端的行为分三类：

- 如果后端成功运行，报告会进入 `optional_reports`。
- 如果后端因为资源不足、外部服务不可用或环境限制被报告明确标记为 `skipped`，验收会进入 `skipped_reports`，不会阻断核心验收。
- 如果后端已经启动但被系统杀掉、外部命令失败或平台限制导致失败，只要 JSON 报告保留 `confidence_level.reasons`、`assessment_conclusion.evidence` 和错误诊断，也会进入 `skipped_reports`；严格模式下仍会失败。

严格模式下，矩阵报告或可选后端被跳过都会失败，适合发布前或评分基准变更前使用。启用严格矩阵时，脚本会在正式测评前和每个矩阵项开始前检查可用内存；默认要求 `MemAvailable` 至少 768 MB，可通过 `PERFASSESS_ACCEPTANCE_STRICT_MIN_MEM_MB` 调整。低于门槛时会提前失败并提示释放内存、增加 swap、停止其他工作负载，或改用非严格模式生成可解释降级报告。

## iperf3 验收

`iperf3` 不内置公共节点。请只使用自有或授权节点。

单节点。`192.0.2.10:5201` 是文档保留地址，只表示格式，运行前必须替换为自有或授权节点：

```bash
PERFASSESS_IPERF3_SERVER=192.0.2.10:5201 scripts/vps-acceptance.sh
```

节点文件：

```bash
PERFASSESS_IPERF3_SERVER_FILE=/path/to/iperf3-servers.txt scripts/vps-acceptance.sh
```

## 必过检查

脚本必须生成并校验：

- `version.txt`
- `check-deps.txt`
- `quick.json`
- `network.json`
- `cpu.json`
- `compare.json`
- `compare-dir.json`
- `summary.md`
- `summary.json`

启用矩阵后还会按需生成：

- `low-basic.json`
- `standard-builtin.json`
- `full-mainstream.json`

如果已安装可选依赖，脚本还会尝试生成：

- `cpu-sysbench.json`
- `memory-sysbench.json`
- `disk-fio.json`
- `network-speedtest.json`
- `network-iperf3.json` 或 `network-iperf3-file.json`

这些可选报告必须至少满足 JSON 与报告契约检查。非严格模式下，资源不足导致的 `skipped` 会写入 `summary.md` 的 `skipped_reports`。

每份核心 JSON 报告必须包含：

- `summary.benchmark_profile`
- `summary.confidence_level`
- `summary.score_calibration`
- `summary.score_breakdown`
- `summary.vps_benchmark_summary`
- `summary.assessment_conclusion`
- `summary.assessment_conclusion.evidence`
- `summary.module_assessments`
- `summary.share_templates`

`summary.score_calibration.version` 当前必须为 `2026-06-v1`。
`summary.module_assessments` 必须至少包含 `cpu`、`memory`、`disk`、`network`、`route`、`ip_quality`、`streaming` 和 `ai_services` 八个模块；未执行的扩展模块也必须以 `skipped` 状态显式出现，避免报告读者误以为没有检测边界。

## 通过标准

- 脚本退出码为 `0`
- 最后一行包含 `VPS acceptance passed`
- `summary.md` 能看到二进制版本、输出目录、总分、等级、置信度、评分基准和校准版本
- `summary.md` 能看到本次验收矩阵、已生成报告、可选报告和跳过报告
- `summary.md` 的 `Report Coverage` 能看到每份 JSON 报告的成功、失败、跳过数量、置信度和置信原因
- `summary.md` 的 `Report Coverage` 会额外展示 `sample_policy`，用于判断单份报告适合 `exploratory_only`、`candidate_eligible` 还是 `formal_eligible`
- `summary.md` 的 `Calibration Readiness` 会汇总正式候选、候选校准和探索样本数量
- 验收脚本会自检 `summary.md` 是否保留 `Report Coverage`、核心报告名和 `success/failed/skipped/confidence/reasons/sample_policy` 字段；缺失会直接失败
- `summary.json` 能被 `python3 -m json.tool` 解析，并包含 `acceptance_matrix`、`generated_reports`、`optional_reports`、`skipped_reports`、`report_coverage`、`confidence_reasons`、`calibration_readiness` 和 `calibration_readiness_counts`，方便 CI 或外部平台消费
- 可选依赖缺失时不应导致核心验收失败
- 已安装可选依赖时，对应后端报告应生成并通过 JSON 与报告契约校验；资源不足导致的可解释跳过必须写入 `skipped_reports`
- 非严格模式下，低内存机器允许内存测试跳过，但报告必须明确标记未完成、低置信和跳过原因
- 严格模式下，矩阵报告必须完成 CPU、内存、磁盘、网络四项核心测试

## 校准样本闭环

`scripts/vps-acceptance.sh` 直接运行二进制并校验多份 JSON 报告，重点是判断真实机器上的报告质量、失败解释和校准适用性。它不会自己生成 `calibration_sample.json`。

真正的脱敏校准样本由自动测评脚本生成：

```bash
scripts/perfassess-auto.sh
cat /tmp/perfassess-auto/summary.md
```

自动测评完成后会生成 `/tmp/perfassess-auto/calibration_sample.json`。该样本已标记 `redacted=true`，不包含公网 IP、ISP、ASN、精确地理位置、路由 hop 或原始日志。收集多台机器样本后，用以下命令判断数据集等级：

```bash
python3 scripts/calibration-summary.py /path/to/samples --require-policy candidate
python3 scripts/calibration-summary.py /path/to/samples --require-policy formal
```

推荐用收集助手归档样本，避免误复制原始报告、日志或完整压缩包：

```bash
scripts/calibration-collect.sh /tmp/perfassess-auto/calibration_sample.json /path/to/samples
scripts/calibration-collect.sh /tmp/perfassess-auto /path/to/samples
```

该脚本只接受已脱敏的 `calibration_sample.json`，会把样本放入独立子目录，并在样本集目录生成：

- 每个样本子目录的 `manifest.json`：记录样本标签、归档时间和 `calibration_sample.json` 的 SHA256
- `summary.md`：人工阅读的校准汇总和下一批采样计划
- `summary.json`：机器可读汇总
- `samples.jsonl`：每份样本的一行摘要，便于外部系统导入

如需给样本指定不含隐私的标签，可以设置：

```bash
PERFASSESS_CALIBRATION_LABEL=vps-low-001 \
scripts/calibration-collect.sh /tmp/perfassess-auto /path/to/samples
```

`vps-acceptance` 的 `sample_policy` 只判断单份报告是否具备进入候选样本的条件；正式调分仍必须满足 [校准样本最小数据集规范](calibration-dataset-policy.md) 中的数据量、分布、置信度和脱敏要求。

## 失败处理

- 二进制不可执行：先构建或指定 `PERFASSESS_BINARY`
- `python3` 缺失：安装 Python 3 后重试
- 外部后端失败：先查看 `check-deps.txt` 和对应 `*.stdout.txt`
- `cpu-sysbench` 超时：确认当前二进制已包含 sysbench 后端预算修正；sysbench CPU 会执行单核和多核多轮采样，耗时高于 builtin
- `memory-sysbench` 跳过：通常是可用内存不足。非严格模式下这是可接受的降级，严格模式下需要释放内存或换更高配置机器复测
- `iperf3` 失败：确认节点可访问、端口正确且属于自有或授权节点
- JSON 字段缺失：说明报告契约可能被破坏，应阻止发布

## 脱敏归档

真实 VPS 输出可能包含公网 IP、ISP、地理位置、CPU 型号等信息。归档前请脱敏：

- 删除或替换 `public_ip`
- 删除或替换 `speedtest_external_ip`
- 按需替换 `isp`、`location`、`country_code`
- 保留 `summary.score_calibration`、`summary.score_breakdown`、`summary.confidence_level` 等结构字段

脱敏样例格式见 [examples/acceptance/README.md](examples/acceptance/README.md)。
