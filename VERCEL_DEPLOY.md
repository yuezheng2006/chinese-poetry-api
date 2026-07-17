# 中国古诗词 API - Vercel 部署指南

本指南说明如何将中国古诗词 API 部署到 Vercel。

## 前提条件

- [Vercel 账号](https://vercel.com)
- [Vercel CLI](https://vercel.com/docs/cli)（可选，用于本地测试）

## 部署步骤

### 方法一：通过 Vercel CLI 部署（推荐用于测试）

1. **安装 Vercel CLI**
   ```bash
   npm i -g vercel
   ```

2. **登录 Vercel**
   ```bash
   vercel login
   ```

3. **部署**
   ```bash
   vercel deploy
   ```

4. **部署到生产环境**
   ```bash
   vercel --prod
   ```

### 方法二：通过 Git 集成自动部署（推荐用于生产）

1. **连接 GitHub 仓库**
   - 访问 [Vercel Dashboard](https://vercel.com/dashboard)
   - 点击 "New Project"
   - 导入你的 GitHub 仓库：`yuezheng2006/chinese-poetry-api`

2. **Vercel 自动检测配置**
   - Vercel 会自动检测到 `Dockerfile.vercel`
   - 使用默认设置即可

3. **部署**
   - 点击 "Deploy"
   - Vercel 会自动构建 Docker 镜像并部署

## 配置说明

### Dockerfile.vercel

项目已包含 `Dockerfile.vercel` 文件，关键配置：

- **端口配置**：监听 `$PORT` 环境变量（Vercel 默认 80）
- **多阶段构建**：优化镜像大小
- **启动脚本**：自动下载诗词数据库

### 环境变量（可选）

在 Vercel 项目设置中可以配置以下环境变量：

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `PORT` | 服务端口 | 80（Vercel 自动设置） |
| `GIN_MODE` | Gin 模式 | `release` |
| `RATE_LIMIT_ENABLED` | 是否启用限流 | `true` |
| `RATE_LIMIT_RPS` | 每秒请求数 | `10` |
| `RATE_LIMIT_BURST` | 突发请求数 | `20` |

## 本地测试

使用 Vercel CLI 在本地测试 Docker 容器：

```bash
# 需要本地运行 Docker
vercel dev
```

或直接使用 Docker：

```bash
# 构建镜像
docker build -f Dockerfile.vercel -t chinese-poetry-api:vercel .

# 运行容器（测试 PORT 环境变量）
docker run -p 3000:80 -e PORT=80 chinese-poetry-api:vercel
```

访问 http://localhost:3000/api/v1/health 验证服务。

## 关键差异

### Dockerfile 对比

| 配置项 | 原 Dockerfile | Dockerfile.vercel |
|--------|---------------|-------------------|
| 默认端口 | 1279 | 80 |
| PORT 环境变量 | 固定 1279 | 读取 $PORT |
| EXPOSE | 1279 | 80 |

### 代码兼容性

✅ **无需修改代码**！项目的 `internal/config/config.go` 已经支持从 `PORT` 环境变量读取端口：

```go
// Line 114-118
if port := os.Getenv("PORT"); port != "" {
    if p, err := strconv.Atoi(port); err == nil {
        v.Set("server.port", p)
    }
}
```

## Vercel 限制

注意以下 Vercel Functions 限制：

- **执行时间**：根据套餐不同（Hobby: 10s, Pro: 60s, Enterprise: 900s）
- **内存**：最大 3GB（Enterprise）
- **容器要求**：
  - 必须监听 `$PORT` 环境变量
  - 必须运行 HTTP 服务器
  - 无状态（数据库文件在容器内，每次重启会重新下载）

## 数据持久化

⚠️ **重要**：Vercel 容器是无状态的。本项目的 `startup.sh` 脚本会在每次启动时：

1. 检查本地是否有数据库文件
2. 如果没有，从 GitHub Releases 下载
3. 如果有，检查是否有更新版本

首次启动会下载约 100MB 的数据库文件，可能需要几秒钟。后续启动会使用缓存。

如需持久化存储，可以考虑：
- 使用 [Vercel Postgres](https://vercel.com/docs/storage/vercel-postgres)
- 使用外部数据库服务

## API 端点

部署成功后，API 地址为：

```bash
# REST API
https://your-project.vercel.app/api/v1/poems

# GraphQL
https://your-project.vercel.app/graphql

# 健康检查
https://your-project.vercel.app/api/v1/health
```

## 故障排查

### 1. 端口错误

**症状**：服务无法访问

**解决**：确保使用 `Dockerfile.vercel`，而非原始 `Dockerfile`

### 2. 启动超时

**症状**：首次部署超时

**原因**：下载数据库文件需要时间

**解决**：
- Vercel 会自动重试
- 或预先将数据库文件构建到镜像中

### 3. 内存不足

**症状**：容器崩溃

**解决**：
- 升级 Vercel 套餐
- 优化数据库连接池设置

## 性能优化

1. **启用 CDN 缓存**：为静态响应添加缓存头
2. **连接池调优**：根据 Vercel 实例规格调整 `DB_MAX_OPEN_CONNS`
3. **限流保护**：使用内置限流防止滥用

## 成本估算

- **Hobby 套餐**：免费，有执行时间和带宽限制
- **Pro 套餐**：$20/月，更长的执行时间和更多带宽
- **Enterprise**：自定义，适合生产环境

详见 [Vercel 定价](https://vercel.com/pricing)。

## 参考资料

- [Vercel Dockerfile 文档](https://vercel.com/docs/functions/container-images)
- [Vercel Functions 限制](https://vercel.com/docs/functions/limitations)
- [项目原始仓库](https://github.com/palemoky/chinese-poetry-api)
