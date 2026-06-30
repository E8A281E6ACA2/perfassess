# 发布验证记录

本文档记录候选发布基线的本地验证结果。它不替代真实 VPS 验收，只用于说明某个本地工作区在进入远程测试前已经通过哪些自动门禁。

## 2026-06-30 本地候选基线

验证命令：

```bash
make pre-commit
make release-check
```

验证结果：通过。

复跑记录：

- 2026-06-30：新增远程服务器测试手册后，重新执行 `make pre-commit`，结果通过。
- 2026-06-30：随后重新执行 `make release-check`，结果通过，覆盖发布二进制端到端冒烟和 Linux/macOS/Windows 跨平台构建。
- 2026-06-30：新增远程测试手册冒烟门禁并接入 `make validate` 后，重新执行 `make validate` 和 `make release-check`，结果通过。
- 2026-06-30：补强校准样本隐私契约，检查 `calibration_sample.json` 不嵌入原始报告结构、敏感字段名或公网 IPv4 值；重新执行 `make validate`，结果通过。
- 2026-06-30：将同一隐私契约下沉到 `scripts/calibration-summary.py`，未脱敏或含敏感字段/公网 IPv4 的样本会被拒绝；重新执行 `make validate`，结果通过。
- 2026-06-30：隐私契约下沉后重新执行 `make pre-commit` 和 `make release-check`，结果通过。
- 2026-06-30：新增 `scripts/redact-report.py` 和脱敏工具冒烟门禁，公开归档前可生成 `.redacted.json`；重新执行 `make validate`，结果通过。
- 2026-06-30：报告脱敏工具接入 `make validate` 后，重新执行 `make pre-commit` 和 `make release-check`，结果通过。
- 2026-06-30：报告脱敏工具补强公网 IPv4 字符串扫描，避免路由目标或日志片段中的公网 IP 残留；重新执行 `python3 -m py_compile scripts/redact-report.py`、`bash -n scripts/redact-report-smoke.sh`、`make redact-report-smoke`、`git diff --check`、`bash -n install.sh uninstall.sh scripts/*.sh`、关键 Python 脚本编译、`make validate`、`make pre-commit` 和 `make release-check`，结果通过。
- 2026-06-30：自动测评新增运行中摘要，`summary.md` 和 `console.txt` 会在准备阶段、长步骤运行中、步骤完成和失败时持续可用；新增 `make auto-running-summary-smoke` 并接入 `make validate`。重新执行 `bash -n install.sh uninstall.sh scripts/*.sh`、关键 Python 脚本编译、`make auto-running-summary-smoke`、`make validate` 和 `make pre-commit`，结果通过。
- 2026-06-30：bootstrap 新增低内存预构建二进制路径，`PERFASSESS_BOOTSTRAP_BINARY=auto|1|0` 可控制是否优先下载 Release 二进制，下载成功后跳过本地 Go 编译和单元测试；新增 `make bootstrap-prebuilt-smoke` 并接入 `make validate`。重新执行 `bash -n install.sh uninstall.sh scripts/*.sh`、关键 Python 脚本编译、`make remote-runbook-smoke`、`make bootstrap-prebuilt-smoke` 和 `make validate`，结果通过。
- 2026-06-30：Release workflow 新增 `checksums.txt`，bootstrap 下载预构建二进制时会优先校验 SHA256，校验失败则回退源码构建；`make bootstrap-prebuilt-smoke` 覆盖 checksum 成功路径，远程手册同步说明 checksum 行为。重新执行 `bash -n install.sh uninstall.sh scripts/*.sh`、`make remote-runbook-smoke`、`make bootstrap-prebuilt-smoke`、`make validate`、`make pre-commit` 和 `make release-check`，结果通过。
- 2026-06-30：本地 `make build-all` 同步生成 `build/checksums.txt`，新增 `make release-checksums-smoke` 校验 Linux/macOS/Windows 发布产物 SHA256 清单；Release workflow 上传前也复用该校验脚本。重新执行 `bash -n scripts/release-checksums-smoke.sh`、`make build-all release-checksums-smoke` 和 `make release-check`，结果通过。
- 2026-06-30：新增 GitHub Release 发布手册 `docs/release-runbook.md`，覆盖 tag 创建、workflow 触发、发布资产、checksums 校验、bootstrap 预构建验证和失败回滚；新增 `make release-runbook-smoke` 并接入 `make validate`。重新执行 `bash -n scripts/release-runbook-smoke.sh`、`make release-runbook-smoke`、`make validate` 和 `make release-check`，结果通过。
- 2026-06-30：新增 `scripts/verify-release-assets.sh`，发布后可自动下载 Linux/macOS/Windows 资产、校验 `checksums.txt` 并执行 Linux amd64 版本命令；新增 `make verify-release-assets-smoke` 并接入 `make validate`。重新执行 `bash -n scripts/verify-release-assets.sh scripts/verify-release-assets-smoke.sh`、`make verify-release-assets-smoke`、`make release-runbook-smoke`、`make validate` 和 `make release-check`，结果通过。
- 2026-06-30：Release workflow 的手动触发新增必填 `version` 输入，并校验 `vMAJOR.MINOR.PATCH` 版本格式，避免 `workflow_dispatch` 发布出 `main` 这类错误版本；新增 `make release-workflow-smoke` 并接入 `make validate`。重新执行 `bash -n scripts/release-workflow-smoke.sh scripts/release-runbook-smoke.sh`、`make release-workflow-smoke`、`make release-runbook-smoke`、`make validate` 和 `make release-check`，结果通过。
- 2026-06-30：Release workflow 显式设置 `tag_name` 和 `name` 为校验后的版本号，确保手动触发和 tag 触发都绑定到同一个 Release 版本；`make release-workflow-smoke` 已覆盖该契约。重新执行 `bash -n scripts/release-workflow-smoke.sh`、`make release-workflow-smoke`、`make validate` 和 `make release-check`，结果通过。
- 2026-06-30：Release workflow 增加 `permissions: contents: write`，避免仓库默认 token 权限较严时无法创建 GitHub Release；`make release-workflow-smoke` 已覆盖该契约。重新执行 `bash -n scripts/release-workflow-smoke.sh`、`make release-workflow-smoke`、`make validate` 和 `make release-check`，结果通过。
- 2026-06-30：自动测评产物校验补强目录模式下的压缩包内部 SHA256 校验，并覆盖构建日志进入 `artifact_manifest.json` 和 `perfassess-report.zip`；`artifact_manifest.json` 不作为自引用 artifact，但目录模式会比较目录 manifest 与压缩包内 manifest 是否一致。`make artifact-verify-smoke` 已验证目录篡改、独立 zip 篡改、目录内 zip 篡改和 manifest 漂移都会失败。重新执行 `python3 -m py_compile scripts/verify-artifacts.py`、`bash -n scripts/artifact-verify-smoke.sh`、`make artifact-verify-smoke`、`git diff --check` 和 `go test ./...`，结果通过。
- 2026-06-30：发布二进制冒烟 `scripts/release-smoke.sh` 增加自动测评 basic 档产物验证，使用发布二进制执行 `perfassess-auto.sh` 后校验报告目录和 `perfassess-report.zip`。重新执行 `bash -n scripts/release-smoke.sh`、`make release-smoke`、`git diff --check` 和 `go test ./...`，结果通过。
- 2026-06-30：发布后资产校验 `scripts/verify-release-assets.sh` 增加默认发布二进制 basic 自动测评产物验证，并按当前平台选择可执行 Release 资产；无法执行当前平台资产时会清晰跳过 smoke，`--skip-smoke` 可用于仅校验下载资产和 SHA256。SHA256 校验优先使用 `sha256sum`，缺失或失败时回退 `shasum -a 256`。`make verify-release-assets-smoke` 覆盖 Linux 路径、skip smoke 路径和 shasum fallback 路径。重新执行 `bash -n scripts/verify-release-assets.sh scripts/verify-release-assets-smoke.sh scripts/release-runbook-smoke.sh`、`make verify-release-assets-smoke`、`make release-runbook-smoke`、`git diff --check` 和 `go test ./...`，结果通过。
- 2026-06-30：bootstrap 预构建二进制 checksum 校验同步支持 `sha256sum` 和 `shasum -a 256`，避免 macOS 低内存/预构建路径因缺少 `sha256sum` 跳过校验；`make bootstrap-prebuilt-smoke` 使用 `basic/builtin/quick` 固定轻量档位，覆盖 sha256sum 路径和 shasum fallback 路径，专注验证预构建二进制下载、校验、跳过源码构建和报告生成。重新执行 `bash -n scripts/bootstrap.sh scripts/bootstrap-prebuilt-smoke.sh`、`make bootstrap-prebuilt-smoke`、`make remote-runbook-smoke`、`git diff --check` 和 `go test ./...`，结果通过。
- 2026-06-30：当前候选基线继续收口外部检测证据可信度、主流外部后端结构化错误分类、VPS 验收校准适用性、校准样本 `Collection Plan` / `Score Profile Policies` / `--require-score-profiles` 门禁，并同步发布清单与手册。随后扩展 speedtest、fio 和 sysbench 常见许可/参数/网络/资源失败分类，收紧 iperf3 示例必须使用文档保留地址并声明自有/授权节点的不变量，并补强报告契约 smoke，验证自动测评生成的 `calibration_sample.json` 可被 `calibration-summary.py` 消费；重新执行 `make pre-commit`，结果通过。
- 2026-06-30：当前候选基线在继续推进前重新执行 `make pre-commit` 和 `make release-check`，结果通过；覆盖提交前门禁、发布二进制端到端冒烟、Linux/macOS/Windows 跨平台构建和发布产物 SHA256 清单校验。
- 2026-06-30：校准样本收集链路新增 `manifest.json`、`--require-manifest`、manifest 审计状态输出和发布前正式校准门禁要求后，重新执行 `make release-check`，结果通过；覆盖校准样本收集/汇总 smoke、发布二进制端到端冒烟、跨平台构建和发布产物 SHA256 清单校验。
- 2026-06-30：新增真实 VPS 校准样本矩阵文档并接入 README、路线图和项目不变量检查后，重新执行 `make release-check`，结果通过；覆盖样本矩阵文档入口、不变量门禁、发布二进制端到端冒烟、跨平台构建和发布产物 SHA256 清单校验。
- 2026-06-30：远程服务器测试手册补齐 `calibration_sample.json` 回收、`scripts/calibration-collect.sh` 归档、`manifest.json` 审计和 `docs/calibration-sampling-matrix.md` 入口，并将该链路接入 `remote-runbook-smoke`；重新执行 `bash -n scripts/remote-runbook-smoke.sh scripts/project-invariants-smoke.sh`、`make remote-runbook-smoke`、`make project-invariants-smoke`、`git diff --check`、`go test ./...` 和 `make pre-commit`，结果通过。
- 2026-06-30：发布清单和发布手册新增真实 VPS 校准样本归档规则，要求只取回 `/tmp/perfassess-auto/calibration_sample.json`，并用 `PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1 scripts/calibration-collect.sh` 进入样本池；不得把原始 `default.json`、日志、压缩包或完整报告目录作为校准输入。同步补强 `release-runbook-smoke` 后，重新执行 `bash -n scripts/release-runbook-smoke.sh scripts/remote-runbook-smoke.sh scripts/project-invariants-smoke.sh`、`make release-runbook-smoke`、`make remote-runbook-smoke`、`make project-invariants-smoke`、`git diff --check`、`go test ./...` 和 `make validate`，结果通过。

