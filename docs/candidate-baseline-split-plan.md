# 候选基线拆分提交计划

本文档用于把当前企业级候选基线从“大工作区改动”拆成可审计、可回滚、可测试的主题提交。它只描述提交组织方式，不替代 `make pre-commit`、`make release-check` 或真实 VPS 验收。

## 拆分原则

- 每个提交只解决一个清晰主题。
- 提交信息使用 `type(scope): 中文描述`。
- 拆分时优先保持每个提交可独立理解；如果某个文件同时涉及多个主题，先用 `git diff` 复核上下文，再决定是否需要分块暂存。
- 每个提交前至少运行 `git diff --check` 和与该主题相关的 smoke；最终合并前必须重新运行 `make pre-commit`。
- 不提交 `calibration-samples/`、本地报告、日志、构建产物或真实服务器原始输出。

## 推荐提交顺序

### 1. 证据化报告

推荐提交信息：

```text
feat(report): 增加外部检测证据可信度摘要
```

候选文件：

```text
internal/models/evidence.go
internal/models/ai_service.go
internal/models/ip_quality.go
internal/models/route.go
internal/models/streaming.go
internal/tests/ai.go
internal/tests/ai_test.go
internal/tests/ip_quality.go
internal/tests/ip_quality_test.go
internal/tests/route_trace.go
internal/tests/route_trace_test.go
internal/tests/streaming.go
internal/tests/streaming_test.go
internal/reporter/reporter.go
internal/reporter/web_server.go
internal/reporter/report_contract_test.go
scripts/perfassess-progress-server.py
scripts/report-contract-smoke.sh
docs/report-schema.md
docs/report.schema.json
docs/examples/report-json-sample.json
```

建议验证：

```bash
go test ./internal/reporter ./internal/tests
make report-contract-smoke
git diff --check
```

注意点：

- JSON schema、示例报告、控制台报告和 Web 报告必须同时更新。
- 不要把启发式检测包装成强结论；证据摘要必须保留限制说明。
- `docs/report-schema.md` 同时包含主流后端错误分类说明；第 1 组只暂存 `evidence_summary`、路由、流媒体、AI 和 IP 质量证据相关段落。
- `scripts/report-contract-smoke.sh` 同时包含校准样本消费检查；第 1 组只暂存报告/Web/证据契约相关段落，校准汇总检查应放入第 3 组。

### 2. 主流后端错误分类

推荐提交信息：

```text
fix(tests): 细化主流后端错误分类
```

候选文件：

```text
internal/tests/benchmark_error.go
internal/tests/cpu_sysbench_test.go
internal/tests/disk_test.go
internal/tests/network_iperf3.go
internal/tests/network_test.go
docs/report-schema.md
docs/report.schema.json
```

建议验证：

```bash
go test ./internal/tests
git diff --check
```

注意点：

- 区分依赖缺失、无效配置、权限不足、资源不足、网络不可达、解析失败、超时、命令失败和运行时错误。
- 错误分类用于解释报告，不应静默把外部后端失败归因成机器性能差。
- `docs/report-schema.md` 中的 iperf3 矩阵错误字段和外部后端分类说明应随本组暂存。
- `docs/report.schema.json` 中 `iperf3_matrix_<n>_error_category/error_stage/error_hint` 的 schema 扩展应随本组暂存。
- `internal/tests/network_iperf3.go` 同时包含授权节点示例文案更新；若严格拆分，错误分类逻辑随本组，纯文档保留地址/授权示例文案可随第 7 组暂存。

### 3. 脱敏校准样本收集链路

推荐提交信息：

```text
scripts(calibration): 增加脱敏校准样本收集链路
```

候选文件：

```text
.gitignore
Makefile
scripts/calibration-collect.sh
scripts/calibration-collect-smoke.sh
scripts/calibration-summary.py
scripts/calibration-summary-smoke.sh
```

建议验证：

```bash
bash -n scripts/calibration-collect.sh scripts/calibration-collect-smoke.sh scripts/calibration-summary-smoke.sh
python3 -m py_compile scripts/calibration-summary.py
make calibration-collect-smoke
make calibration-summary-smoke
git diff --check
```

注意点：

