# 安全策略

## 支持的版本

| 版本   | 支持状态    |
| ------ | ----------- |
| v0.2.x | ✅ 积极支持 |
| v0.1.x | ❌ 不再支持 |

## 最低 Go 版本

本项目要求 **Go 1.23+**，以确保使用最新的安全特性和标准库修复。

## 报告漏洞

如果您发现了安全漏洞，请**不要**通过公开 Issue 报告。

### 报告方式

1. **GitHub Security Advisories** (推荐)
   - 访问: https://github.com/darkit/slog/security/advisories
   - 点击 "New draft security advisory"

2. **邮件联系**
   - 发送邮件至: security@zishuo.net
   - 主题格式: `[SECURITY] slog - 简要描述`

### 报告内容

请包含以下信息：

- 漏洞描述
- 复现步骤
- 影响版本
- 可能的修复建议（如有）
- 您的联系方式（可选）

## 响应流程

| 阶段 | 时间       | 说明           |
| ---- | ---------- | -------------- |
| 确认 | 24 小时内  | 确认收到报告   |
| 评估 | 72 小时内  | 完成初步评估   |
| 修复 | 视严重程度 | 制定修复计划   |
| 发布 | 修复后     | 发布安全更新   |
| 披露 | 协商后     | 协调披露时间线 |

## 安全最佳实践

### 使用建议

```go
// 生产环境推荐配置
cfg := slog.DefaultConfig()
cfg.SetEnableJSON(true)  // 便于日志收集分析

// 启用 DLP 保护敏感数据
slog.EnableDLPLogger()

// 设置合适的日志级别
slog.SetLevelInfo()  // 或更高级别
```

### 避免记录敏感数据

```go
// ❌ 不要直接记录敏感信息
logger.Info("user login", "password", password)

// ✅ 使用 DLP 或避免记录
logger.Info("user login", "user_id", userID)
```

### 文件日志安全

```go
// 设置合理的文件权限
writer := slog.NewWriter("logs/app.log").
    SetMaxAge(30).       // 定期清理
    SetMaxBackups(10)

// 确保日志目录权限正确
// chmod 750 logs/
```

## 安全扫描

项目使用以下工具进行安全扫描：

| 工具          | 说明         | 命令                |
| ------------- | ------------ | ------------------- |
| `govulncheck` | Go 漏洞检查  | `govulncheck ./...` |
| `gosec`       | Go 安全扫描  | `gosec ./...`       |
| Dependabot    | 依赖漏洞监控 | 自动运行            |

### CI/CD 集成

```yaml
# .github/workflows/security.yml
- name: Run govulncheck
  uses: golang/govulncheck-action@v1
```

## 安全配置检查清单

- [ ] 生产环境启用 DLP
- [ ] 日志级别设置为 Info 或更高
- [ ] 避免在日志中记录密码、Token 等
- [ ] 定期更新依赖版本
- [ ] 使用 `govulncheck` 扫描漏洞
- [ ] 日志文件设置适当权限

## 历史漏洞

| CVE | 版本 | 描述 | 修复版本 |
| --- | ---- | ---- | -------- |
| 无  | -    | -    | -        |

## 相关资源

- [Go 安全最佳实践](https://go.dev/security)
- [OWASP 日志备忘单](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [Go 漏洞数据库](https://pkg.go.dev/vuln/)

---

**注意**: 此安全策略可能会更新，请定期查看最新版本。
