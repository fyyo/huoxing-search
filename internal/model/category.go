package model

import (
	"time"
	"gorm.io/gorm"
)

// Category 分类模型
type Category struct {
	CategoryID int    `gorm:"primaryKey;column:category_id;autoIncrement" json:"category_id"`
	Name       string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Keyword    string `gorm:"column:keyword;type:varchar(500)" json:"keyword"`
	Sort       int    `gorm:"column:sort;type:int;default:0" json:"sort"`
	IsType     int    `gorm:"column:is_type;type:tinyint;default:0" json:"is_type"` // 0: 网络, 1: 本地
	Status     int    `gorm:"column:status;type:tinyint;default:1" json:"status"`
	CreateTime int64  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime int64  `gorm:"column:update_time;not null" json:"update_time"`
}

// TableName 指定表名
func (Category) TableName() string {
	return "qf_source_category"
}

// BeforeCreate GORM钩子:创建前
func (c *Category) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	c.CreateTime = now
	c.UpdateTime = now
	return nil
}

// BeforeUpdate GORM钩子:更新前
func (c *Category) BeforeUpdate(tx *gorm.DB) error {
	c.UpdateTime = time.Now().Unix()
	return nil
}