# 贡献指南

感谢您对 slog 项目的关注！本文档将指导您如何为项目做出贡献。

## 开发环境

### 系统要求

- **Go 版本**: 1.23 或更高版本
- **操作系统**: Linux, macOS, Windows

### 环境准备

```bash
# 克隆仓库
git clone https://github.com/darkit/slog.git
cd slog

# 下载依赖
go mod download

# 验证环境
go version
go env GOPATH
```

### 必备工具

```bash
# 代码格式化
go install golang.org/x/tools/cmd/goimports@latest

# 静态检查
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 漏洞扫描
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## 代码规范

### 格式化

所有代码必须通过 `gofmt` 和 `goimports` 格式化：

```bash
# 格式化代码
gofmt -w .

# 整理 import
goimports -w .

# 或使用 go fmt
go fmt ./...
```

### 代码检查

提交前运行所有检查：

```bash
# 静态检查
golangci-lint run

# 漏洞扫描
govulncheck ./...

# go vet
go vet ./...
```

### 注释规范

导出符号必须有文档注释：

```go
// NewLogger creates a new Logger instance with the specified writer.
// The noColor parameter disables colored output when true.
// The addSource parameter includes source file information in logs.
func NewLogger(w io.Writer, noColor, addSource bool) *Logger {
    // ...
}
```

### 命名规范

- 使用驼峰命名法 (CamelCase)
- 导出符号以大写字母开头
- 接口名以 `-er` 结尾 (如 `Handler`, `Formatter`)
- 避免使用下划线命名 (snake_case)

## 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行带覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率详情
go tool cover -func=coverage.out
```

### 竞态检测

并发代码必须通过竞态检测：

```bash
go test -race ./...
```

### 基准测试

```bash
# 运行所有基准测试
go test -bench=. -benchmem ./...

# 运行特定基准测试
go test -bench=BenchmarkLogger -benchmem

# 详细输出
go test -bench=. -benchmem -benchtime=3s ./...
```

### 测试要求

- 新增功能必须包含单元测试
- 测试覆盖率应保持在 **80%** 以上
- 边界条件和错误路径必须测试
- 并发代码需进行竞态检测

### Example 测试

Example 测试会出现在 pkg.go.dev 文档中：

```go
// ExampleNewLogger demonstrates basic logger creation.
func ExampleNewLogger() {
    logger := slog.NewLogger(os.Stdout, false, false)
    logger.Info("Hello Slog!")
    // Output: Hello Slog!
}
```

## 提交规范

### Commit 格式

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型 (type)**：

| 类型       | 说明                   |
| ---------- | ---------------------- |
| `feat`     | 新功能                 |
| `fix`      | Bug 修复               |
| `docs`     | 文档更新               |
| `style`    | 代码格式（不影响功能） |
| `refactor` | 重构                   |
| `perf`     | 性能优化               |
| `test`     | 测试相关               |
| `chore`    | 构建/工具变动          |

**示例**：

```
feat(dlp): add bank card desensitization support

Implement bank card number desensitization, keeping first 6 and last 4
digits visible with asterisks in between.

Closes #123
```

### 分支管理

- `main`: 主分支，保持稳定可发布状态
- `feature/*`: 功能分支
- `fix/*`: 修复分支

## 开发流程

### 1. Fork & Clone

```bash
# Fork 后克隆
git clone https://github.com/YOUR_USERNAME/slog.git
cd slog

# 添加上游仓库
git remote add upstream https://github.com/darkit/slog.git
```

### 2. 创建分支

```bash
git checkout -b feature/your-feature-name
```

### 3. 开发与测试

```bash
# 编写代码
# 添加测试

# 运行测试
go test ./...
go test -race ./...

# 运行检查
golangci-lint run
govulncheck ./...
```

### 4. 提交更改

```bash
git add .
git commit -m "feat: add new feature"
```

### 5. 推送并创建 PR

```bash
git push origin feature/your-feature-name
```

在 GitHub 上创建 Pull Request。

## PR 要求

- [ ] 描述清楚改动内容和原因
- [ ] 关联相关 Issue（如有）
- [ ] 所有测试通过 (`go test ./...`)
- [ ] 竞态检测通过 (`go test -race ./...`)
- [ ] 静态检查通过 (`golangci-lint run`)
- [ ] 测试覆盖率不降低
- [ ] 更新相关文档

## 文档更新

更改涉及以下情况时，请同步更新文档：

- 新增 API 或改变 API 行为 → 更新 `doc.go` 和 `README.md`
- 修复 Bug → 更新 `CHANGELOG.md`
- 添加新功能 → 更新 `README.md` 和添加 Example
- 更新依赖 → 更新 `go.mod`

## 问题反馈

### 报告 Bug

请包含以下信息：

````markdown
**环境**:

- Go 版本: `go version` 输出
- 操作系统:
- slog 版本:

**复现步骤**:

1. ...
2. ...

**预期行为**:

**实际行为**:

**代码示例**:

```go
// 可复现的代码
```
````

````

### 功能建议

请描述：

- 使用场景
- 期望的 API 设计
- 可能的实现方案

## 发布流程

维护者发布新版本：

```bash
# 1. 更新 CHANGELOG.md
# 2. 创建标签
git tag -a v1.x.x -m "Release v1.x.x"

# 3. 推送标签
git push origin v1.x.x
````

## 联系方式

- Issues: https://github.com/darkit/slog/issues
- Discussions: https://github.com/darkit/slog/discussions

## 行为准则

- 尊重他人
- 欢迎新手
- 建设性反馈
- 关注技术本身

感谢您的贡献！
