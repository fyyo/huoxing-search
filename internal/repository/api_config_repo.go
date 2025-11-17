package repository

import (
	"context"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/database"

	"gorm.io/gorm"
)

// APIConfigRepository API配置仓储接口
type APIConfigRepository interface {
	List(ctx context.Context, page, pageSize int) ([]model.APIConfig, int64, error)
	GetByID(ctx context.Context, id int) (*model.APIConfig, error)
	Create(ctx context.Context, config *model.APIConfig) error
	Update(ctx context.Context, config *model.APIConfig) error
	Delete(ctx context.Context, id int) error
	BatchDelete(ctx context.Context, ids []int) error
	UpdateStatus(ctx context.Context, id int, status int) error
}

type apiConfigRepository struct {
	db *gorm.DB
}

// NewAPIConfigRepository 创建API配置仓储
func NewAPIConfigRepository() APIConfigRepository {
	return &apiConfigRepository{
		db: database.GetDB(),
	}
}

// List 获取API配置列表
func (r *apiConfigRepository) List(ctx context.Context, page, pageSize int) ([]model.APIConfig, int64, error) {
	var configs []model.APIConfig
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.APIConfig{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Order("weight DESC, id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&configs).Error

	return configs, total, err
}

// GetByID 根据ID获取API配置
func (r *apiConfigRepository) GetByID(ctx context.Context, id int) (*model.APIConfig, error) {
	var config model.APIConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Create 创建API配置
func (r *apiConfigRepository) Create(ctx context.Context, config *model.APIConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

// Update 更新API配置
func (r *apiConfigRepository) Update(ctx context.Context, config *model.APIConfig) error {
	return r.db.WithContext(ctx).Model(config).Where("id = ?", config.ID).Updates(config).Error
}

// Delete 删除API配置
func (r *apiConfigRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.APIConfig{}).Error
}

// BatchDelete 批量删除API配置
func (r *apiConfigRepository) BatchDelete(ctx context.Context, ids []int) error {
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.APIConfig{}).Error
}

// UpdateStatus 更新API配置状态
func (r *apiConfigRepository) UpdateStatus(ctx context.Context, id int, status int) error {
	return r.db.WithContext(ctx).Model(&model.APIConfig{}).
		Where("id = ?", id).
		Update("status", status).Error
}