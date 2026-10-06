# NJU-Connect

[下载 Windows 运行包](https://github.com/tukwhy/NJU-Connect/releases/latest) · [查看 Releases](https://github.com/tukwhy/NJU-Connect/releases) · [提交问题](https://github.com/tukwhy/NJU-Connect/issues)

基于 [Mythologyli/zju-connect](https://github.com/Mythologyli/zju-connect) 的南京大学 aTrust 适配版。通过独立本机 SOCKS5 / HTTP 代理访问账号获授权的校内 IPv4 TCP 资源，并与现有 Clash 分流配合。

默认不安装官方 aTrust 客户端，不创建系统 TUN，不修改系统代理、DNS 或路由。程序仍通过学校 aTrust 网关传输，登录和资源访问遵循服务端认证与授权。

## 相比上游的修改

- 新增南大预设：`vpn.nju.edu.cn:443`，动态发现 LDAP / OAuth 登录域。
- 修正 aTrust TCP-only 启动：跳过虚拟 IP 申请与用户态 L3 隧道初始化。
- 默认 SOCKS5 `127.0.0.1:11080`、HTTP `127.0.0.1:11081`，可同时连接多个目标 IP 和 TCP 端口。
- 南大发布二进制锁定预设，拒绝 TUN、路由修改、DNS 劫持、Fake IP 及非 loopback 监听。
- 每次登录后将学校下发的 TCP IPv4 地址范围合并为 CIDR，生成 Clash YAML 与 CFW JavaScript Mixin。
- 提供隐藏密码输入、端口占用检查、可选会话保存与可选 SSH banner 检查。

aTrust 协议、LDAP/OAuth、二次认证、代理服务、资源权限检查和端口转发基于上游实现。上游说明见 [README.upstream.md](README.upstream.md) 与 [README_en.md](README_en.md)。

## 最少需要哪些文件

| 使用方式 | 必需文件 | 说明 |
|---|---|---|
| 直接运行 EXE | `nju-connect.exe` | 可独立运行；从当前目录生成规则文件 |
| 使用启动脚本 | `Start-NJU.ps1`、`nju.toml`、`dist/nju-connect.exe` | 推荐，保留这个目录结构 |
| 接入现有 CFW | 上述运行文件；把 `CFW-Mixin-NJU.js` 的代码粘贴一次到 CFW | JS 文件不是 EXE 的运行依赖；粘贴后可不保留它 |

不需要 Go、Python、Node.js、Docker、aTrust SDK 或额外 `wintun.dll` 来运行发布包。`LICENSE` 和 `NOTICE.md` 随分发包保留。

以下文件由程序生成，初次运行前无需准备，也不要提交到公开仓库：

| 文件 | 用途 |
|---|---|
| `clash-nju.generated.yaml` | 当前账号的实际学校网段与代理节点 |
| `clash-nju.generated.cfw-mixin.js` | 内嵌本次网段的 CFW Mixin |
| `nju-client-data.json` | 仅使用 `-RememberSession` 时保存设备标识和会话 Cookie |

## Windows 快速开始

从本仓库 [Releases](https://github.com/tukwhy/NJU-Connect/releases/latest) 下载 `windows-amd64.zip`。发布包目标为 Windows x64。完整解压后，在解压目录打开普通 PowerShell：

```powershell
.\Start-NJU.ps1
```

输入学校账号与密码，随后按提示完成短信、邮箱或其他二次认证。密码隐藏输入，经当前进程环境传给子进程，不写入配置或命令行；脚本结束后恢复原变量。

保持这个窗口运行。看到 `Exported ... school TCP IPv4 ranges` 后，按下一节加载 Clash 分流。停止程序使用 Ctrl+C，只关闭本程序的连接与监听。

也可以预填学号：

```powershell
.\Start-NJU.ps1 -Username '你的学号'
```

如果 PowerShell 的脚本执行策略阻止运行，可由你手动用一次性子进程执行，不修改系统执行策略：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\Start-NJU.ps1
```

## 与现有 Clash / CFW 配合

访问路径为：应用 → 已有 Clash TUN → 学校网段规则 → `NJU-VPN` → 本机 SOCKS5 → 学校网关 → 校内目标。

要在 SSH、浏览器、远程桌面中直接输入校内 IP，现有 Clash TUN 必须已接管这些流量；只有系统 HTTP 代理并不能自动接管所有应用。程序不会替你开启、关闭或修改 TUN。

### CFW 0.20.39：读取生成文件的 JavaScript Mixin

1. 完成学校登录，确认解压目录中生成了 `clash-nju.generated.yaml`。
2. 打开 `CFW-Mixin-NJU.js`，把 `const file` 改成实际生成文件的绝对路径，例如 `C:/Tools/NJU-Connect/clash-nju.generated.yaml`。
3. 在 CFW 的 Mixin 编辑器中使用 JavaScript，粘贴脚本并保存，手动重新载入当前配置。
4. 如果已有 Mixin，先保留原逻辑再合并；不要覆盖其中已有的 TUN、DNS 或其他配置。`myRules` 数组默认为空，可填自己的规则。

脚本只增补 `NJU-VPN` 节点、前置学校规则并移除旧的学校 IP 规则，保留原订阅节点、其他规则及所有其他配置字段。重复加载不会重复追加。

**规则文件更新不等于内核自动加载。每次学校网段文件更新后，需手动重新载入 CFW 当前配置。** 缺少文件或文件尚无网段时会通知，保留原配置。

运行文件可以放到其他目录，但需同步修改 Mixin 的 `const file`。使用脚本时规则文件写入脚本所在目录；直接运行 EXE 时默认写入当前工作目录，或用 `--clash-rules-file` 指定。

### 通用 YAML 合并

登录后将 `clash-nju.generated.yaml` 中的代理条目和规则加入现有配置的对应列表。不要把整份订阅替换成片段。学校规则放在原有局域网 DIRECT、私网规则与 MATCH 前面，之后手动重新加载。

生成文件包含账号下发的所有 TCP IPv4 资源，可能同时包含校内地址和学校授权的图书馆数据库地址。端口与协议限制仍由 VPN 客户端依据原始资源清单检查。

## 常用参数

```powershell
# 只查询公开认证方式，不登录
.\Start-NJU.ps1 -AuthInfo

# 修改本程序独立监听端口；Clash 节点需使用同一 SOCKS 端口
.\Start-NJU.ps1 -SocksPort 12080 -HttpPort 12081

# 可选保存会话；默认不保存，也不自动绑定授信终端
.\Start-NJU.ps1 -RememberSession

# 指定本程序出口网卡，仅影响自己的连接，不改网卡或路由
.\Start-NJU.ps1 -BindInterface 'Wi-Fi'

# 可选 SSH banner 检查，不提交 SSH 用户凭据
.\Start-NJU.ps1 -CheckTarget '10.0.0.42:22'

# 可选单机转发；通用 SOCKS5 仍可同时访问其他地址
.\Start-NJU.ps1 -Target '10.0.0.42:22' -SshPort 2222
```

直接运行 EXE 也支持原有参数及环境变量：

```powershell
.\dist\nju-connect.exe --username '你的学号' --password '你的密码'
.\dist\nju-connect.exe --help
```

命令行密码可能出现在进程参数或命令历史中，日常建议用隐藏输入的启动脚本；也支持 `ZJU_CONNECT_PASSWORD` 环境变量。

新增选项包括 `--profile nju`、`--clash-rules-file <路径>` 和 `--check-target <IP:端口>`。发布 EXE 已锁定 `nju` 预设，无需手动设置。

### 浏览器 OAuth

```powershell
.\Start-NJU.ps1 -LoginMode Browser
```

沿用上游交互式 OAuth：终端给出学校登录 URL，用户完成浏览器登录，并从 Network 中获取 `/passport/v1/auth/httpsOauth2?code=...` 回调 URL 粘贴到终端。回调是敏感凭据，不要公开。本发布版已验证 LDAP 登录后的数据通路，南大 OAuth 全流程仍未验证；优先使用 LDAP。

## 限制与排障

- 当前面向 IPv4 TCP；不支持此模式下的 UDP、ICMP 或 IPv6，不能用 `ping` 判断 SSH 是否可达。
- 若 Clash 连接页面仍显示梯子节点，先检查 Mixin 已启用、生成文件路径正确，并重新载入当前配置。
- 若流量根本未进入 Clash，需由用户检查已有 TUN 的排除规则；程序不会修改这些设置。
- 网段与家庭局域网重叠时，需要更具体的本地 DIRECT 规则；完全相同的 IP 不能仅靠 IP 规则区分两个网络。
- 若提示端口占用，修改本程序端口，不会自动停止占用端口的程序。
- 若提示多个登录域，使用 `-AuthInfo` 查看，再用 `-LoginDomain` 明确选择。
- 账号资源清单未包含某个 IP / 端口时会拒绝连接，不会静默回落到本机其他出口。
- CFW/Mixin 的配置保存与重载由用户执行；程序不调用 Clash 控制 API 改配置。

## 从源码构建

`go.mod` 声明 Go 最低版本。本次 Windows 发布包使用 Go 1.27.1、`CGO_ENABLED=0` 构建。

```powershell
.\Build-NJU.ps1 -Test
```

产物为 `dist/nju-connect.exe`。构建脚本将模块与编译缓存放入 `work/`，仅临时设置本进程 Go 编译变量并恢复。支持 Git checkout 和无 `.git` 的源码压缩包。

可指定版本和另一个输出路径，避免覆盖正在运行的 EXE：

```powershell
.\Build-NJU.ps1 -OutputPath work/release-build/nju-connect.exe -Version v0.1.0-nju.1
```

可选 Node.js Mixin 测试（运行发布包不需要 Node.js）：

```powershell
node tests/cfw_mixin.test.cjs
node tests/cfw_mixin_file.test.cjs
```

## 本地打包与 GitHub

打包脚本仅使用 Python 标准库，生成干净源码包、Windows 运行包和 SHA256 校验清单：

```powershell
python scripts/package_release.py --version v0.1.0-nju.1 --binary work/release-build/nju-connect.exe
```

输出到 `releases/`。打包使用源码白名单，不收集 `.git`、编译工具链、缓存、账号会话、生成网段、日志、HAR、抓包或本机 Clash 配置。

本仓库为 [Mythologyli/zju-connect](https://github.com/Mythologyli/zju-connect) 的 Fork。源码通过 Git 保留上游历史，Windows 运行 ZIP、对应源码 ZIP 和 SHA256 校验清单作为本仓库 Release 附件发布。两者保留许可证和上游归属。

开发 checkout 中建议将本仓库配置为 `origin`，上游配置为 `upstream`。构建产物、缓存和运行时敏感文件不进入 Git 历史。

GitHub Actions 仅测试和构建 Windows x64 包，并上传构建产物；不会自动发布 Release、推送镜像或要求学校凭据。原上游工作流保留在 `docs/upstream-workflows/` 供参考，不会自动运行。

## 验证情况

已通过 Go 全量测试及 JavaScript Mixin 测试，覆盖网络模式限制、动态登录域、TCP-only 启动、多网段导出、旧规则更新和现有配置保留。已实测在现有 CFW TUN 下，校内 TCP 连接命中 `NJU-VPN` 并读取到 SSH banner；这不代表已测试所有校内资源，也不涉及 SSH 用户认证。

## 许可证与来源

保留上游 [AGPL-3.0 许可证](LICENSE) 及已有第三方许可证。基线为 `Mythologyli/zju-connect` 提交 `923672d`，来源和修改说明见 [NOTICE.md](NOTICE.md)。本项目不是学校或深信服官方客户端，不包含官方 aTrust 二进制、SDK 或账号凭据。
