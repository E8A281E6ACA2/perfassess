# GitHub Release 发布手册

本文档用于执行一次可复现的 GitHub Release。发布前必须先完成代码提交、推送和真实 VPS 验收；不要从脏工作区直接打 tag。

## 前置条件

- 当前分支为 `main`，并且已经推送到远程。
- 工作区干净：`git status --short --branch` 不应显示未提交改动。
- 本地已通过 `make release-check`。
- 至少完成一次真实 VPS 验收，并保留脱敏后的验收摘要。
- 如果真实 VPS 结果用于评分校准，只能从远程机器取回 `/tmp/perfassess-auto/calibration_sample.json`，并用 `PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1 scripts/calibration-collect.sh` 归档到样本池；不要把原始报告、日志、压缩包或完整报告目录作为校准输入。
- 如果本次改动影响评分阈值或 `score_calibration.version`，必须先按 `docs/calibration-dataset-policy.md` 完成样本准入检查；默认要求运行 `python3 scripts/calibration-summary.py /path/to/samples --require-manifest --require-score-profiles vps,server,workstation --require-policy formal`，除非发布记录明确说明本次只覆盖某个单一 `score_profile`。

## 发布前检查

```bash
git status --short --branch
git pull --ff-only origin main
make release-check
```

确认本地产物包含跨平台二进制和 SHA256 清单：

```bash
ls -lh build/perfassess_linux_amd64 \
       build/perfassess_linux_arm64 \
       build/perfassess_darwin_amd64 \
       build/perfassess_darwin_arm64 \
       build/perfassess.exe \
       build/checksums.txt

cd build
sha256sum -c checksums.txt
cd ..
```

## 创建 Tag

版本号使用 `vMAJOR.MINOR.PATCH`，例如：

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

推送 tag 后，GitHub Actions 会触发 `.github/workflows/release.yml`：

- 执行 `make validate`
- 构建 Linux amd64/arm64、macOS amd64/arm64、Windows amd64
- 生成 `checksums.txt`
- 执行发布二进制冒烟
- 上传 Release 资产

## 手动触发

如果需要手动触发 workflow，可以在 GitHub Actions 页面运行 `Release` workflow，并填写 `version` 输入，例如 `v1.0.0`。workflow 会拒绝不符合 `vMAJOR.MINOR.PATCH` 形式的版本号。手动触发时应确认当前 `main` 已经包含目标提交；正式公开发布仍推荐使用 tag 触发。

## 发布后验证

Release 页面必须包含：

- `perfassess_linux_amd64`
- `perfassess_linux_arm64`
- `perfassess_darwin_amd64`
- `perfassess_darwin_arm64`
- `perfassess.exe`
- `checksums.txt`

下载并校验全部 Release 资产：

```bash
scripts/verify-release-assets.sh --version latest
```

该命令默认会下载全部资产、校验 `checksums.txt`，并在当前平台有可执行发布二进制时跑一次 basic 自动测评，随后校验报告目录和 `perfassess-report.zip`。校验 SHA256 时会优先使用 `sha256sum`，没有时使用 macOS 常见的 `shasum -a 256`。如果当前平台不支持执行 Release 资产，会清晰提示并只完成下载与 SHA256 校验；如果只想校验下载资产和 SHA256，可以加 `--skip-smoke`。

指定 tag：

```bash
scripts/verify-release-assets.sh --version v1.0.0
```

验证 bootstrap 可以使用预构建二进制：

```bash
curl -fsSL https://raw.githubusercontent.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap.sh \
  | PERFASSESS_BOOTSTRAP_BINARY=1 PERFASSESS_AUTO_ACCEPTANCE=0 bash
```

512MB 或低内存机器上，默认 `PERFASSESS_BOOTSTRAP_BINARY=auto` 会优先尝试下载 Release 二进制，并在 `checksums.txt` 可用时校验 SHA256；失败时才回退源码构建。

## 发布失败处理

- 如果 `make validate` 失败，不要发布；先修复测试或契约。
- 如果 `checksums.txt` 缺失或校验失败，不要发布；重新运行 release workflow。
- 如果 Release 资产缺失，删除该 Release 和 tag，修复 workflow 后重新发版。
- 如果 bootstrap 无法下载预构建二进制，先确认 Release 资产名和 `checksums.txt` 是否一致，再检查服务器网络和 GitHub 访问。

删除错误 tag 的命令：

```bash
git push origin :refs/tags/v1.0.0
git tag -d v1.0.0
```

只在确认错误 Release 已经删除或不可见后再重新创建同名 tag。
