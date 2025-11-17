# Xinyue-Go 网盘搜索系统

> **版本**: v1.0
> **语言**: Go 1.21+
> **框架**: Gin + GORM
> **状态**: ✅ 重构完成，已集成 Pansou 搜索引擎

一个高性能的网盘资源搜索和转存系统，支持多网盘、全网搜索、微信回调等功能。

**🎉 重大更新**：已完成从 PHP 到 Go 的完整重构，采用 **Pansou 深度集成方案**，实现单进程部署，性能提升 5-10 倍！

📖 **重要文档**：
- [PANSOU_INTEGRATION.md](PANSOU_INTEGRATION.md) - Pansou 集成详细说明
- [REFACTORING_SUMMARY.md](REFACTORING_SUMMARY.md) - 重构完成总结
- [DEPLOY.md](DEPLOY.md) - 部署指南

---

## ✨ 特性

### 🚀 高性能
- **Go语言重写**：相比PHP版本性能提升5-10倍
- **并发处理**：支持1000+ QPS
- **智能缓存**：Redis缓存，搜索响应 < 1秒
- **低资源占用**：内存 < 200MB，CPU < 40%

### 🔍 搜索功能
- **整合Pansou搜索引擎**：50+搜索插件
- **多网盘支持**：夸克、百度、阿里云盘、UC、迅雷
- **全网搜索**：TG频道、网页爬虫、API接口
- **智能排序**：结果自动去重和优先级排序

### 💾 转存功能
- **一键转存**：支持5种网盘快速转存
- **并发处理**：最多5个同时转存，速度提升3-4倍
- **批量导入**：支持批量链接导入
- **智能过滤**：自动过滤无效链接

### 📱 微信集成
- **对话开放平台**：支持智能对话
- **公众号回调**：支持公众号消息回调
- **配置测试**：一键测试连接功能

### 🎨 现代化界面
- **Vue3 前端**：响应式设计
- **Element Plus UI**：组件丰富
- **管理后台**：功能完善的后台管理

---

## 📦 快速开始

### 方式一：Docker部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/your-repo/xinyue-go.git
cd xinyue-go

# 2. 复制配置文件
cp config.yaml.example config.yaml

# 3. 编辑配置（必须配置数据库信息）
vim config.yaml

# 4. 一键部署
chmod +x deploy.sh
./deploy.sh install

# 5. 查看服务状态
./deploy.sh status
```

**Docker部署说明：**
- ✅ 单一服务进程：Pansou 已作为核心库编译进主程序
- ✅ 轻量镜像：约 80MB，启动时间 < 2秒
- ✅ 低资源占用：内存 ~180MB，CPU < 40%
- ✅ 支持健康检查和自动重启
- 📖 详细部署文档请查看 [DEPLOY.md](DEPLOY.md)
- 📖 集成方案说明请查看 [PANSOU_INTEGRATION.md](PANSOU_INTEGRATION.md)

### 方式二：手动编译

```bash
# 1. 安装Go 1.21+
# 2. 下载依赖
go mod download

# 3. 编译
go build -o xinyue-server cmd/server/main.go

# 4. 运行
./xinyue-server
```

---

## 🔧 配置说明

### 数据库配置

```yaml
database:
  host: your-mysql-host    # MySQL地址
  port: 3306
  username: root
  password: your-password  # 修改为实际密码
  database: xinyue
```

### 网盘配置

在管理后台"网盘配置"页面填写：
- **夸克网盘**：Cookie
- **百度网盘**：Cookie
- **阿里云盘**：RefreshToken
- **UC网盘**：Cookie
- **迅雷网盘**：Cookie

配置完成后点击"测试连接"验证。

### 微信配置

在管理后台"微信配置"页面填写：
- **对话开放平台**：AppID、Token、EncodingAESKey
- **公众号**：Token

配置完成后点击"测试连接"验证。

**注意**：微信回调需要HTTPS，请配置Nginx反向代理。

---

## 📱 访问系统

- **前台页面**：http://localhost:6060
- **管理后台**：http://localhost:6060/admin
- **默认账号**：admin / admin123

**首次登录后请立即修改密码！**

---

## 🏗️ 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端语言 | Go 1.21+ | 高性能、低资源 |
| Web框架 | Gin v1.9+ | 轻量高效 |
| ORM | GORM v2.0+ | 功能完善 |
| 数据库 | MySQL 8.0 | 稳定可靠 |
| 缓存 | Redis 7.0 | 高性能缓存 |
| 搜索引擎 | Pansou（深度集成） | 50+插件 |
| 前端框架 | Vue 3.x | 现代化 |
| UI库 | Element Plus | 组件丰富 |
| 容器化 | Docker | 标准化部署 |

---

## 📁 项目结构

```
xinyue-go/
├── cmd/server/              # 程序入口
├── internal/                # 核心代码
│   ├── api/                # HTTP处理器
│   ├── service/            # 业务逻辑
│   ├── repository/         # 数据访问
│   ├── model/              # 数据模型
│   ├── netdisk/            # 网盘SDK（5种）
│   ├── middleware/         # 中间件
│   └── pkg/                # 工具包
├── pansou/                 # ✅ Pansou搜索引擎（深度集成）
│   ├── init.go            # 初始化接口
│   ├── config/            # 配置管理
│   ├── model/             # 数据模型
│   ├── plugin/            # 50+搜索插件
│   ├── service/           # 搜索服务
│   └── util/              # 工具函数
├── web/                    # 前端代码
├── install/                # 安装文件
├── config.yaml.example     # 配置模板
├── docker-compose.yml      # Docker编排
├── Dockerfile              # Docker镜像（单服务）
├── deploy.sh               # 部署脚本
├── DEPLOY.md               # 部署文档
├── PANSOU_INTEGRATION.md   # ✅ Pansou集成说明
├── REFACTORING_SUMMARY.md  # ✅ 重构总结
└── README.md               # 项目说明
```

---

## 🔐 安全建议

1. **修改默认密码**：首次登录后立即修改
2. **JWT密钥**：在config.yaml中设置强随机密钥
3. **数据库密码**：使用强密码
4. **防火墙**：只开放必要端口（80, 443）
5. **定期备份**：定期备份MySQL数据

---

## 📊 性能对比

| 指标 | PHP版本 | Go版本 | 提升 |
|------|---------|--------|------|
| 搜索响应 | 2-5秒 | <1秒 | 5-10倍 |
| 并发能力 | 50-100 QPS | 1000+ QPS | 10倍 |
| 转存速度 | 3-8秒 | <2秒 | 3-4倍 |
| 内存占用 | 500MB-1GB | <200MB | 降低70% |
| CPU占用 | 60-80% | <40% | 降低50% |

---

## 🤝 贡献

欢迎提交Issue和Pull Request！

---

## 📄 开源协议

本项目仅供学习交流使用，请勿用于非法用途。

---

## 📞 支持

- **问题反馈**：提交Issue
- **交流群**：见原项目说明
- **文档**：查看 [DEPLOY.md](DEPLOY.md)

---

## 🙏 致谢

感谢以下项目：
- [Gin](https://github.com/gin-gonic/gin) - Web框架
- [GORM](https://github.com/go-gorm/gorm) - ORM
- [Vue3](https://github.com/vuejs/core) - 前端框架
- [Element Plus](https://github.com/element-plus/element-plus) - UI组件库
- [Pansou](https://github.com/your-repo/pansou) - 搜索引擎

---

**注意**：本项目仅供技术交流与学习使用，请勿将本项目用于任何违法用途！