当前候选基线建议按主题拆分提交。完整文件边界、验证命令和注意事项见 [候选基线拆分提交计划](candidate-baseline-split-plan.md)：

1. `feat(report): 增加外部检测证据可信度摘要`
   - 范围：`internal/models/evidence.go`、IP 质量、流媒体、AI 服务、路由模型与报告/Web 展示、报告契约测试、schema 和示例报告。
2. `fix(tests): 细化主流后端错误分类`
   - 范围：`internal/tests/benchmark_error.go`、iperf3、speedtest、fio、sysbench 相关解析与单元测试。
3. `scripts(calibration): 增加脱敏校准样本收集链路`
   - 范围：`scripts/calibration-collect.sh`、`scripts/calibration-collect-smoke.sh`、`scripts/calibration-summary.py`、`.gitignore`、Makefile 校准 smoke。
4. `docs(calibration): 补齐真实 VPS 样本矩阵和归档规范`
   - 范围：`docs/calibration-dataset-policy.md`、`docs/calibration-sampling-matrix.md`、README 校准说明、远程测试手册样本回收说明。
5. `docs(release): 收紧发布和远程验收门禁`
   - 范围：`docs/release-checklist.md`、`docs/release-runbook.md`、`scripts/release-runbook-smoke.sh`、`scripts/remote-runbook-smoke.sh`、`scripts/project-invariants-smoke.sh`。
