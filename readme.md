# FIFU Gateway

一个基于 WebAuthn 的身份认证网关，使用 Go 语言开发，支持通行密钥（Passkey）登录，并提供完整的认证和授权机制。

## 功能特性

- **WebAuthn 认证**: 支持使用通行密钥（Passkey）进行安全的无密码登录
- **PASETO Token**: 使用 PASETO V2 标准进行令牌管理（支持对称/非对称加密）
- **角色权限系统**: 基于 Token 的权限控制，支持多角色管理
- **API 网关**: 反向代理业务服务，自动注入用户身份信息
- **嵌入式前端**: 将前端静态文件嵌入二进制程序中，单文件部署
- **SQLite 数据库**: 使用轻量级数据库存储用户信息

## 技术栈

### 后端
- **Go 1.25.4**: 主要开发语言
- **Gin**: Web 框架
- **GORM**: ORM 框架
- **SQLite**: 数据库
- **go-webauthn**: WebAuthn 协议实现
- **PASETO**: 安全令牌标准

### 前端
- **原生 JavaScript**: 无框架依赖
- **WebAuthn API**: 浏览器原生 WebAuthn 支持

## 项目结构

```
fifu-gateway/
├── main.go              # 程序入口
├── database/            # 数据库初始化
│   └── database.go
├── handlers/            # HTTP 请求处理器
│   ├── auth_handler.go  # WebAuthn 注册/登录处理
│   └── user_handler.go  # 用户相关处理
├── middleware/          # 中间件
│   └── paseto_auth.go   # 认证和授权中间件
├── models/              # 数据模型
│   └── user_model.go    # 用户模型（实现 WebAuthn 接口）
├── router/              # 路由配置
│   ├── router.go        # 路由定义
│   └── public/          # 前端静态文件（嵌入）
│       ├── index.html
│       ├── main.js
│       └── utils/
├── utils/               # 工具函数
│   └── paseto.go        # PASETO 令牌生成和验证
└── webauthn/            # WebAuthn 封装
    └── webauthn.go      # WebAuthn 协议实现
```

## 快速开始

### 环境要求

- Go 1.25.4 或更高版本
- 现代浏览器（支持 WebAuthn API）

### 安装与运行

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd fifu-gateway
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **运行服务**
   ```bash
   go run main.go
   ```

4. **访问应用**
   打开浏览器访问：http://localhost:5000

### 编译部署

```bash
# 编译为可执行文件
go build -o fifu-gateway.exe

# 运行
./fifu-gateway.exe
```

## API 接口

### WebAuthn 认证

#### 1. 开始注册
```
POST /webauthn/register/start
Content-Type: application/json

{
  "username": "user123"
}
```

#### 2. 完成注册
```
POST /webauthn/register/finish
Content-Type: application/json

{
  "username": "user123",
  // WebAuthn 凭证创建响应
}
```

#### 3. 开始登录
```
POST /webauthn/login/start
Content-Type: application/json

{
  "username": "user123"
}
```

#### 4. 完成登录
```
POST /webauthn/login/finish
Content-Type: application/json

{
  "username": "user123",
  // WebAuthn 凭证断言响应
}
```

### 受保护的路由

#### 获取用户信息
```
GET /profile
Authorization: Bearer <token>
```

#### 管理员路由
```
GET /admin
Authorization: Bearer <token>
```

### API 网关代理

所有 `/api/*` 路由会被代理到 `http://localhost:5100`，并在请求头中注入用户信息：
- `X-User-ID`: 用户 ID
- `X-Username`: 用户名
- `X-User-Role`: 用户角色

```
/api/* (需要 admin 权限)
Authorization: Bearer <token>
```

## 配置说明

### WebAuthn 配置

在 `webauthn/webauthn.go` 中配置：

```go
config := &webauthn.Config{
    RPDisplayName: "WebAuthn Demo",  // 显示名称
    RPID:          "localhost",      // 依赖方 ID
    RPOrigins: []string{              // 允许的源
        "http://localhost:5000",
        "http://127.0.0.1:5000",
    },
}
```

### CORS 配置

在 `router/router.go` 中配置允许的源：

```go
AllowOrigins: []string{
    "http://localhost:5000",
    "http://127.0.0.1:5000",
}
```

### 业务服务代理

在 `router/router.go` 中修改业务服务地址：

```go
targetURL, _ := url.Parse("http://localhost:5100")
```

## 数据库

项目使用 SQLite 数据库，数据库文件为 `webauthn.db`。

### 用户表结构

```go
type User struct {
    ID          uint
    Username    string
    Role        string
    Credentials []webauthn.Credential
}
```

### 查看数据

```bash
sqlite3 webauthn.db "SELECT * FROM users"
```

## 安全说明

1. **Token 有效期**: 访问令牌默认有效期为 24 小时
2. **加密算法**: 使用 Ed25519 非对称加密和 PASETO V2 标准
3. **CORS 配置**: 请根据生产环境调整允许的源
4. **HTTPS**: 生产环境建议使用 HTTPS

## 开发说明

### 添加新的受保护路由

```go
auth := r.Group("/").Use(middleware.AuthMiddleware(tokenMaker))
{
    auth.GET("/new-route", handler)
}
```

### 添加需要特定角色的路由

```go
admin := r.Group("/").
    Use(middleware.AuthMiddleware(tokenMaker)).
    Use(middleware.RoleMiddleware("admin"))
{
    admin.GET("/admin-route", handler)
}
```

## 常见问题

### Q: 浏览器提示不支持 WebAuthn
A: 请确保使用现代浏览器（Chrome、Edge、Firefox、Safari 等）的最新版本

### Q: 注册失败
A: 检查浏览器是否支持 WebAuthn，以及设备是否支持安全密钥

### Q: Token 验证失败
A: 检查 Token 是否过期，以及 Authorization header 格式是否正确

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

