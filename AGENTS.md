# AGENTS.md

本文件用于约定本仓库中 AI 编码助手和协作者的工作规则。所有自动化修改、提交和推送都应遵守这些要求。

## 项目原则

- 项目名称统一使用 `perfassess`，展示名称统一使用 `Perfassess`。
- Go 模块路径统一使用 `github.com/E8A281E6ACA2/perfassess`。
- 优先保持现有代码结构和风格，不做无关重构。
- 修改应尽量小而明确，一次提交只解决一个清晰问题。
- 不要删除或覆盖用户已有改动。提交前必须先检查工作区状态。

## 可以做

- 修复明确的问题、补充必要测试、更新相关文档。
- 新增脚本时必须考虑可重复执行、错误提示、退出码和清理方式。
- 修改 CLI、脚本或 README 时，保持新服务器首次使用流程可直接复制执行。
- 涉及 Web 报告时，保持 Material Design 3 / Google 风格体验一致。
- 对耗时流程提供可见进度，同时保留日志和报告文件。

## 不该做

- 不要在用户未要求时做大范围重构。
- 不要静默安装系统依赖，除非用户显式运行 bootstrap 或安装脚本。
- 不要把密钥、token、服务器私密信息、测试产物或本地日志提交到仓库。
- 不要使用破坏性 Git 命令，例如 `git reset --hard`、`git checkout -- <file>` 覆盖用户改动。
- 不要在测试失败时推送到远程。

## 提交格式

提交信息必须使用作用域形式，并用中文描述：

```text
type(scope): 中文描述
```

常用类型：

- `feat`: 新功能
- `fix`: 修复问题
- `docs`: 文档
- `scripts`: 脚本和自动化
- `test`: 测试
- `refactor`: 重构
- `chore`: 杂项维护

示例：

```text
docs(readme): 更新新服务器拉取与测试说明
scripts(auto): 增加实时测评进度输出
feat(web): 增加测评完成后的报告服务入口
fix(cli): 修复 Web 报告端口提示
```

## 提交前检查

提交前必须完成：

```bash
git status --short --branch
git diff --check
go test ./...
```

如果修改了 shell 脚本，还必须执行：

```bash
bash -n scripts/*.sh
```

如果修改了 README、安装脚本、bootstrap 或自动测评流程，应至少检查相关命令片段是否仍然匹配当前代码。

## 推送规则

- 只有测试通过后才能提交。
- 只有提交成功并确认内容符合预期后才能推送。
- 推送前再次确认当前分支和远程：

```bash
git status --short --branch
git remote -v
```

- 推送后确认远程 `main` 指向最新提交：

```bash
git ls-remote --heads origin main
```

## 测试与验收

常规代码修改至少运行：

```bash
go test ./...
```

脚本、安装、bootstrap、自动测评相关修改，优先额外运行：

```bash
scripts/perfassess-auto.sh
```

如果改动可能影响真实服务器使用流程，应在 README 中同步更新使用说明。

## 文档要求

- README 中的新服务器命令必须适合直接复制执行。
- 默认自动测评应说明是否会安装依赖、是否会启动 Web 服务、输出文件在哪里。
- 涉及端口访问时，应说明公网访问和 SSH 端口转发两种方式。
- 文档中不要保留旧仓库名或旧模块名。

## 远程与发布

- 远程仓库地址应为 `git@github.com:E8A281E6ACA2/perfassess.git` 或 `https://github.com/E8A281E6ACA2/perfassess.git`。
- 不要把本地临时报告、构建产物、日志文件或覆盖率报告推送到远程。
- 发布相关改动应参考 `docs/release-checklist.md`。
