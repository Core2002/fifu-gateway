# 第一阶段：构建阶段
FROM docker.1ms.run/golang:1.25-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# 设置工作目录
WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用（禁用 CGO 可以减小体积，但本项目使用了 SQLite 需要 CGO）
# 使用 -ldflags 减小二进制文件大小
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -tags sqlite_omit_load_extension \
    -o fifu-gateway .

# 第二阶段：运行阶段
FROM alpine:latest

# 安装运行时依赖（如果需要 SQLite）
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为中国标准时间
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /build/fifu-gateway .

# 创建数据目录并设置权限
RUN mkdir -p /app/data && \
    chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 5000

# 设置环境变量
ENV GIN_MODE=release

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:5000/ping || exit 1

# 启动应用
CMD ["./fifu-gateway"]
