#!/bin/bash

# Xinyue-Go 一键部署脚本
# 用途：快速部署和管理Xinyue-Go服务

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose未安装，请先安装Docker Compose"
        exit 1
    fi
    
    print_info "Docker环境检查通过"
}

# 检查配置文件
check_config() {
    if [ ! -f "config.yaml" ]; then
        print_warning "未找到config.yaml，正在从模板创建..."
        cp config.yaml.example config.yaml
        print_info "已创建config.yaml，请编辑后重新运行部署脚本"
        print_warning "必须配置的项：database.host, database.password, jwt.secret"
        exit 0
    fi
    print_info "配置文件检查通过"
}

# 构建镜像
build_image() {
    print_info "开始构建Docker镜像..."
    docker-compose build --no-cache
    print_info "镜像构建完成"
}

# 启动服务
start_service() {
    print_info "启动服务..."
    docker-compose up -d
    print_info "服务已启动"
    
    # 等待服务启动
    sleep 5
    
    # 检查服务状态
    if docker-compose ps | grep -q "Up"; then
        print_info "服务运行正常"
        print_info "访问地址："
        print_info "  - 前台: http://localhost:6060"
        print_info "  - 管理后台: http://localhost:6060/admin"
        print_info "  - Pansou搜索: http://localhost:8888"
    else
        print_error "服务启动失败，请查看日志："
        docker-compose logs
        exit 1
    fi
}

# 停止服务
stop_service() {
    print_info "停止服务..."
    docker-compose down
    print_info "服务已停止"
}

# 重启服务
restart_service() {
    print_info "重启服务..."
    docker-compose restart
    print_info "服务已重启"
}

# 查看日志
view_logs() {
    docker-compose logs -f --tail=100
}

# 查看状态
check_status() {
    print_info "服务状态："
    docker-compose ps
    echo ""
    print_info "健康检查："
    curl -s http://localhost:6060/api/health || print_error "API服务无响应"
    curl -s http://localhost:8888/health || print_error "Pansou服务无响应"
}

# 更新服务
update_service() {
    print_info "更新服务..."
    
    # 拉取最新代码
    if [ -d ".git" ]; then
        print_info "拉取最新代码..."
        git pull
    fi
    
    # 停止服务
    stop_service
    
    # 重新构建
    build_image
    
    # 启动服务
    start_service
    
    print_info "更新完成"
}

# 显示帮助信息
show_help() {
    echo "Xinyue-Go 部署管理脚本"
    echo ""
    echo "用法: ./deploy.sh [命令]"
    echo ""
    echo "命令："
    echo "  install    - 首次安装并启动服务"
    echo "  start      - 启动服务"
    echo "  stop       - 停止服务"
    echo "  restart    - 重启服务"
    echo "  status     - 查看服务状态"
    echo "  logs       - 查看实时日志"
    echo "  update     - 更新并重新部署服务"
    echo "  help       - 显示此帮助信息"
    echo ""
}

# 主函数
main() {
    case "$1" in
        install)
            check_docker
            check_config
            build_image
            start_service
            ;;
        start)
            check_docker
            start_service
            ;;
        stop)
            stop_service
            ;;
        restart)
            restart_service
            ;;
        status)
            check_status
            ;;
        logs)
            view_logs
            ;;
        update)
            check_docker
            update_service
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"