6. `test(acceptance): 增强 VPS 验收摘要和报告覆盖检查`
   - 范围：`scripts/vps-acceptance.sh`、`scripts/vps-acceptance-summary-smoke.sh`、`docs/vps-acceptance.md`、相关 Makefile 入口。

覆盖范围：

- Go 格式、JSON schema、项目不变量、脚本语法和关键 Python 脚本编译。
- 远程服务器测试手册冒烟，确保 README 入口、bootstrap 复制命令、Web 访问、失败排查、产物位置、验收矩阵、iperf3 授权节点和清理流程没有漂移。
- 全量 `go test ./...`。
- 当前平台构建、CLI 冒烟、报告契约、Web 报告渲染契约。
- 脱敏校准样本隐私契约，包括 `redacted=true`、`privacy_note`、环境字段白名单和敏感值扫描。
- 校准汇总工具拒绝未脱敏或含敏感内容的样本，防止坏样本进入评分回测。
- JSON 报告脱敏工具冒烟，确保 IP、ISP、ASN、地理位置、反向 DNS 和路由 hop 细节会从公开归档报告中移除。
- 自动测评产物清单、目录 SHA256 校验、目录模式下的压缩包内部 SHA256 校验、目录 manifest 与压缩包内 manifest 一致性、压缩包 SHA256 校验和篡改负向测试。
- 校准样本汇总工具冒烟。
- 发布二进制端到端冒烟，包括版本、help、依赖检查、quick JSON、报告对比、目录排序和自动测评产物校验；发布后资产校验也会用下载的当前平台可执行二进制跑 basic 自动测评并校验报告包。
- Linux amd64/arm64、macOS amd64/arm64、Windows amd64 跨平台构建。

