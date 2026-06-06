# 真实回程探针设计

本文档定义 Perfassess 后续“真实回程”能力的设计边界和实施路线。

当前项目已经支持 `--route-trace`，它执行的是被测服务器到目标节点的本机出站路径。这个结果适合判断本机出口路径、国内方向参考和中间跳点可见性，但不等同于真实回程。真实回程必须由远端探针从外部网络反向追踪被测服务器。

## 目标

- 明确区分本机出站路由、国内方向参考和真实回程。
- 支持由远端探针追踪被测服务器公网 IP，并把结果汇总进报告。
- 不依赖固定第三方平台，优先支持用户自有探针节点。
- 不把远端探针设计成可执行任意命令的后门。
- 报告继续保持可复制、可离线保存、可通过 Web 页面查看。

## 非目标

- 不内置未经授权的公共探针节点。
- 不伪造“真实回程”结果。
- 不要求默认一把梭必须执行真实回程。
- 不在第一阶段维护中心化服务、账号体系或公共结果库。

## 术语

- **被测端**：正在运行 Perfassess 的服务器。
- **探针端**：另一台服务器或网络节点，从它的网络方向追踪被测端公网 IP。
- **出站路由**：被测端到外部目标的 traceroute/tracert 结果。
- **真实回程**：探针端到被测端公网 IP 的 traceroute/tracert 结果。
- **国内方向参考**：被测端到国内目标的出站路径，仅作方向参考，不是真实回程。

## 总体架构

```text
被测端 Perfassess
  ├─ 采集系统/性能/IP/网络质量
  ├─ 执行本机出站 route_trace
  ├─ 生成一次性回程任务 token
  ├─ 等待或导入探针结果
  └─ 汇总 return_route_report

探针端 Perfassess Probe
  ├─ 接收目标 IP、任务 token、可选探针标签
  ├─ 从探针端执行 traceroute/tracert 到被测端公网 IP
  ├─ 生成结构化 ProbeResult
  └─ 回传或导出 JSON
```

## 分阶段实施

### Phase 1：手动探针导入

这是最稳的第一阶段，不需要开放被测端端口，也不需要中心服务器。

用户在被测端运行：

```bash
./build/perfassess --return-route-file ./probe-cn-telecom.json --return-route-file ./probe-cn-unicom.json
```

用户在探针端运行：

```bash
./build/perfassess probe-route --target SERVER_PUBLIC_IP --probe-name cn-telecom-shanghai -o probe-cn-telecom.json
```

然后把探针 JSON 拷回被测端导入。

Phase 1 需要新增：

- CLI：
  - `probe-route`
  - `--target`
  - `--probe-name`
  - `--probe-region`
  - `--probe-isp`
  - `--output`
  - `--return-route-file`
- 报告：
  - `summary.return_route_report`
  - `return_route.json`
  - `return_route.txt`
  - Web 报告“真实回程”模块
- 自动脚本：
  - 若存在 `PERFASSESS_RETURN_ROUTE_FILES`，自动导入
  - 若不存在，显示“未执行真实回程，需要远端探针”

优点：

- 安全边界清楚。
- 不需要被测端开放端口。
- 可离线保存和复现。
- 适合用户自己有国内三网、海外节点时使用。

缺点：

- 操作上需要把探针 JSON 复制回来。

### Phase 2：一次性回传模式

这一阶段减少手动复制。

被测端运行：

```bash
./build/perfassess --return-route-listen --return-route-port 8090
```

被测端输出一次性命令：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/probe-route.sh \
  | bash -s -- --target SERVER_PUBLIC_IP --post http://SERVER_PUBLIC_IP:8090/api/return-route --token TOKEN