- 收集助手只接受脱敏 `calibration_sample.json`。
- 正式样本池必须支持 `PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1` 和 manifest SHA256 审计。
- `calibration-samples/` 必须保持在 `.gitignore` 中。
- `scripts/report-contract-smoke.sh` 中调用 `scripts/calibration-summary.py` 消费 `calibration_sample.json` 的检查应随本组暂存。
- `Makefile` 中只有 `calibration-collect-smoke` 目标、`validate` 对该目标的引用和 help 文案随本组暂存；`candidate-baseline-split-smoke` 属于第 8 组，`vps-acceptance-summary-smoke` 属于第 6 组。
- `README.md`、`docs/remote-test-runbook.md` 和 `docs/calibration-dataset-policy.md` 中的人工流程说明更适合随第 4 组文档提交；第 3 组只提交脚本、工具和自动化验证。

### 4. 真实 VPS 校准文档

推荐提交信息：

```text
docs(calibration): 补齐真实 VPS 样本矩阵和归档规范
```

候选文件：

```text
README.md
docs/calibration-dataset-policy.md
docs/calibration-sampling-matrix.md
docs/remote-test-runbook.md
docs/mainstream-vps-benchmark-analysis.md
docs/project-analysis-and-roadmap.md
```

建议验证：

```bash
make remote-runbook-smoke
make project-invariants-smoke
git diff --check
```

注意点：

- 远程样本回收只复制 `/tmp/perfassess-auto/calibration_sample.json`。
- 不把原始 `default.json`、日志、压缩包或完整报告目录作为校准输入。
- candidate 阶段只观察，不直接调整评分阈值；formal 达标后才允许提出 `score_calibration.version` 变更。
- `README.md` 中校准样本矩阵入口、收集助手说明、`--require-manifest` 和 `--require-score-profiles` 说明随本组；候选基线拆分计划入口随第 8 组，iperf3 文档保留地址示例随第 7 组。
- `docs/mainstream-vps-benchmark-analysis.md` 和 `docs/project-analysis-and-roadmap.md` 中真实样本、校准矩阵和下一阶段采样路线随本组；证据可信度、外部后端错误分类和 VPS 验收摘要已落地能力描述可随第 1、第 2、第 6 组分别暂存。
- `docs/remote-test-runbook.md` 中校准样本回收、`scp`、`calibration-collect.sh`、manifest 和样本矩阵引用随本组；iperf3 授权节点示例地址随第 7 组。

### 5. 发布和远程验收门禁

推荐提交信息：

```text
docs(release): 收紧发布和远程验收门禁
```

候选文件：

```text
docs/release-checklist.md
docs/release-runbook.md
docs/release-validation-log.md
scripts/release-runbook-smoke.sh
scripts/remote-runbook-smoke.sh
scripts/project-invariants-smoke.sh
```

建议验证：

```bash
make release-runbook-smoke
make remote-runbook-smoke
make project-invariants-smoke
git diff --check
```

注意点：

- 发布前门禁必须要求 `--require-manifest`、`--require-score-profiles` 和 `--require-policy formal`。
- 发布记录只能记录脱敏摘要和验证结论，不提交真实服务器原始报告。
- `docs/release-checklist.md` 中发布校准门禁、正式样本要求、`make validate`/`make pre-commit`/`make release-check` 覆盖说明随本组；iperf3 手动冒烟示例随第 7 组，候选基线拆分计划入口随第 8 组。
- `docs/release-runbook.md` 中发布前正式校准命令、只回收 `calibration_sample.json`、禁止原始报告进入样本池的规则随本组。
- `docs/release-validation-log.md` 中发布门禁复跑记录随本组；候选拆分建议段落随第 8 组，证据可信度、错误分类、校准链路和验收摘要记录应分别随第 1、第 2、第 3/4 和第 6 组暂存。
- `scripts/release-runbook-smoke.sh` 中 release checklist/runbook 的正式校准门禁检查随本组。
- `scripts/remote-runbook-smoke.sh` 同时覆盖第 4 组样本回收和第 7 组 iperf3 示例；第 5 组只暂存用于约束远程验收门禁不漂移的检查。
- `scripts/project-invariants-smoke.sh` 是跨组护栏：正式校准门禁检查随本组，样本矩阵入口随第 4 组，VPS 验收摘要覆盖随第 6 组，iperf3 文档保留地址检查随第 7 组，候选拆分计划入口随第 8 组。

### 6. VPS 验收摘要和覆盖检查

推荐提交信息：

