-- Huoxing-Go 数据库初始化脚本

-- 管理员表
CREATE TABLE IF NOT EXISTS `qf_admin` (
  `admin_id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password` varchar(255) NOT NULL COMMENT '密码',
  `nickname` varchar(50) DEFAULT NULL COMMENT '昵称',
  `email` varchar(100) DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `avatar` varchar(255) DEFAULT NULL COMMENT '头像',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态:0禁用,1启用',
  `last_login_time` bigint(20) DEFAULT NULL COMMENT '最后登录时间',
  `create_time` bigint(20) NOT NULL COMMENT '创建时间',
  `update_time` bigint(20) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`admin_id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员表';

-- 资源表
CREATE TABLE IF NOT EXISTS `qf_source` (
  `source_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL COMMENT '资源标题',
  `url` varchar(500) NOT NULL COMMENT '分享链接',
  `content` varchar(500) DEFAULT NULL COMMENT '原始链接',
  `password` varchar(50) DEFAULT NULL COMMENT '提取码',
  `is_type` tinyint(4) DEFAULT '0' COMMENT '网盘类型:0夸克,2百度,3阿里,4UC,5迅雷',
  `fid` varchar(500) DEFAULT NULL COMMENT '文件ID',
  `size` bigint(20) DEFAULT NULL COMMENT '文件大小',
  `source_name` varchar(100) DEFAULT NULL COMMENT '原始来源名称',
  `source_time` varchar(50) DEFAULT NULL COMMENT '原始资源时间',
  `is_time` tinyint(4) DEFAULT '0' COMMENT '是否临时:0否,1是',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态:0禁用,1启用',
  `view_count` int(11) DEFAULT '0' COMMENT '查看次数',
  `transfer_count` int(11) DEFAULT '0' COMMENT '转存次数',
  `category_id` int(11) DEFAULT NULL COMMENT '分类ID',
  `create_time` bigint(20) NOT NULL COMMENT '创建时间',
  `update_time` bigint(20) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`source_id`),
  KEY `idx_is_type` (`is_type`),
  KEY `idx_status` (`status`),
  KEY `idx_is_time` (`is_time`),
  KEY `idx_category_id` (`category_id`),
  KEY `idx_create_time` (`create_time`),
  KEY `idx_is_time_create_time` (`is_time`,`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源表';

-- 分类表
CREATE TABLE IF NOT EXISTS `qf_source_category` (
  `category_id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '分类名称',
  `keyword` varchar(500) DEFAULT NULL COMMENT '关键词',
  `sort` int(11) DEFAULT '0' COMMENT '排序',
  `is_type` tinyint(4) DEFAULT '0' COMMENT '类型:0网络,1本地',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态:0禁用,1启用',
  `create_time` bigint(20) NOT NULL COMMENT '创建时间',
  `update_time` bigint(20) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`category_id`),
  KEY `idx_is_type` (`is_type`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源分类表';

-- API配置表
CREATE TABLE IF NOT EXISTS `qf_api_list` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(100) NOT NULL COMMENT '线路名称',
  `type` varchar(20) NOT NULL DEFAULT 'api' COMMENT '接口类型:api,html,tg',
  `pantype` tinyint(1) NOT NULL DEFAULT '0' COMMENT '网盘类型:0夸克,2百度,3阿里,4UC,5迅雷',
  `url` varchar(255) DEFAULT NULL COMMENT '请求地址',
  `method` varchar(10) DEFAULT 'GET' COMMENT '请求方式',
  `fixed_params` text COMMENT '固定参数(JSON)',
  `headers` text COMMENT '请求头(JSON)',
  `field_map` text COMMENT '字段映射(JSON)',
  `html_item` varchar(255) DEFAULT NULL COMMENT 'HTML列表项选择器',
  `html_title` varchar(255) DEFAULT NULL COMMENT 'HTML标题选择器',
  `html_url` varchar(255) DEFAULT NULL COMMENT 'HTML链接选择器',
  `html_type` tinyint(4) DEFAULT '0' COMMENT 'HTML类型',
  `html_url2` varchar(255) DEFAULT NULL COMMENT 'HTML备用链接选择器',
  `weight` int(11) DEFAULT '0' COMMENT '权重',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态:0禁用,1启用',
  `create_time` bigint(20) NOT NULL DEFAULT '0' COMMENT '创建时间',
  `update_time` bigint(20) NOT NULL DEFAULT '0' COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API配置表';

-- 配置表
CREATE TABLE IF NOT EXISTS `qf_conf` (
  `conf_id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '配置名称',
  `value` text COMMENT '配置值',
  `title` varchar(100) DEFAULT NULL COMMENT '标题',
  `description` varchar(255) DEFAULT NULL COMMENT '描述',
  `group` tinyint(4) DEFAULT '0' COMMENT '分组',
  `type` tinyint(4) DEFAULT '1' COMMENT '类型:1文本,2数字,3开关',
  `options` text COMMENT '选项(JSON)',
  `sort` int(11) DEFAULT '0' COMMENT '排序',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态',
  `create_time` bigint(20) DEFAULT NULL COMMENT '创建时间',
  `update_time` bigint(20) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`conf_id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';

-- 操作日志表
CREATE TABLE IF NOT EXISTS `qf_log` (
  `log_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` int(11) DEFAULT NULL COMMENT '管理员ID',
  `username` varchar(50) DEFAULT NULL COMMENT '用户名',
  `action` varchar(100) NOT NULL COMMENT '操作',
  `module` varchar(50) DEFAULT NULL COMMENT '模块',
  `method` varchar(10) DEFAULT NULL COMMENT '请求方法',
  `url` varchar(500) DEFAULT NULL COMMENT '请求URL',
  `ip` varchar(50) DEFAULT NULL COMMENT 'IP地址',
  `user_agent` varchar(500) DEFAULT NULL COMMENT 'User-Agent',
  `request_data` text COMMENT '请求数据',
  `response_data` text COMMENT '响应数据',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态:0失败,1成功',
  `create_time` bigint(20) NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`log_id`),
  KEY `idx_admin_id` (`admin_id`),
  KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作日志表';

-- ========================================
-- 初始数据
-- ========================================

-- 注意: 管理员账号将在Web安装向导中创建，此处不预先插入

-- 插入默认分类
INSERT INTO `qf_source_category` (`name`, `keyword`, `sort`, `is_type`, `status`, `create_time`, `update_time`) VALUES
('电影', '电影,影视,大片', 1, 0, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('电视剧', '电视剧,连续剧,美剧,韩剧', 2, 0, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('动漫', '动漫,动画,番剧', 3, 0, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('综艺', '综艺,娱乐,真人秀', 4, 0, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('纪录片', '纪录片,记录片', 5, 0, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());

-- 插入默认系统配置
INSERT INTO `qf_conf` (`name`, `value`, `title`, `description`, `group`, `type`, `sort`, `status`, `create_time`, `update_time`) VALUES
('site_name', '火星网盘搜索', '网站名称', '网站的名称', 0, 1, 1, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('site_keywords', '火星搜索,网盘搜索,资源搜索,夸克网盘,百度网盘', '网站关键词', 'SEO关键词,用逗号分隔', 0, 1, 2, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('site_description', '火星网盘搜索系统 - 支持多网盘资源搜索与转存', '网站描述', 'SEO描述信息', 0, 1, 3, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('max_search_results', '5', '最大搜索结果数', '单次搜索返回的最大结果数', 1, 2, 10, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('max_transfer_count', '2', '最大转存数量', '单次转存的最大数量', 1, 2, 11, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('cache_expire', '60', '缓存过期时间', '搜索结果缓存时间(秒)', 1, 2, 12, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('ban_keywords', '', '屏蔽关键词', '屏蔽的搜索关键词,用逗号分隔', 1, 1, 13, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- Pansou搜索引擎配置
('pansou_url', 'http://localhost:8888', 'Pansou服务地址', 'Pansou搜索引擎的API地址', 1, 1, 14, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('pansou_timeout', '30', 'Pansou超时时间', 'Pansou API调用超时时间(秒)', 1, 2, 15, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 网盘配置 - 夸克网盘 (group=2)
('quark_cookie', '', '夸克网盘Cookie', '夸克网盘的Cookie值', 2, 1, 20, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('quark_file', '0', '夸克默认文件夹ID', '转存资源默认保存的文件夹ID，0表示根目录', 2, 1, 21, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('quark_file_time', '0', '夸克临时文件夹ID', '临时有效期资源的存储文件夹ID', 2, 1, 22, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('quark_banned', '广告,推广,福利,热门', '夸克广告过滤', '广告文件关键词,用逗号分隔', 2, 1, 23, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 网盘配置 - 百度网盘
('baidu_cookie', '', '百度网盘Cookie', '百度网盘的Cookie值', 2, 1, 30, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('baidu_file', '/转存', '百度默认路径', '转存资源默认保存的文件夹路径', 2, 1, 31, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('baidu_file_time', '/临时资源', '百度临时路径', '临时有效期资源的存储文件夹路径', 2, 1, 32, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 网盘配置 - 阿里云盘
('Authorization', '', '阿里RefreshToken', '阿里云盘的RefreshToken', 2, 1, 40, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('ali_file', 'root', '阿里默认文件夹ID', '转存资源默认保存的文件夹ID，root表示根目录', 2, 1, 41, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('ali_file_time', 'root', '阿里临时文件夹ID', '临时有效期资源的存储文件夹ID', 2, 1, 42, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 网盘配置 - UC网盘
('uc_cookie', '', 'UC网盘Cookie', 'UC网盘的Cookie值', 2, 1, 50, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('uc_file', '0', 'UC默认文件夹ID', '转存资源默认保存的文件夹ID，0表示根目录', 2, 1, 51, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('uc_file_time', '0', 'UC临时文件夹ID', '临时有效期资源的存储文件夹ID', 2, 1, 52, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 网盘配置 - 迅雷网盘
('xunlei_cookie', '', '迅雷RefreshToken', '迅雷网盘的RefreshToken值', 2, 1, 60, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('xunlei_file', '', '迅雷默认文件夹ID', '转存资源默认保存的文件夹ID，留空表示根目录', 2, 1, 61, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('xunlei_file_time', '', '迅雷临时文件夹ID', '临时有效期资源的存储文件夹ID', 2, 1, 62, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 系统功能配置 (group=4)
('delete_netdisk_files', '0', '清理网盘文件', '清理临时资源时是否同时删除网盘中的文件：0=仅删除数据库记录，1=同时删除网盘文件', 4, 3, 90, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 微信配置 - 对话开放平台 (group=3)
('wx_chat_token', '', '对话平台Token', '微信对话开放平台的Token', 3, 1, 70, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('wx_chat_aes_key', '', '对话平台AESKey', '微信对话开放平台的EncodingAESKey', 3, 1, 71, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('wx_chat_appid', '', '对话平台AppID', '微信对话开放平台的AppID', 3, 1, 72, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 微信配置 - 公众号
('wx_official_token', '', '公众号Token', '微信公众号的Token', 3, 1, 80, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('wx_official_appid', '', '公众号AppID', '微信公众号的AppID', 3, 1, 81, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('wx_official_secret', '', '公众号Secret', '微信公众号的AppSecret', 3, 1, 82, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('wx_official_aeskey', '', '公众号EncodingAESKey', '微信公众号的消息加密密钥（可选）', 3, 1, 83, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),

-- 微信配置 - 对话平台个性化
('wx_chatbot_name', '火星搜索', '机器人名称', '微信对话平台显示的机器人名称', 3, 1, 84, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('wx_chatbot_welcome', '', '自定义欢迎语', '用户首次访问时的欢迎消息（留空使用默认）', 3, 1, 85, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());

