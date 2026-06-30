# 发布前检查清单

本文档定义每个版本合并和发版前必须通过的最低门槛。目标是保证项目作为 VPS 测评脚本时具备稳定的一把梭执行、可复现报告和可追踪的输出契约。

## 本地验证

日常开发完成后先运行：

```bash
make validate
```

`make validate` 覆盖：

- Go 格式检查
- JSON schema 与示例报告语法检查
- 项目级不变量检查，包括项目命名、bootstrap 入口、隐私默认关闭、真实回程边界和本地历史移除状态
- 远程测试手册冒烟检查，包括 README 入口、bootstrap 复制命令、Web 访问、失败排查、产物位置、验收矩阵、iperf3 授权节点和清理流程
- 全量单元测试
- 当前平台构建
- CLI help 冒烟测试
- 依赖检查冒烟测试
- 快速报告语义契约检查，包括测评结论、关键证据、模块可信度和分享模板
- Web 报告 HTML 渲染契约检查，包括首屏决策面板、模块健康和报告目录
- 脱敏校准样本隐私契约检查，确保 `calibration_sample.json` 标记 `redacted=true`，不嵌入原始报告结构、敏感字段名或公网 IPv4 值
- JSON 报告脱敏工具冒烟测试，确保公开归档前可生成 `.redacted.json`，并移除 IP、ISP、ASN、地理位置和路由 hop 细节
- 自动测评报告产物清单校验，包括文件大小、SHA256、压缩包内容、目录模式下的压缩包内部 SHA256 校验、目录 manifest 与压缩包内 manifest 一致性、目录篡改和压缩包内部篡改负向测试
- 严格 VPS 验收资源预检冒烟测试，确保资源不足时在生成报告前快速失败并给出提示

发布前运行：

```bash
make release-check
```

候选发布基线的本地验证记录见 [发布验证记录](release-validation-log.md)。新增或更新记录后，仍需按下方真实 VPS 验收要求完成实机验证。

正式创建 GitHub Release 时，按 [GitHub Release 发布手册](release-runbook.md) 执行 tag、workflow、资产校验和发布后 bootstrap 验证。

提交前运行：

```bash
make pre-commit
```

`make pre-commit` 会先显示当前分支和工作区状态，再执行空白检查、shell 脚本语法检查、关键 Python 脚本编译检查和 `make validate`。

`make release-check` 在 `make validate` 基础上额外覆盖：

- 发布二进制版本命令、help 和依赖检查冒烟测试
- 发布二进制快速 JSON 报告生成与 JSON 语法检查
- 发布二进制报告契约检查，包括测评结论、关键证据和模块可信度
- Web 报告 HTML 渲染契约检查，包括首屏决策面板、模块可信度和报告目录
- 发布二进制报告对比和目录批量排序 JSON 冒烟测试
- 发布二进制自动测评 basic 档冒烟测试，并校验 `artifact_manifest.json` 与 `perfassess-report.zip`
- Linux amd64/arm64 构建
- macOS amd64/arm64 构建
- Windows amd64 构建
- 发布产物 `checksums.txt` 生成和 `sha256sum -c` 校验

## 手动冒烟

在真实 VPS 或测试机上建议至少运行：

```bash
scripts/vps-acceptance.sh
PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh
make vps-acceptance-low-standard
./build/perfassess version
./build/perfassess --quick --output-format json -o /tmp/perfassess-quick.json
./build/perfassess -b network --output-format json -o /tmp/perfassess-network.json
./build/perfassess check-deps
```

`scripts/vps-acceptance.sh` 会保留验收产物并生成 `summary.md`。详细流程见 [VPS 真实环境验收](vps-acceptance.md)。

如果机器已安装可选依赖，再补充：

```bash
./build/perfassess --full --output-format json -o /tmp/perfassess-full.json
./build/perfassess -b disk --disk-backend fio --output-format json -o /tmp/perfassess-fio.json
./build/perfassess -b network --network-backend speedtest --output-format json -o /tmp/perfassess-speedtest.json
```

如有可用 iperf3 服务端，再补充。使用节点文件前，先把 `docs/examples/iperf3-servers.txt` 替换为自有或授权节点：

```bash
./build/perfassess -b network --network-backend iperf3 --iperf3-server 1.2.3.4:5201 --output-format json -o /tmp/perfassess-iperf3.json
./build/perfassess -b network --network-backend iperf3 --iperf3-server-file docs/examples/iperf3-servers.txt --output-format json -o /tmp/perfassess-iperf3-file.json
```

## 发布确认

- README 中的安装地址、二进制文件名和 GitHub 仓库地址必须一致。
- `docs/report.schema.json`、`docs/report-schema.md` 与示例报告必须同步。
- 新增或修改报告字段时必须补充契约测试或快照测试。
- JSON 报告必须保留 `summary.assessment_conclusion.evidence` 和 `summary.module_assessments`，否则阻止发布。
- Web 报告必须保留结论优先首屏，包括 `decision-panel`、适用判断、优先建议、模块健康和报告目录。
- 修改评分基准或 `score_calibration.version` 前，必须使用脱敏样本运行 `python3 scripts/calibration-summary.py /path/to/samples --require-policy formal`，并在发布记录中保留样本数量、过滤条件和汇总输出。
- 发布 workflow 必须在上传前执行 Linux amd64 发布二进制冒烟测试。
- 发布 workflow 必须上传 `checksums.txt`，并在上传前校验所有发布二进制的 SHA256。
- 发布后必须执行 `scripts/verify-release-assets.sh --version latest` 或指定 tag；该验证默认会用当前平台可执行的发布二进制跑 basic 自动测评并校验报告目录和压缩包。
- 公开发版前建议至少在一台真实 VPS 上执行 `scripts/vps-acceptance.sh`。
- 公开发版前建议额外执行 `PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh`；正式发布基准或核心评分变更时，优先使用 `PERFASSESS_ACCEPTANCE_MATRIX=low,standard,full PERFASSESS_ACCEPTANCE_STRICT=1 scripts/vps-acceptance.sh`。
- 非严格 VPS 验收允许可选外部后端因资源不足、外部命令失败或环境限制进入 `skipped_reports`，但必须生成包含置信度原因和证据的可解释报告；严格验收不允许跳过或失败。
- 可选外部依赖只允许检测和提示，不允许静默安装。
- 默认无参数一把梭必须在没有可选依赖时仍可运行。
- 发布前不得提交本地 IDE/workspace 文件、日志文件或临时报告。