真实 VPS 验收：

```bash
PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh
```

验证结果：通过。

验收摘要：

- 输出目录：`/tmp/perfassess-acceptance`
- 验收矩阵：`low,standard`
- 生成报告：`quick.json`、`network.json`、`cpu.json`、`low-basic.json`、`standard-builtin.json`、`compare.json`、`compare-dir.json`
- 可选后端报告：`cpu-sysbench.json`、`memory-sysbench.json`、`disk-fio.json`
- 跳过报告：无
- `low-basic.json`：总分 89.23，等级良好，置信度 medium
- `standard-builtin.json`：总分 88.06，等级良好，置信度 medium

如果本次改动影响评分阈值、主流后端或 full 档位，发布前继续执行：

```bash
PERFASSESS_ACCEPTANCE_MATRIX=low,standard,full PERFASSESS_ACCEPTANCE_STRICT=1 scripts/vps-acceptance.sh
```

本次另行执行了 full 主流口径验收：

```bash
PERFASSESS_ACCEPTANCE_MATRIX=full scripts/vps-acceptance.sh
```

验证结果：通过。

full 验收摘要：

- 输出目录：`/tmp/perfassess-acceptance`
- 验收矩阵：`full`
- 生成报告：`quick.json`、`network.json`、`cpu.json`、`full-mainstream.json`、`compare.json`、`compare-dir.json`
- 可选后端报告：`cpu-sysbench.json`、`memory-sysbench.json`、`disk-fio.json`
- 跳过报告：无
- `full-mainstream.json`：总分 96.25，等级优秀，核心测试成功 4 项，失败 0 项，跳过 0 项，置信度 medium
- 降置信原因：未提供授权 iperf3 节点时网络上传仍为估算值

