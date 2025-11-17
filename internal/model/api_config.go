package model

import (
	"time"

	"gorm.io/gorm"
)

// APIConfig API配置模型
type APIConfig struct {
	ID          int    `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Name        string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Type        string `gorm:"column:type;type:varchar(20);default:'api'" json:"type"`
	PanType     int    `gorm:"column:pantype;type:tinyint;default:0" json:"pantype"`
	URL         string `gorm:"column:url;type:varchar(255)" json:"url"`
	Method      string `gorm:"column:method;type:varchar(10);default:'GET'" json:"method"`
	FixedParams string `gorm:"column:fixed_params;type:text" json:"fixed_params"`
	Headers     string `gorm:"column:headers;type:text" json:"headers"`
	FieldMap    string `gorm:"column:field_map;type:text" json:"field_map"`
	Count       int    `gorm:"column:count;type:int;default:0" json:"count"`
	Weight      int    `gorm:"column:weight;type:int;default:0" json:"weight"`
	Status      int    `gorm:"column:status;type:tinyint;default:1" json:"status"`
	CreateTime  int64  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  int64  `gorm:"column:update_time;not null" json:"update_time"`
}

// TableName 指定表名
func (APIConfig) TableName() string {
	return "qf_api_list"
}

// BeforeCreate GORM钩子 - 创建前
func (a *APIConfig) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	a.CreateTime = now
	a.UpdateTime = now
	return nil
}

// BeforeUpdate GORM钩子 - 更新前
func (a *APIConfig) BeforeUpdate(tx *gorm.DB) error {
	a.UpdateTime = time.Now().Unix()
	return nil
}