# NJU-Connect

基于 [zju-connect](https://github.com/Mythologyli/zju-connect) 的南京大学 aTrust 适配版，通过本地 SOCKS5 / HTTP 代理访问校内资源，可配合 Clash 分流使用。

[下载 Windows 版](https://github.com/tukwhy/NJU-Connect/releases/latest)

## 主要改动

- 新增南大预设，自动发现登录域。
- 修正 TCP-only 启动流程，跳过虚拟 IP 申请和用户态 L3 隧道初始化。
- 登录后自动生成学校资源对应的 Clash 规则及 CFW Mixin。
- 提供隐藏密码输入、端口检查、会话保存和 SSH 连通检查脚本。

aTrust 协议与认证实现沿用上游。发布版仅使用本地 TCP 代理，不创建系统 TUN。

## 使用

### 启动

下载并解压 Windows x64 运行包。启动脚本所需的文件结构：

```text
NJU-Connect/
├── Start-NJU.ps1
├── nju.toml
└── dist/
    └── nju-connect.exe
```

在解压目录打开 PowerShell：

```powershell
.\Start-NJU.ps1
```

按提示输入学号、密码和验证码，保持窗口运行。退出使用 Ctrl+C。

默认代理地址：

| 类型 | 地址 |
|---|---|
| SOCKS5 | `127.0.0.1:11080` |
| HTTP | `127.0.0.1:11081` |

也可直接运行 EXE：

```powershell
.\dist\nju-connect.exe --username '学号' --password '密码'
```

直接运行只需要 EXE。推荐使用脚本隐藏输入密码，避免密码出现在命令历史和进程参数中。

### Clash 分流

登录成功后，程序会生成 `clash-nju.generated.yaml`。

**Clash for Windows 0.20.39：**

1. 打开 `CFW-Mixin-NJU.js`，将 `const file` 改为生成文件的绝对路径。
2. 将脚本粘贴到 CFW 的 JavaScript Mixin 编辑器并保存；已有 Mixin 应保留原逻辑后合并。
3. 启用 Mixin，重新载入当前配置。

**其他 Clash 客户端：** 将生成文件中的节点和规则合并到现有配置，学校规则放在局域网 DIRECT、私网规则和 MATCH 之前，再重新载入配置。

要在应用中直接输入校内 IP，已有 Clash TUN 必须接管这些流量；也可在支持代理的应用中直接设置上述 SOCKS5 / HTTP 地址。

**每次学校规则文件更新后，都需要重新载入 Clash 配置。**

### 常用选项

```powershell
# 预填学号，密码仍隐藏输入
.\Start-NJU.ps1 -Username '学号'

# 保存会话
.\Start-NJU.ps1 -RememberSession

# 修改代理端口
.\Start-NJU.ps1 -SocksPort 12080 -HttpPort 12081

# 检查 SSH 连通性
.\Start-NJU.ps1 -CheckTarget '10.0.0.42:22'
```

完整参数：

```powershell
.\dist\nju-connect.exe --help
```

## 从源码构建

安装满足 `go.mod` 要求的 Go，执行：

```powershell
.\Build-NJU.ps1 -Test
```

产物为 `dist/nju-connect.exe`。

## 限制

- 支持账号获授权的 IPv4 TCP 资源；不支持 UDP、ICMP 和 IPv6。
- LDAP 登录已验证，浏览器 OAuth 完整流程尚未验证。

## 许可证

沿用上游 AGPL-3.0 许可证，详见 [LICENSE](LICENSE)。来源与修改说明见 [NOTICE.md](NOTICE.md)。
