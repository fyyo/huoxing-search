.PHONY: help build run test clean docker-build docker-up docker-down install-deps lint fmt

# 变量定义
APP_NAME=xinyue-go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(shell go version | awk '{print $$3}')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)"

# 默认目标
help: ## 显示帮助信息
	@echo "Xinyue-Go 构建工具"
	@echo ""
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-deps: ## 安装依赖
	@echo "安装Go依赖..."
	go mod download
	go mod tidy

build: ## 编译项目
	@echo "编译 $(APP_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(APP_NAME) cmd/server/main.go
	@echo "编译完成: bin/$(APP_NAME)"

build-linux: ## 交叉编译Linux版本
	@echo "编译Linux版本..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 cmd/server/main.go
	@echo "编译完成: bin/$(APP_NAME)-linux-amd64"

build-windows: ## 交叉编译Windows版本
	@echo "编译Windows版本..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe cmd/server/main.go
	@echo "编译完成: bin/$(APP_NAME)-windows-amd64.exe"

build-mac: ## 交叉编译Mac版本
	@echo "编译Mac版本..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 cmd/server/main.go
	@echo "编译完成: bin/$(APP_NAME)-darwin-amd64"

build-all: build-linux build-windows build-mac ## 编译所有平台版本

run: ## 运行项目
	@echo "运行 $(APP_NAME)..."
	go run cmd/server/main.go

dev: ## 开发模式运行(热重载需要安装air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "请先安装air: go install github.com/cosmtrek/air@latest"; \
		echo "或直接运行: make run"; \
	fi

test: ## 运行测试
	@echo "运行测试..."
	go test -v -cover ./...

test-coverage: ## 生成测试覆盖率报告
	@echo "生成覆盖率报告..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

lint: ## 代码检查
	@echo "运行代码检查..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "请先安装golangci-lint: https://golangci-lint.run/usage/install/"; \
	fi

fmt: ## 格式化代码
	@echo "格式化代码..."
	go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	fi

clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf logs/
	rm -f coverage.out coverage.html
	go clean

docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest
	@echo "镜像构建完成: $(APP_NAME):$(VERSION)"

docker-up: ## 启动Docker容器
	@echo "启动Docker容器..."
	docker-compose up -d
	@echo "容器已启动,查看日志: make docker-logs"

docker-down: ## 停止Docker容器
	@echo "停止Docker容器..."
	docker-compose down

docker-restart: docker-down docker-up ## 重启Docker容器

docker-logs: ## 查看Docker日志
	docker-compose logs -f api

docker-ps: ## 查看运行中的容器
	docker-compose ps

init-db: ## 初始化数据库
	@echo "初始化数据库..."
	mysql -h localhost -u root -p xinyue < deploy/mysql/init.sql
	@echo "数据库初始化完成"

migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	# TODO: 添加迁移工具

gen-swagger: ## 生成Swagger文档
	@echo "生成API文档..."
	@if command -v swag > /dev/null; then \
		swag init -g cmd/server/main.go -o docs/swagger; \
	else \
		echo "请先安装swag: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

mod-update: ## 更新依赖
	@echo "更新依赖..."
	go get -u ./...
	go mod tidy

mod-vendor: ## 创建vendor目录
	@echo "创建vendor目录..."
	go mod vendor

version: ## 显示版本信息
	@echo "应用名称: $(APP_NAME)"
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Go版本: $(GO_VERSION)"

install: build ## 安装到系统
	@echo "安装 $(APP_NAME) 到 /usr/local/bin..."
	sudo cp bin/$(APP_NAME) /usr/local/bin/
	@echo "安装完成"

uninstall: ## 卸载
	@echo "卸载 $(APP_NAME)..."
	sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "卸载完成"