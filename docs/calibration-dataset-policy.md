# 校准样本最小数据集规范

本文档定义 `calibration_sample.json` 进入评分阈值回测的最低准入规则。它用于约束 `scripts/calibration-summary.py` 的过滤与建议输出，避免用 quick、builtin、低置信或覆盖不足的样本直接调整评分基准。

## 样本分级

### 正式校准集

用于调整下一版 `score_calibration.version` 的基准线。

必须满足：

- 每个 `score_profile` 至少 20 个样本。
- 样本必须使用同一个 `score_calibration.version`。
- `confidence_level` 必须为 `high`。
- `mainstream_count` 必须为 4，即 CPU、内存、磁盘、网络四个核心后端均为主流后端。
- 每个核心模块状态必须为 `success`。
- 同一虚拟化类型不应超过样本总数的 70%。
- 同一 CPU 型号不应超过样本总数的 40%。
- x86_64 和 aarch64 如同时作为目标平台，应分别独立统计，不混用一个结论。

推荐命令：

```bash
python3 scripts/calibration-summary.py samples \
  --score-profile server \
  --min-confidence high \
  --require-mainstream \
  --require-policy formal \
  --format markdown \
  -o calibration-summary-server.md
```

### 候选校准集

用于观察阈值是否可能需要调整，但不能直接发布新基准线。

必须满足：

- 每个 `score_profile` 至少 8 个样本。
- `confidence_level` 至少为 `medium`。
- `mainstream_count` 至少为 3。
- CPU、内存、磁盘、网络中被统计的组件状态必须为 `success`。

推荐命令：

```bash
python3 scripts/calibration-summary.py samples \
  --score-profile server \
  --min-confidence medium \
  --min-mainstream-count 3 \
  --require-policy candidate \
  --format markdown \
  -o calibration-summary-candidate.md
```

### 探索分析集

用于研发观察和报告调试，不用于调整评分阈值。

允许包含：

- `builtin` 后端样本
- `confidence_level=low`
- `basic` 或快速测评样本
- 少量未完成模块

探索分析输出的 `Calibration Recommendations` 只能视作候选值，不能直接修改 `internal/reporter/score.go`。

如果希望在脚本或发布流程中强制检查数据集等级，可以使用：

```bash
python3 scripts/calibration-summary.py samples --require-policy candidate
python3 scripts/calibration-summary.py samples --require-policy formal
```

当过滤后的样本集达不到指定等级时，命令会返回非零退出码，并在 stderr 输出主要阻断项或警告项。

`scripts/calibration-summary.py` 只接受 `schema_version=perfassess-calibration-sample-v1` 且 `redacted=true` 的样本。工具会拒绝嵌入原始报告结构、敏感字段名、非白名单环境字段或公网 IPv4 值的样本，避免未脱敏数据进入评分回测。

## 不得进入正式校准的数据

- 未脱敏原始报告。
- 包含公网 IP、ISP、ASN、组织、精确地理位置、路由 hop 或原始日志的文件。
- `redacted` 不为 `true` 的样本。
- 不同 Perfassess 主版本或不兼容 schema 混合后的样本。
- 手工修改过核心指标但没有记录来源的样本。

## 阈值调整原则

- 使用 P75 作为主要候选基准，避免少数高配机器把阈值拉得过高。
- 样本不足时只能输出 `collect_more`，不能发布新阈值。
- 候选值相对当前基准变化小于 10% 时默认保持。
- 延迟指标越低越好，调整前必须人工检查地域分布和网络路径。
- 上传速度为估算值的样本不得用于正式网络上传基准。

## 发布要求

调整评分基准前必须保留：

- 汇总命令
- 样本数量
- 过滤条件
- `python3 scripts/calibration-summary.py --format json` 输出
- 人工判断记录

发布说明应写明新旧 `score_calibration.version`、主要阈值变化和样本覆盖范围。
