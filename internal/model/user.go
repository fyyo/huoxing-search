package model

import (
	"time"
	"gorm.io/gorm"
)

// Admin 管理员模型
type Admin struct {
	AdminID       uint   `gorm:"primaryKey;column:admin_id;autoIncrement" json:"admin_id"`
	Username      string `gorm:"column:username;type:varchar(50);uniqueIndex;not null" json:"username"`
	Password      string `gorm:"column:password;type:varchar(255);not null" json:"-"` // 不返回密码
	Nickname      string `gorm:"column:nickname;type:varchar(50)" json:"nickname"`
	Email         string `gorm:"column:email;type:varchar(100)" json:"email"`
	Mobile        string `gorm:"column:mobile;type:varchar(20)" json:"mobile"`
	Status        int    `gorm:"column:status;type:tinyint;default:1" json:"status"` // 0=禁用 1=启用
	LastLoginTime int64  `gorm:"column:last_login_time" json:"last_login_time"`
	LastLoginIP   string `gorm:"column:last_login_ip;type:varchar(50)" json:"last_login_ip"`
	CreateTime    int64  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime    int64  `gorm:"column:update_time;not null" json:"update_time"`
}

// TableName 指定表名
func (Admin) TableName() string {
	return "qf_admin"
}

// User 用户模型(兼容旧代码，实际上就是Admin的别名)
type User = Admin

// BeforeCreate GORM钩子:创建前
func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	a.CreateTime = now
	a.UpdateTime = now
	return nil
}

// BeforeUpdate GORM钩子:更新前
func (a *Admin) BeforeUpdate(tx *gorm.DB) error {
	a.UpdateTime = time.Now().Unix()
	return nil
}

// IsAdmin 判断是否为管理员（管理员表中的都是管理员）
func (a *Admin) IsAdmin() bool {
	return true
}

// IsActive 判断用户是否激活
func (a *Admin) IsActive() bool {
	return a.Status == 1
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    string `json:"token"`
	UserInfo *Admin `json:"user_info"`
}

// ApiList 接口配置模型
type ApiList struct {
	ID          uint   `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Name        string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Type        string `gorm:"column:type;type:varchar(20);default:api" json:"type"` // api/html/tg
	PanType     int    `gorm:"column:pantype;type:tinyint;default:0" json:"pantype"`
	URL         string `gorm:"column:url;type:varchar(255)" json:"url"`
	Method      string `gorm:"column:method;type:varchar(10);default:GET" json:"method"`
	FixedParams string `gorm:"column:fixed_params;type:text" json:"fixed_params"`
	Headers     string `gorm:"column:headers;type:text" json:"headers"`
	FieldMap    string `gorm:"column:field_map;type:text" json:"field_map"`
	Count       int    `gorm:"column:count;default:0" json:"count"`
	Weight      int    `gorm:"column:weight;default:0" json:"weight"`
	Status      int    `gorm:"column:status;type:tinyint;default:1" json:"status"`
	CreateTime  int64  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  int64  `gorm:"column:update_time;not null" json:"update_time"`
}

// TableName 指定表名
func (ApiList) TableName() string {
	return "qf_api_list"
}