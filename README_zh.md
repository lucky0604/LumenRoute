# LumenRoute

面向公司内网已部署 vLLM / SGLang 服务的轻量模型控制台，提供 OpenAI 兼容统一代理、React 管理界面、模型路由配置、API Key 管理、请求日志和 Prometheus 指标——不接管模型部署或 GPU 调度。

## 技术栈

| 层级 | 技术 |
|---|---|
| 后端 | Go, SQLite, WAL 模式 |
| 前端 | React, Vite, TypeScript, Ant Design |
| 代理 | OpenAI 兼容 `/v1/models`、`/v1/chat/completions` |
| 指标 | Prometheus `/metrics` |
| 部署 | Docker Compose, 单二进制 + systemd |

## 快速开始

### 环境要求

- Go 1.23+
- Node.js 18+
- （可选）Docker

### 后端启动

```bash
# 运行所有测试
go test ./...

# 编译并运行
go build -o lumenroute ./cmd/server
./lumenroute
```

服务默认监听 `:8080`。首次启动时自动创建管理员账号：

- 如果设置了 `LUMENROUTE_ADMIN_PASSWORD`，使用该密码。
- 否则生成随机密码并写入 `data/bootstrap-admin-password` 文件。

### 前端启动

```bash
cd web
npm install
npm run dev      # 开发服务器（支持热更新）
npm run build    # 生产构建
```

## 环境变量配置

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LUMENROUTE_SERVER_PORT` | `8080` | HTTP 监听端口 |
| `LUMENROUTE_DB_DSN` | `file:data/lumenroute.db?...` | SQLite 连接串 |
| `LUMENROUTE_ADMIN_USER` | `admin` | 管理员用户名 |
| `LUMENROUTE_ADMIN_PASSWORD` | （自动生成） | 管理员密码 |
| `LUMENROUTE_PROXY_AUTH_MODE` | `required` | 代理鉴权模式：`required` / `optional` / `disabled` |
| `LUMENROUTE_SESSION_SECRET` | （自动生成） | 会话签名密钥 |
| `LUMENROUTE_API_KEY_PREFIX` | `llmcp_` | 代理 API Key 前缀 |
| `LUMENROUTE_METRICS_PATH` | `/metrics` | Prometheus 指标路径 |
| `LUMENROUTE_HEALTH_CHECK_INTERVAL_SECONDS` | `30` | Provider 健康检查间隔 |
| `LUMENROUTE_REQUEST_LOG_RETENTION_DAYS` | `7` | 日志保留天数 |

## Docker 部署

```bash
docker compose -f docker/docker-compose.yml up -d
```

内网部署默认使用 `network_mode: host`。

## API 参考

### Admin API（需要 Session Cookie 登录态）

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/auth/login` | 管理员登录 |
| `POST` | `/api/auth/logout` | 管理员登出 |
| `GET` | `/api/providers` | 获取 Provider 列表 |
| `POST` | `/api/providers` | 新增 Provider |
| `GET` | `/api/providers/:id` | 查看 Provider |
| `PUT` | `/api/providers/:id` | 更新 Provider |
| `DELETE` | `/api/providers/:id` | 删除 Provider |
| `POST` | `/api/providers/:id/check` | 手动健康检查 |
| `GET` | `/api/routes` | 获取 Route 列表 |
| `POST` | `/api/routes` | 新增 Route |
| `GET` | `/api/routes/:id` | 查看 Route |
| `PUT` | `/api/routes/:id` | 更新 Route |
| `DELETE` | `/api/routes/:id` | 删除 Route |
| `GET` | `/api/routes/:id/targets` | 获取 Route Target 列表 |
| `POST` | `/api/routes/:id/targets` | 新增 Target |
| `PUT` | `/api/route-targets/:id` | 更新 Target |
| `DELETE` | `/api/route-targets/:id` | 删除 Target |
| `POST` | `/api/route-targets/:id/test` | 测试 Target |
| `GET` | `/api/api-keys` | 获取 API Key 列表 |
| `POST` | `/api/api-keys` | 创建 API Key（仅创建时返回完整密钥） |
| `DELETE` | `/api/api-keys/:id` | 删除 API Key |
| `POST` | `/api/api-keys/:id/disable` | 禁用 API Key |
| `POST` | `/api/api-keys/:id/enable` | 启用 API Key |
| `GET` | `/api/request-logs` | 请求日志列表 |
| `GET` | `/api/request-logs/:id` | 请求日志详情 |

### 代理 API（OpenAI 兼容）

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/v1/models` | 获取可用模型列表 |
| `POST` | `/v1/chat/completions` | 对话补全（支持流式与非流式） |

代理鉴权由 `LUMENROUTE_PROXY_AUTH_MODE` 控制：

- `required`：必须携带 `Authorization: Bearer <api_key>`
- `optional`：携带 key 时校验权限；不携带则放行（request_log 中 api_key_id 为空）
- `disabled`：不校验 LumenRoute API Key

## 项目结构

```
cmd/server/         服务入口
internal/
  api/              HTTP 处理层
  apikey/           API Key 管理（SHA-256 哈希存储，仅创建时返回明文）
  auth/             管理员引导、密码哈希、会话管理
  config/           环境变量配置
  db/               SQLite 连接、版本化迁移、索引
  logs/             请求日志写入、查询、过滤
  metrics/          Prometheus 指标注册
  models/           领域模型
  provider/         Provider CRUD、健康状态
  proxy/            OpenAI 代理（模型列表、对话补全、SSE 流式转发）
  route/            Route/Target CRUD、加权路由选择
  scheduler/        Provider 健康检查、日志清理
web/
  src/pages/        登录、Providers、Routes、API Keys、Request Logs、Health
  src/components/   AdminLayout
docker/             Dockerfile、docker-compose.yml
tests/              协议测试
```

## 核心概念

| 概念 | 说明 |
|---|---|
| **Provider** | 已有的 OpenAI 兼容后端服务（vLLM / SGLang） |
| **Route** | 对客户端暴露的公开模型名 |
| **Route Target** | Route 下的具体后端目标（指定 provider、upstream model、权重） |
| **API Key** | `/v1/*` 代理接口的访问凭证（SHA-256 哈希存储） |
| **Request Log** | 每次代理请求的元数据记录（不存储 prompt 和 response 内容） |

## 开发命令

```bash
# 后端
go test ./...              # 运行所有测试
go build -o lumenroute ./cmd/server  # 编译
go vet ./...               # 静态分析

# 前端
cd web
npm run dev                # 开发服务器（http://localhost:5173）
npm run build              # 生产构建
npm run preview            # 预览生产构建
```

## 许可证

内部使用。
