# Changelog

## v0.1.0-nju.1 — 2026-10-06

- 增加锁定的南大 aTrust TCP-only 预设和动态认证域发现。
- 跳过 TCP-only 模式下的虚拟 IP 分配与用户态 L3 初始化。
- 登录后导出账号获授权的 IPv4 TCP 地址范围及 Clash / CFW 分流片段。
- 增加通用 CFW 文件读取 Mixin，保留原订阅规则与网络设置。
- 增加隐藏输入密码的 Windows 启动脚本、可选 SSH 检查及会话保存。
- 增加 Windows 构建、源码/运行包打包脚本和最小运行文件说明。
- 增加相关 Go / JavaScript 测试，适配两处 Windows 文件权限测试。
