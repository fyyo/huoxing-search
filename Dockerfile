# 多阶段构建 - 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装必要的工具
RUN apk add --no-cache git

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制整个项目源代码（包括pansou子目录）
COPY . .

# 编译xinyue-go主程序（pansou已作为库集成）
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/xinyue-server ./cmd/server

# 最终运行镜像
FROM alpine:latest

# 安装必要工具
RUN apk --no-cache add ca-certificates tzdata wget

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/xinyue-server /app/xinyue-server

# 复制必要的资源文件
COPY --from=builder /build/install /app/install
COPY --from=builder /build/web /app/web

# 创建必要的目录
RUN mkdir -p /app/logs /app/cache

# 暴露端口（只需要xinyue-go的API端口）
EXPOSE 6060

# 启动xinyue-go服务
CMD ["/app/xinyue-server"]