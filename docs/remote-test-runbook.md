# 远程服务器测试手册

本文档用于指导在另一台全新服务器上拉取、运行、查看报告、回传排查信息和清理环境。它面向真实 VPS 验收，不替代开发机上的 `make validate` 和 `make release-check`。

## 推荐流程

普通 VPS 首次测试建议运行标准档位，并打开实时 Web 页面：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile standard --web --port 8080
```

如果只是想生成完整报告，不需要开发验收流程：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile full --web --skip-acceptance
```

如果要尽量贴近主流 VPS 测评口径，使用 `sysbench` 和 `fio`：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile full --quality mainstream --web --skip-acceptance
```

`mainstream` 档位会安装并使用主流后端。没有自有或授权 iperf3 节点时，网络上传仍会标记为估算值，报告置信度会说明原因。

## 512MB 或低内存机器

低内存机器建议先跑标准一把梭，让 bootstrap 自动降级：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash
```

脚本会在低内存模式下优先尝试下载 GitHub Release 中的预构建二进制，并在 Release 提供 `checksums.txt` 时校验 SHA256；校验会优先使用 `sha256sum`，没有时使用 macOS 常见的 `shasum -a 256`。成功后跳过本地 Go 编译和 `go test ./...`，直接运行测评。这是 512MB 机器的推荐路径，可以避开 Go 编译器被系统杀掉的问题。

如果当前仓库还没有可用 Release 二进制，bootstrap 会回退到源码构建：降低 Go 构建并发，Linux 主机会尽量创建临时 swap，并默认跳过 `go test ./...`。临时 swap 会在脚本退出时清理。

如果要强制尝试预构建二进制：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | PERFASSESS_BOOTSTRAP_BINARY=1 bash
```

如果要禁用预构建二进制、始终源码构建：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | PERFASSESS_BOOTSTRAP_BINARY=0 bash
```

如果要显式禁用临时 swap：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | PERFASSESS_BOOTSTRAP_SWAP=0 bash
```

## Web 页面访问

`--web` 默认从 8080 开始监听。如果 8080 被占用，会继续尝试 8081、8082，直到找到可用端口。终端会打印最终端口。

浏览器不要访问 `0.0.0.0:8080`。`0.0.0.0` 只是监听所有网卡的地址，应使用服务器公网 IP：

```text
http://SERVER_PUBLIC_IP:8080
```

如果不想开放安全组或防火墙端口，可以在本地电脑使用 SSH 端口转发。端口以终端实际显示为准：

```bash
ssh -L 8080:localhost:8080 root@SERVER_PUBLIC_IP
```

然后在本地浏览器打开：

```text
http://localhost:8080
```

如需测试完成后自动关闭 Web 页面：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile standard --web --web-ttl 600
```

## 报告位置

默认输出目录是 `/tmp/perfassess-auto/`。测试完成后优先查看：

```bash
cat /tmp/perfassess-auto/summary.md
```

常用文件：

- `/tmp/perfassess-auto/summary.md`: Markdown 摘要，适合直接复制。
- `/tmp/perfassess-auto/console.txt`: 终端纯文本报告。
- `/tmp/perfassess-auto/default.json`: 完整 JSON 报告。
- `/tmp/perfassess-auto/default.txt`: 完整文本报告。
- `/tmp/perfassess-auto/build.stdout.txt`: 源码构建标准输出。
- `/tmp/perfassess-auto/build.stderr.log`: 源码构建错误输出，低内存或 Go 编译失败时优先查看。
- `/tmp/perfassess-auto/artifact_manifest.json`: 产物清单，包含大小和 SHA256。
- `/tmp/perfassess-auto/perfassess-report.zip`: 报告压缩包，包含报告、构建日志、测评日志和验收摘要。
- `/tmp/perfassess-auto/hardware_quality.json`: 硬件质量模块。
- `/tmp/perfassess-auto/net_quality.json`: 网络质量模块。
- `/tmp/perfassess-auto/ip_quality.json`: IP 质量模块。
- `/tmp/perfassess-auto/route_trace.json`: 路由追踪模块。
- `/tmp/perfassess-auto/backroute_trace.json`: 国内方向参考模块，不是真实回程。
- `/tmp/perfassess-auto/calibration_sample.json`: 脱敏校准样本。

校验报告目录或压缩包：