```

探针端运行该命令后：

- 执行 traceroute/tracert 到被测端公网 IP。
- 把结果 POST 到被测端。
- 被测端校验 token 后写入临时结果目录。
- 自动报告等待指定时间，例如 10 分钟。

安全要求：

- token 至少 128 bit 随机。
- 只接受 `POST /api/return-route`。
- token 只在本次运行有效。
- 请求体大小限制。
- 不允许探针端要求被测端执行命令。
- 默认监听端口只在用户显式开启时启用。

优点：

- 使用体验接近主流脚本的“远端结果回传”。
- 不需要中心服务。

缺点：

- 被测端必须能被探针端访问到回传端口。
- 用户需要开放安全组或使用 SSH 反向转发。

### Phase 3：自建探针服务

适合团队或长期维护者。

部署多个探针节点：

```bash
perfassess probe-server --region cn-east --isp telecom --token-file /etc/perfassess/probe.token
```

被测端调用：

```bash
./build/perfassess --return-route-probe https://probe.example.com --return-route-token TOKEN
```

探针服务只暴露固定 API：

- `POST /v1/trace`
- 输入：目标 IP、任务 ID、签名
- 输出：结构化 traceroute 结果

安全要求：

- 服务端限制目标端口和命令参数。
- 只允许 traceroute/tracert，不允许任意 shell。
- 目标 IP 可以设置白名单/公网限制。
- 单 IP 速率限制。
- 记录审计日志。

### Phase 4：公共探针池

只有在项目稳定后再考虑。

前置条件：

- 明确探针节点授权。
- 有滥用防护。
- 有成本控制。
- 有可撤销的节点注册机制。
- 有公开的隐私和使用说明。

## 数据模型

### ReturnRouteReport

```json
{
  "enabled": true,
  "is_real_return_route": true,
  "target_ip": "203.0.113.10",
  "mode": "manual_import",
  "probe_count": 3,
  "success_count": 2,
  "summary": "真实回程探针 2/3 成功，电信方向需关注。",
  "probes": [],
  "recommendations": []
}
```

字段：

- `enabled`: 是否启用真实回程模块
- `is_real_return_route`: 必须为 true 才能标记为真实回程
- `target_ip`: 被测端公网 IP
- `mode`: `manual_import`、`callback`、`probe_server`
- `probe_count`: 探针数量
- `success_count`: 成功数量
- `summary`: 人类可读结论
- `probes`: 探针结果列表
- `recommendations`: 建议

### ReturnRouteProbeResult

```json
{
  "probe_name": "cn-telecom-shanghai",
  "probe_region": "CN-SH",
  "probe_isp": "China Telecom",
  "source_ip": "198.51.100.20",
  "target_ip": "203.0.113.10",
  "success": true,
  "total_hops": 14,
  "timeout_hops": 2,
  "average_latency_ms": 42.6,
  "last_visible_hop": "203.0.113.10",
  "quality": {
    "grade": "B",
    "status": "success",
    "summary": "电信上海到被测端 14 跳，平均 42.60 ms，超时跳 2。"
  },
  "hops": []
}
```

建议复用当前 `TraceResult` 的结构和质量字段，额外增加探针元数据。

## 报告展示

控制台：

```text
真实回程
探针              地区       运营商          评级      跳数   超时   平均延迟   末跳
cn-telecom-sh     CN-SH      China Telecom  B/良好    14     2      42.60 ms  203.0.113.10
cn-unicom-bj      CN-BJ      China Unicom   C/需关注  18     5      68.10 ms  203.0.113.10
```

Web 报告：

- 模块名称：真实回程
- 指标卡：
  - 探针数量
  - 成功探针
  - 平均跳数
  - 平均延迟
  - 超时跳
- 表格：
  - 探针
  - 地区
  - 运营商
  - 评级
  - 状态
  - 跳数
  - 超时跳
  - 平均延迟
  - 最后可见跳

如果没有真实回程结果：

```text
真实回程未执行。当前路由追踪仅为本机出站路径；如需真实回程，请在远端探针运行 probe-route 并导入结果。
```

## CLI 设计

第一阶段：

```bash
perfassess probe-route --target 203.0.113.10 --probe-name cn-telecom-shanghai -o probe.json
perfassess --return-route-file probe.json --route-trace --network-profile standard
```

第二阶段：

```bash
perfassess --return-route-listen --return-route-port 8090 --return-route-wait 10m
perfassess probe-route --target 203.0.113.10 --post http://203.0.113.10:8090/api/return-route --token TOKEN
```

第三阶段：

```bash
perfassess --return-route-probe https://probe.example.com --return-route-token TOKEN
```

## 验证要求

- `probe-route` 输出 JSON schema 测试
- traceroute/tracert 解析复用当前路由解析测试
- 导入多个 probe JSON 的合并测试
- 失败探针不影响主报告生成
- Web 报告在无探针、有部分探针、全部成功三种状态下都有快照测试
- 自动脚本在未提供探针时只提示，不失败

## 风险与边界

- 真实回程依赖远端节点位置和网络策略，同一地区不同探针可能结果不同。
- traceroute 中间跳超时不一定表示丢包或链路异常。
- 如果被测服务器防火墙拒绝探测包，真实回程可能失败。
- 公共探针池可能被滥用，必须等安全机制成熟后再做。

## 推荐实施顺序

1. 实现 Phase 1 手动探针导入。
2. 报告新增真实回程模块，并保持未执行时的清晰提示。
3. 自动脚本支持 `PERFASSESS_RETURN_ROUTE_FILES`。
4. 实现 Phase 2 一次性回传模式。
5. 评估是否需要 Phase 3 自建探针服务。
