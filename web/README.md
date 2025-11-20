# Huoxing-Go Web前端（前后端一体化）

## 架构说明

采用**前后端一体化**架构，类似原PHP项目：
- Go服务器直接提供HTML页面
- 静态资源（CSS/JS/图片）由Go服务
- 使用Go模板引擎渲染HTML
- AJAX调用同域API接口

## 目录结构

```
web/
├── static/              # 静态资源
│   ├── css/            # 样式文件
│   ├── js/             # JavaScript文件
│   ├── images/         # 图片资源
│   └── lib/            # 第三方库（Vue3、Element Plus等）
│
├── templates/          # HTML模板
│   ├── index/         # 前台页面
│   │   ├── layout.html      # 前台布局
│   │   ├── home.html        # 首页
│   │   ├── search.html      # 搜索页
│   │   └── detail.html      # 详情页
│   │
│   └── admin/         # 后台页面
│       ├── layout.html      # 后台布局
│       ├── login.html       # 登录页
│       ├── dashboard.html   # 控制台
│       ├── source.html      # 资源管理
│       ├── config.html      # 系统配置
│       └── user.html        # 用户管理
│
└── README.md          # 本文件
```

## 技术栈

### 前端框架
- **Vue 3** - 通过CDN引入，无需npm构建
- **Element Plus** - UI组件库
- **Axios** - HTTP请求
- **Vue Router** - 前端路由（SPA模式）

### 为什么不用npm构建？

1. **简化部署** - 无需Node.js环境
2. **快速开发** - 修改即生效，无需编译
3. **前后端一体** - Go服务器直接提供所有资源
4. **类似原项目** - 保持原PHP项目的部署方式

## 使用方式

### 开发模式
```bash
# 启动Go服务
cd huoxing-search
go run cmd/server/main.go

# 访问
http://localhost:6060           # 前台
http://localhost:6060/admin     # 后台
```

### 生产模式
```bash
# 使用Docker
docker-compose up -d

# web目录会被自动打包到Docker镜像中
```

## 页面路由

### 前台
- `/` - 首页
- `/search?keyword=xxx` - 搜索页
- `/detail/:id` - 详情页

### 后台
- `/admin/login` - 登录页
- `/admin` - 控制台
- `/admin/source` - 资源管理
- `/admin/config` - 系统配置
- `/admin/user` - 用户管理

## API调用示例

```javascript
// 搜索
axios.post('/api/search', {
  keyword: '关键词',
  pan_type: 0
})

// 获取配置
axios.get('/api/admin/config')

// 保存配置
axios.post('/api/admin/config', {
  site_name: '新网站名',
  // ...其他配置
})
```

## 开发指南

### 1. 添加新页面

1) 在 `web/templates/` 创建HTML模板
2) 在 `internal/api/frontend.go` 添加路由
3) 刷新浏览器即可看到效果

### 2. 修改样式

直接编辑 `web/static/css/` 中的文件，刷新即生效

### 3. 添加新功能

在 `web/static/js/` 中添加JavaScript文件，在HTML中引入

## 与原PHP项目对比

| 功能 | PHP项目 | Go项目（本方案） |
|------|---------|------------------|
| 后端语言 | PHP | Go |
| 前端框架 | jQuery + 原生JS | Vue 3 |
| UI库 | 自定义 | Element Plus |
| 模板引擎 | ThinkPHP | Go Template |
| 构建工具 | 无 | 无（CDN引入） |
| 部署方式 | Apache/Nginx + PHP | Docker / 单一二进制 |
| 性能 | 一般 | 高（Go原生性能） |

## 注意事项

1. Vue3通过CDN引入，无需npm
2. 所有静态资源都在web目录下
3. Go服务器会自动处理静态文件和模板
4. 修改HTML/CSS/JS后刷新即可，无需重启Go服务