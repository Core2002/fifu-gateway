# ============================================
# Global args
# ============================================
ARG GO_IMAGE=golang:1.25-alpine
ARG ALPINE_IMAGE=alpine:latest

# 第一阶段：构建阶段
FROM ${GO_IMAGE} AS builder

# 使用中科大镜像源（如果需要国内加速）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories || true

# 安装构建依赖
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# 设置工作目录
WORKDIR /build

# 设置 Go 模块代理（国内加速）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.org

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
FROM ${ALPINE_IMAGE}

# 使用中科大镜像源（如果需要国内加速）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories || true

# 安装运行时依赖
RUN apk update && apk --no-cache add ca-certificates tzdata

# 设置时区为中国标准时间
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /build/fifu-gateway /build/.env.* ./

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
