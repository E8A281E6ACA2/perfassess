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
- 全量单元测试
- 当前平台构建
- CLI help 冒烟测试
- 依赖检查冒烟测试

发布前运行：

```bash
make release-check
```

`make release-check` 在 `make validate` 基础上额外覆盖：

- Linux amd64/arm64 构建
- macOS amd64/arm64 构建
- Windows amd64 构建
- 快速 JSON 报告生成与 JSON 语法检查

## 手动冒烟

在真实 VPS 或测试机上建议至少运行：

```bash
./build/perfassess --quick --output-format json -o /tmp/perfassess-quick.json
./build/perfassess -b network --output-format json -o /tmp/perfassess-network.json
./build/perfassess check-deps
```

如果机器已安装可选依赖，再补充：

```bash
./build/perfassess --full --output-format json -o /tmp/perfassess-full.json
./build/perfassess -b disk --disk-backend fio --output-format json -o /tmp/perfassess-fio.json
./build/perfassess -b network --network-backend speedtest --output-format json -o /tmp/perfassess-speedtest.json
```

如有可用 iperf3 服务端，再补充：

```bash
./build/perfassess -b network --network-backend iperf3 --iperf3-server 1.2.3.4:5201 --output-format json -o /tmp/perfassess-iperf3.json
```

## 发布确认

- README 中的安装地址、二进制文件名和 GitHub 仓库地址必须一致。
- `docs/report.schema.json`、`docs/report-schema.md` 与示例报告必须同步。
- 新增或修改报告字段时必须补充契约测试或快照测试。
- 可选外部依赖只允许检测和提示，不允许静默安装。
- 默认无参数一把梭必须在没有可选依赖时仍可运行。
- 发布前不得提交本地 IDE/workspace 文件、日志文件或临时报告。
