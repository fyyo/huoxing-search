# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置 Go 镜像源
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/huoxing-server ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

ENV TZ=Asia/Shanghai

WORKDIR /app

# 复制文件
COPY --from=builder /app/huoxing-server /app/huoxing-server
COPY --from=builder /build/install /app/install
COPY --from=builder /build/web /app/web

# 创建目录
RUN mkdir -p /app/data /app/logs /app/cache

EXPOSE 6060

CMD ["/app/huoxing-server"]