```bash
cd ~/perfassess
python3 scripts/verify-artifacts.py /tmp/perfassess-auto
python3 scripts/verify-artifacts.py /tmp/perfassess-auto/perfassess-report.zip
```

## 失败时怎么排查

如果脚本中途失败，也应生成失败摘要：

```bash
cat /tmp/perfassess-auto/summary.md
```

优先回传这些文件：

```text
/tmp/perfassess-auto/summary.md
/tmp/perfassess-auto/console.txt
/tmp/perfassess-auto/build.stdout.txt
/tmp/perfassess-auto/build.stderr.log
/tmp/perfassess-auto/default.stdout.txt
/tmp/perfassess-auto/default.stdout.txt.stderr.log
/tmp/perfassess-auto/check-deps.txt
/tmp/perfassess-auto/progress.json
```

如果报告压缩包已经生成，优先回传：

```text
/tmp/perfassess-auto/perfassess-report.zip
```

真实服务器输出可能包含公网 IP、ISP、地区、ASN、CPU 型号等信息。公开提交前应先脱敏；内部排查可以直接用压缩包。

如果要公开归档或提交单个 JSON 报告，先生成脱敏版本：

```bash
cd ~/perfassess
python3 scripts/redact-report.py /tmp/perfassess-auto/default.json
```

这会生成 `/tmp/perfassess-auto/default.redacted.json`。也可以对整个报告目录生成脱敏副本：

```bash
cd ~/perfassess
python3 scripts/redact-report.py /tmp/perfassess-auto
```

## 校准样本回收

如果本次远程测试用于后续评分校准，优先保存脱敏样本：

```text
/tmp/perfassess-auto/calibration_sample.json
```

这个文件由自动测评生成，标记为 `redacted=true`，用于评分阈值回测；它不应包含公网 IP、ISP、ASN、精确地理位置、路由 hop 或原始日志。公开归档时建议改名为：

```text
calibration_sample.redacted.json
```

多台机器样本放到同一个目录后，可以在开发机上汇总：

```bash
python3 scripts/calibration-summary.py /path/to/samples --format markdown -o calibration-summary.md
python3 scripts/calibration-summary.py /path/to/samples --require-policy candidate
```

样本分级和正式校准准入规则见 `docs/calibration-dataset-policy.md`。`builtin`、低置信、低内存降级或未完成模块的样本可以用于研发观察，但不能直接用于发布新评分阈值。

## 真实 VPS 验收

已经克隆仓库时，可以运行真实 VPS 验收矩阵：

```bash
cd ~/perfassess
PERFASSESS_ACCEPTANCE_MATRIX=low,standard scripts/vps-acceptance.sh
cat /tmp/perfassess-acceptance/summary.md
```

发布前或评分基准变更前，资源足够的机器再跑严格矩阵：

```bash
cd ~/perfassess
PERFASSESS_ACCEPTANCE_MATRIX=low,standard,full \
PERFASSESS_ACCEPTANCE_STRICT=1 \
scripts/vps-acceptance.sh
```

严格模式默认要求 `MemAvailable >= 768 MB`。低于门槛时会提前失败，这是预期行为；低内存机器应使用非严格矩阵生成可解释降级报告。

## iperf3 授权节点

Perfassess 不内置公共 iperf3 节点。只使用自有或授权节点。

单节点：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh \
  | PERFASSESS_IPERF3_SERVER=1.2.3.4:5201 bash -s -- --profile full --quality mainstream
```

节点文件必须声明授权，例如 `auth=owned` 或 `auth=authorized`。示例见 `docs/examples/iperf3-servers.txt`。

## 清理

只清理构建产物和自动测评输出：

```bash
cd ~/perfassess
scripts/bootstrap.sh --clean
```

连同默认克隆目录一起清理：

```bash
cd ~/perfassess
scripts/bootstrap.sh --clean-all
```

如果希望 Web 查看窗口结束后自动删除源码、构建产物和报告：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh | bash -s -- --profile standard --destroy-after-web --web-ttl 600
```

`--destroy-after-web` 会删除本次报告，适合一次性临时测评。需要保存结果时不要使用这个选项。

## 回传测试结果模板

测试完成后，可以复制下面几项给开发侧：

```text
测试机器:
运行命令:
是否开启 Web:
输出目录:
summary.md 是否生成:
perfassess-report.zip 是否生成:
总分/等级/置信度:
失败或跳过模块:
遇到的问题:
```
