# 使用多阶段构建，减小最终镜像体积
# 构建阶段
FROM golang:alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置环境变量，使用代理加速下载依赖（如果在国内）
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制项目所有代码
COPY . .

# 编译项目 (禁用 CGO 可以确保在基于 Alpine 的容器中也能完美运行)
RUN CGO_ENABLED=0 GOOS=linux go build -o village-bill main.go

# 运行阶段
FROM alpine:latest

# 设置时区为上海（根据需要修改）
RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 从构建阶段复制编译好的可执行文件
COPY --from=builder /app/village-bill .

# 复制静态资源目录
COPY --from=builder /app/public ./public

# 确保有 uploads 目录，避免程序启动报错
RUN mkdir -p uploads

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./village-bill"]
