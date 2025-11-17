# Xinyue-Go Docker 部署指南

## 📋 部署说明

本项目提供一体化Docker镜像，将Xinyue-Go API和Pansou搜索引擎打包在同一个容器中，简化部署流程。

## 🔧 前置要求

### 服务器环境
- 操作系统：Linux (推荐 Ubuntu 20.04+/CentOS 7+)
- Docker：20.10+
- Docker Compose：1.29+
- 内存：至少 2GB
- 磁盘：至少 10GB

### 数据库要求
- MySQL 5.7+ 或 8.0+ (需自行准备)
- 建议使用独立的MySQL服务器或云数据库

### 域名和SSL证书 (微信功能需要)
- 已备案的域名
- SSL证书 (可使用Let's Encrypt免费证书)
- 微信公众平台要求必须使用HTTPS

## 🚀 快速部署

### 1. 准备配置文件

```bash
# 复制配置模板
cp config.yaml.example config.yaml

# 编辑配置文件
vim config.yaml
```

**重要配置项：**

```yaml
database:
  host: your-mysql-host      # MySQL服务器地址
  port: 3306
  username: root
  password: your-password    # MySQL密码
  database: xinyue           # 数据库名

pansou:
  url: http://localhost:8888 # 容器内部通信

jwt:
  secret: your-random-secret-key  # 请修改为随机字符串
```

### 2. 初始化数据库

```bash
# 在MySQL中创建数据库
mysql -h your-mysql-host -u root -p -e "CREATE DATABASE xinyue DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 导入数据库结构
mysql -h your-mysql-host -u root -p xinyue < install/data.sql
```

### 3. 构建并启动服务

```bash
# 构建Docker镜像
docker-compose build

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 4. 验证部署

```bash
# 检查服务状态
docker-compose ps

# 检查API健康状态
curl http://localhost:6060/api/health

# 检查Pansou健康状态
curl http://localhost:8888/health
```

## 🌐 配置反向代理 (微信回调需要HTTPS)

### 使用Nginx

创建Nginx配置文件 `/etc/nginx/sites-available/xinyue`:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    # HTTP重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # SSL证书配置
    ssl_certificate /path/to/your/fullchain.pem;
    ssl_certificate_key /path/to/your/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # 日志
    access_log /var/log/nginx/xinyue-access.log;
    error_log /var/log/nginx/xinyue-error.log;

    # 代理到Docker容器
    location / {
        proxy_pass http://localhost:6060;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket支持 (如果需要)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # API接口
    location /api/ {
        proxy_pass http://localhost:6060;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

启用配置：

```bash
# 创建软链接
ln -s /etc/nginx/sites-available/xinyue /etc/nginx/sites-enabled/

# 测试配置
nginx -t

# 重载Nginx
systemctl reload nginx
```

### 使用Let's Encrypt获取免费SSL证书

```bash
# 安装certbot
apt-get update
apt-get install certbot python3-certbot-nginx

# 获取证书
certbot --nginx -d your-domain.com

# 证书会自动配置到Nginx
```

## 📱 配置微信回调

### 1. 微信对话开放平台

1. 登录 [微信对话开放平台](https://openai.weixin.qq.com/)
2. 创建技能，选择"智能对话"
3. 配置回调URL：
   ```
   https://your-domain.com/api/wechat/chatbot/callback
   ```
4. 在系统后台"微信配置"页面填入：
   - AppID
   - Token
   - EncodingAESKey
5. 点击"测试连接"验证配置

### 2. 微信公众号

1. 登录 [微信公众平台](https://mp.weixin.qq.com/)
2. 进入"基本配置"
3. 配置服务器地址：
   ```
   https://your-domain.com/api/wechat/official/callback
   ```
4. 在系统后台"微信配置"页面填入Token
5. 点击"测试连接"验证配置
6. 在微信公众平台点击"启用"

## 🔍 访问系统

- **前台页面**：https://your-domain.com
- **管理后台**：https://your-domain.com/admin
- **默认账号**：admin / admin123 (首次登录后请修改密码)

## 📊 日志查看

```bash
# 查看所有日志
docker-compose logs -f

# 查看Xinyue日志
docker-compose logs -f xinyue

# 查看容器内的详细日志
docker exec -it xinyue-app tail -f /app/logs/xinyue.log
docker exec -it xinyue-app tail -f /app/logs/pansou.log
```

## 🔄 更新部署

```bash
# 停止服务
docker-compose down

# 拉取最新代码
git pull

# 重新构建镜像
docker-compose build --no-cache

# 启动服务
docker-compose up -d
```

## 🛠 故障排查

### 服务无法启动

```bash
# 检查容器状态
docker-compose ps

# 查看详细错误日志
docker-compose logs

# 检查配置文件
cat config.yaml
```

### 数据库连接失败

1. 检查MySQL服务是否运行
2. 检查防火墙是否允许3306端口
3. 检查config.yaml中的数据库配置
4. 测试数据库连接：
   ```bash
   mysql -h your-mysql-host -u root -p
   ```

### 微信回调失败

1. 确认域名已正确解析
2. 确认SSL证书有效
3. 检查回调URL是否可以从外网访问
4. 在系统后台点击"测试连接"查看详细错误

### 端口冲突

如果6060或8888端口被占用，修改docker-compose.yml：

```yaml
ports:
  - "7070:6060"  # 使用7070端口代替6060
  - "9999:8888"  # 使用9999端口代替8888
```

## 🔐 安全建议

1. **修改默认密码**：首次登录后立即修改管理员密码
2. **定期备份**：定期备份MySQL数据库
3. **更新JWT密钥**：在config.yaml中设置强随机JWT密钥
4. **防火墙配置**：只开放必要端口（80, 443）
5. **日志监控**：定期检查日志文件，关注异常访问

## 📞 技术支持

如有问题，请查看项目文档或联系技术支持。