```text
test(acceptance): 增强 VPS 验收摘要和报告覆盖检查
```

候选文件：

```text
Makefile
scripts/vps-acceptance.sh
scripts/vps-acceptance-summary-smoke.sh
docs/vps-acceptance.md
```

建议验证：

```bash
bash -n scripts/vps-acceptance.sh scripts/vps-acceptance-summary-smoke.sh
make vps-acceptance-summary-smoke
make vps-acceptance-strict-preflight-smoke
git diff --check
```

注意点：

- 严格验收资源不足时应提前失败并解释原因。
- 非严格验收允许可解释跳过，但必须保留报告覆盖、置信度原因和样本适用性。
- `scripts/vps-acceptance.sh` 中 `summary.json`、`report_coverage`、`calibration_readiness`、`sample_policy` 和严格资源预检逻辑随本组。
- `scripts/vps-acceptance-summary-smoke.sh` 与 `scripts/vps-acceptance-strict-preflight-smoke.sh` 随本组。
- `Makefile` 中 `vps-acceptance-summary-smoke` 目标、`validate` 对该目标的引用和 help 文案随本组；`calibration-collect-smoke` 属于第 3 组，`candidate-baseline-split-smoke` 属于第 8 组。
- `docs/vps-acceptance.md` 中验收摘要、报告覆盖、`summary.json`、`sample_policy`、严格/非严格矩阵说明随本组；校准样本收集助手和样本池归档流程说明可随第 4 组文档提交，iperf3 文档保留地址示例随第 7 组。
- `scripts/project-invariants-smoke.sh` 中 `docs/vps-acceptance.md`、`scripts/vps-acceptance.sh`、`Makefile` 的验收覆盖检查随本组。

### 7. 一键启动与远程体验细节

推荐提交信息：

```text
scripts(bootstrap): 优化远程启动和端口说明
```

候选文件：

```text
README.md
scripts/bootstrap.sh
docs/examples/iperf3-servers.txt
docs/remote-test-runbook.md
docs/vps-acceptance.md
internal/tests/network_iperf3.go
```

建议验证：

```bash
bash -n scripts/bootstrap.sh
make bootstrap-prebuilt-smoke
make remote-runbook-smoke
git diff --check
```

注意点：

- iperf3 示例必须使用文档保留地址，并明确自有或授权节点。
- bootstrap 只能在用户显式运行时安装系统依赖；项目默认 CLI 不应静默安装依赖。
- `README.md`、`docs/remote-test-runbook.md` 和 `docs/vps-acceptance.md` 中纯 iperf3 示例地址、授权节点说明、公共节点边界随本组；校准样本文档随第 4 组，验收摘要说明随第 6 组。
- `internal/tests/network_iperf3.go` 中错误分类逻辑随第 2 组；只把文档保留地址、授权节点提示这类用户文案随本组暂存。
- `scripts/project-invariants-smoke.sh` 中禁止旧 iperf3 示例地址、要求文档保留地址和授权声明的检查随本组。

### 8. 候选基线拆分护栏

推荐提交信息：

```text
docs(release): 增加候选基线拆分提交计划
```

候选文件：

```text
docs/candidate-baseline-split-plan.md
scripts/candidate-baseline-split-smoke.sh
README.md
docs/release-checklist.md
docs/release-validation-log.md
scripts/project-invariants-smoke.sh
Makefile
```

建议验证：

```bash
bash -n scripts/candidate-baseline-split-smoke.sh
make candidate-baseline-split-smoke
make project-invariants-smoke
git diff --check
```

注意点：

- 这组提交只维护拆分计划和计划入口，不应混入新的功能实现。
- 拆分计划必须覆盖当前候选基线全部改动文件；如果新增文件无法归入前七组，应先更新本计划再提交。

## 最终收口验证

全部主题提交完成后，重新执行：

```bash
git status --short --branch
git diff --check
go test ./...
make pre-commit
make release-check
```

如果发布评分阈值或 `score_calibration.version`，还必须基于正式样本池执行：

```bash
python3 scripts/calibration-summary.py calibration-samples \
  --require-manifest \
  --require-score-profiles vps,server,workstation \
  --require-policy formal \
  --format markdown \
  -o calibration-samples/formal-all-profiles.md
```

真实发布前再按 `docs/release-runbook.md` 完成 tag、Release workflow、资产校验和发布后 bootstrap 验证。
