# 来源与归属

NJU-Connect 是基于 https://github.com/Mythologyli/zju-connect 的派生适配版本。

- 上游基线：提交 `923672d`。
- 上游作者与贡献者的归属及致谢保留在 `README.upstream.md`、`README_en.md`、`LICENSE` 和相关源码中。
- 上游基于 EasierConnect；原有相关致谢继续保留。
- 根目录 `LICENSE` 为上游 GNU Affero General Public License v3。`internal/ping/LICENSE` 等已有第三方许可证继续保留。

本适配版新增南大预设、动态登录域发现、TCP-only 初始化修正、学校规则导出、CFW Mixin、Windows 启动与打包脚本、文档和测试。aTrust 协议、认证及代理基础能力沿用上游。

运行包与对应源码包应一同提供。运行包不包含官方 aTrust 客户端、厂商 SDK、学校凭据或实际账号资源清单。