随后执行严格矩阵验收：

```bash
PERFASSESS_ACCEPTANCE_MATRIX=low,standard,full PERFASSESS_ACCEPTANCE_STRICT=1 scripts/vps-acceptance.sh
```

验证结果：通过。

严格矩阵摘要：

- 输出目录：`/tmp/perfassess-acceptance`
- 验收矩阵：`low,standard,full`
- 生成报告：`quick.json`、`network.json`、`cpu.json`、`low-basic.json`、`standard-builtin.json`、`full-mainstream.json`、`compare.json`、`compare-dir.json`
- 可选后端报告：`cpu-sysbench.json`、`memory-sysbench.json`、`disk-fio.json`
- 跳过报告：无
- `low-basic.json`：总分 88.59，等级良好，核心测试成功 4 项，失败 0 项，跳过 0 项，置信度 medium
- `standard-builtin.json`：总分 88.55，等级良好，核心测试成功 4 项，失败 0 项，跳过 0 项，置信度 medium
- `full-mainstream.json`：总分 96.25，等级优秀，核心测试成功 4 项，失败 0 项，跳过 0 项，置信度 medium
- 降置信原因：`low-basic` 和 `standard-builtin` 使用内置后端且上传为估算值；`full-mainstream` 未提供授权 iperf3 节点时网络上传仍为估算值

后续收紧真实 VPS 验收契约后，`scripts/vps-acceptance.sh` 会强制检查 `cpu`、`memory`、`disk`、`network`、`route`、`ip_quality`、`streaming` 和 `ai_services` 八个模块的模块可信度。严格矩阵还会在运行前和每个矩阵项开始前检查可用内存，默认要求 `MemAvailable >= 768 MB`。

在当前共享开发机内存被其他服务占用时，严格矩阵会快速失败并给出资源提示，例如：

```text
VPS acceptance failed: strict matrix requires at least 768 MB available memory
```

该行为符合预期：严格模式要求核心测试完整成功；如果需要保留降级报告，应使用非严格矩阵。已验证非严格 low 矩阵可以在资源紧张时通过，并在 `skipped_reports` 中记录可选后端或核心模块的可解释跳过。

CI 与 release workflow 均调用 `make validate`，因此严格验收资源预检冒烟会在远程 CI 和发布流程中自动执行。

注意事项：

- 本地验证不证明公网访问、低内存机器、外部测速源、流媒体平台或授权 iperf3 节点在真实环境一定成功。
- 若需要提交真实 VPS 验收产物，只提交按 `docs/examples/acceptance/README.md` 脱敏后的摘要或样例，不提交原始报告目录。
- 若后续继续修改 CLI、报告 schema、自动脚本、Web 报告或发布 workflow，需要重新运行 `make pre-commit`；公开发版前重新运行 `make release-check`。
