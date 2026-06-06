# VPS 真实环境验收

本文档用于在真实 VPS 或测试机上验证项目是否达到可运行、可解释、可复现的最低发布门槛。

## 目标

- 验证最终二进制可以直接运行
- 验证默认无可选依赖时仍能完成核心验收
- 验证 JSON 报告包含关键摘要、评分校准、分享模板和评分拆解
- 验证报告对比与历史趋势链路可用
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

## iperf3 验收

`iperf3` 不内置公共节点。请只使用自有或授权节点。

单节点：

```bash
PERFASSESS_IPERF3_SERVER=1.2.3.4:5201 scripts/vps-acceptance.sh
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
- `history-list.json`
- `history-trend.json`
- `summary.md`

每份核心 JSON 报告必须包含：

- `summary.benchmark_profile`
- `summary.confidence_level`
- `summary.score_calibration`
- `summary.score_breakdown`
- `summary.vps_benchmark_summary`
- `summary.share_templates`

`summary.score_calibration.version` 当前必须为 `2026-06-v1`。

## 通过标准

- 脚本退出码为 `0`
- 最后一行包含 `VPS acceptance passed`
- `summary.md` 能看到二进制版本、输出目录、总分、等级、置信度、评分基准和校准版本
- 可选依赖缺失时不应导致核心验收失败
- 已安装可选依赖时，对应后端报告应生成并通过 JSON 校验

## 失败处理

- 二进制不可执行：先构建或指定 `PERFASSESS_BINARY`
- `python3` 缺失：安装 Python 3 后重试
- 外部后端失败：先查看 `check-deps.txt` 和对应 `*.stdout.txt`
- `iperf3` 失败：确认节点可访问、端口正确且属于自有或授权节点
- JSON 字段缺失：说明报告契约可能被破坏，应阻止发布

## 脱敏归档

真实 VPS 输出可能包含公网 IP、ISP、地理位置、CPU 型号等信息。归档前请脱敏：

- 删除或替换 `public_ip`
- 删除或替换 `speedtest_external_ip`
- 按需替换 `isp`、`location`、`country_code`
- 保留 `summary.score_calibration`、`summary.score_breakdown`、`summary.confidence_level` 等结构字段

脱敏样例格式见 [examples/acceptance/README.md](examples/acceptance/README.md)。
