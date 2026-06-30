# 真实 VPS 校准样本矩阵

本文档定义下一阶段真实 VPS 样本采集顺序。目标是让评分校准从“工程默认阈值”进入“真实样本回测”，同时保持脱敏、可审计和可复现。

## 采样原则

- 只归档自动测评生成的 `calibration_sample.json`，不归档原始报告、日志、压缩包、公网 IP、ISP、ASN 或路由 hop。
- 每份样本必须通过 `scripts/calibration-collect.sh` 收集，生成 `manifest.json` 和样本集汇总。
- 正式样本池必须开启 `PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1`。
- 样本标签不得包含公网 IP、服务商账号、精确机房名或用户私有信息。
- candidate 阶段先观察趋势，不调整评分阈值；formal 阶段才允许提出 `score_calibration.version` 变更。

## 第一批矩阵

第一批目标是覆盖低配、常规、高配、不同架构和网络差异，优先拿到 candidate 级别的数据。

| 分组 | 建议数量 | 标签示例 | 目标配置 | 采样目的 |
| --- | ---: | --- | --- | --- |
| low-memory | 3 | `vps-low-001` | 512MB-1GB RAM, 1C | 验证低内存降级、预构建二进制、跳过解释和报告完整性 |
| small-vps | 3 | `vps-small-001` | 1C-2C, 1GB-2GB RAM | 覆盖常见入门 VPS，观察 builtin 与 mainstream 差异 |
| standard-vps | 4 | `vps-standard-001` | 2C-4C, 2GB-8GB RAM | 作为 VPS/server 基准主力样本 |
| high-vps | 2 | `vps-high-001` | 4C+, 8GB+ RAM | 观察评分上限、fio 和 sysbench 的高性能区间 |
| arm64 | 2 | `vps-arm64-001` | aarch64 VPS | 验证非 x86_64 架构的构建、运行和评分解释 |
| ipv6-network | 2 | `vps-ipv6-001` | IPv6 可用机器 | 验证 IPv4/IPv6 网络质量矩阵和报告表达 |

第一批不要求每组都一次达到 formal。目标是至少得到：

- `server` 或 `vps` profile 的 candidate 观察集。
- 至少 8 份 `confidence_level >= medium` 的样本。
- 尽量让 `mainstream_count >= 3`，优先覆盖 `sysbench`、`fio`、`speedtest`。

## 单台机器执行流程

在被测 VPS 上运行自动测评：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile standard --quality mainstream --skip-acceptance
cat /tmp/perfassess-auto/summary.md
```

低内存机器优先使用保守档位：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile basic --quality builtin --skip-acceptance
cat /tmp/perfassess-auto/summary.md
```

如有自有或授权 iperf3 节点，可在 mainstream 样本中补真实上传/下载。下面的 `192.0.2.10:5201` 是文档保留地址，运行前必须替换：

```bash
PERFASSESS_IPERF3_SERVER=192.0.2.10:5201 \
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile standard --quality mainstream --skip-acceptance
```

## 样本归档流程

把被测机器上的 `/tmp/perfassess-auto/calibration_sample.json` 取回到维护样本池的机器后，使用收集助手归档：

```bash
PERFASSESS_CALIBRATION_LABEL=vps-standard-001 \
PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1 \
scripts/calibration-collect.sh /path/to/calibration_sample.json calibration-samples
```

如果样本已经在一个自动测评输出目录中：

```bash
PERFASSESS_CALIBRATION_LABEL=vps-standard-001 \
PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1 \
scripts/calibration-collect.sh /path/to/perfassess-auto calibration-samples
```

收集后目录应类似：

```text
calibration-samples/
  vps-standard-001/
    calibration_sample.json
    manifest.json
  summary.md
  summary.json
  samples.jsonl
```

`calibration-samples/` 已被 `.gitignore` 忽略。不要把真实样本池直接提交到仓库。

## 数据集检查

候选观察：

```bash
python3 scripts/calibration-summary.py calibration-samples \
  --require-manifest \
  --score-profile server \
  --min-confidence medium \
  --min-mainstream-count 3 \
  --require-policy candidate \
  --format markdown \
  -o calibration-samples/candidate-server.md
```

多 profile 发布前检查：

```bash
python3 scripts/calibration-summary.py calibration-samples \
  --require-manifest \
  --require-score-profiles vps,server,workstation \
  --require-policy candidate \
  --format markdown \
  -o calibration-samples/candidate-all-profiles.md
```

正式校准门禁：

```bash
python3 scripts/calibration-summary.py calibration-samples \
  --require-manifest \
  --require-score-profiles vps,server,workstation \
  --require-policy formal \
  --format markdown \
  -o calibration-samples/formal-all-profiles.md
```

## 判断下一步

查看 `summary.md` 或候选汇总中的：

- `manifest_required` 和 `manifest_checked_count`：确认样本完整性审计已执行。
- `Dataset Policy`：确认整体是 `exploratory`、`candidate` 还是 `formal`。
- `Collection Plan`：按 `collect_more_samples`、`raise_confidence`、`increase_mainstream_coverage` 等动作补样本。
- `Score Profile Policies`：分别确认 `vps`、`server`、`workstation`，不要用整体样本数替代单个 profile 判断。
- `Calibration Recommendations`：只作为候选阈值观察，formal 达标前不得直接调整评分基准。

## 不采集的内容

- 不采集原始 `default.json`、`console.txt`、日志、zip 包或完整报告目录作为校准输入。
- 不用未授权公共 iperf3 节点制造 mainstream 样本。
- 不把单一云厂商、单一 CPU 型号或单一架构样本直接作为正式校准依据。
- 不因为 candidate 样本的候选阈值变化就直接改 `internal/reporter/score.go`。
