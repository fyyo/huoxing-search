package repository

import (
	"context"

	"gorm.io/gorm"
	"huoxing-search/internal/model"
	"huoxing-search/internal/pkg/database"
)

// CategoryRepository 分类仓储接口
type CategoryRepository interface {
	Create(ctx context.Context, category *model.Category) error
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.Category, error)
	List(ctx context.Context, page, pageSize int, isType int) ([]*model.Category, int64, error)
	BatchDelete(ctx context.Context, ids []int) error
}

type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository 创建分类仓储
func NewCategoryRepository() CategoryRepository {
	return &categoryRepository{
		db: database.GetDB(),
	}
}

// Create 创建分类
func (r *categoryRepository) Create(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// Update 更新分类
func (r *categoryRepository) Update(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// Delete 删除分类
func (r *categoryRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&model.Category{}, id).Error
}

// GetByID 根据ID获取分类
func (r *categoryRepository) GetByID(ctx context.Context, id int) (*model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).Where("category_id = ?", id).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// List 获取分类列表
func (r *categoryRepository) List(ctx context.Context, page, pageSize int, isType int) ([]*model.Category, int64, error) {
	var categories []*model.Category
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Category{})

	// 筛选条件
	if isType >= 0 {
		query = query.Where("is_type = ?", isType)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("sort ASC, create_time DESC").Offset(offset).Limit(pageSize).Find(&categories).Error
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// BatchDelete 批量删除分类
func (r *categoryRepository) BatchDelete(ctx context.Context, ids []int) error {
	return r.db.WithContext(ctx).Where("category_id IN ?", ids).Delete(&model.Category{}).Error
}