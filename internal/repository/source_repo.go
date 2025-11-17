package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/database"
)

// SourceRepository 资源仓储接口
type SourceRepository interface {
	Create(ctx context.Context, source *model.Source) error
	Update(ctx context.Context, source *model.Source) error
	Delete(ctx context.Context, sourceID uint64) error
	GetByID(ctx context.Context, sourceID uint64) (*model.Source, error)
	GetByURL(ctx context.Context, url string) (*model.Source, error)
	List(ctx context.Context, page, pageSize int, isType int, status int) ([]*model.Source, int64, error)
	Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.Source, int64, error)
	SearchByKeywordAndType(ctx context.Context, keyword string, panType int, limit int) ([]*model.Source, error)
	BatchCreate(ctx context.Context, sources []*model.Source) error
	DeleteExpiredTemp(ctx context.Context, expiryTime int64) (int64, error)
}

type sourceRepository struct {
	db *gorm.DB
}

// NewSourceRepository 创建资源仓储
func NewSourceRepository() SourceRepository {
	return &sourceRepository{
		db: database.GetDB(),
	}
}

// Create 创建资源
func (r *sourceRepository) Create(ctx context.Context, source *model.Source) error {
	return r.db.WithContext(ctx).Create(source).Error
}

// Update 更新资源
func (r *sourceRepository) Update(ctx context.Context, source *model.Source) error {
	return r.db.WithContext(ctx).Save(source).Error
}

// Delete 删除资源
func (r *sourceRepository) Delete(ctx context.Context, sourceID uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Source{}, sourceID).Error
}

// GetByID 根据ID获取资源
func (r *sourceRepository) GetByID(ctx context.Context, sourceID uint64) (*model.Source, error) {
	var source model.Source
	err := r.db.WithContext(ctx).Where("source_id = ?", sourceID).First(&source).Error
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// GetByURL 根据URL获取资源
func (r *sourceRepository) GetByURL(ctx context.Context, url string) (*model.Source, error) {
	var source model.Source
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&source).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &source, nil
}

// List 获取资源列表
func (r *sourceRepository) List(ctx context.Context, page, pageSize int, isType int, status int) ([]*model.Source, int64, error) {
	var sources []*model.Source
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Source{})

	// 筛选条件
	if isType >= 0 {
		query = query.Where("is_type = ?", isType)
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("create_time DESC").Offset(offset).Limit(pageSize).Find(&sources).Error
	if err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

// Search 搜索资源
func (r *sourceRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.Source, int64, error) {
	var sources []*model.Source
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Source{}).Where("status = 1")

	// 搜索条件
	if keyword != "" {
		searchPattern := fmt.Sprintf("%%%s%%", keyword)
		query = query.Where("title LIKE ?", searchPattern)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("create_time DESC").Offset(offset).Limit(pageSize).Find(&sources).Error
	if err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

// BatchCreate 批量创建资源
func (r *sourceRepository) BatchCreate(ctx context.Context, sources []*model.Source) error {
	if len(sources) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(sources, 100).Error
}

// SearchByKeywordAndType 按关键词和网盘类型搜索本地资源
func (r *sourceRepository) SearchByKeywordAndType(ctx context.Context, keyword string, panType int, limit int) ([]*model.Source, error) {
	var sources []*model.Source
	
	query := r.db.WithContext(ctx).Model(&model.Source{}).Where("status = 1")
	
	// 网盘类型筛选
	if panType >= 0 {
		query = query.Where("is_type = ?", panType)
	}
	
	// 关键词搜索
	if keyword != "" {
		searchPattern := fmt.Sprintf("%%%s%%", keyword)
		query = query.Where("title LIKE ?", searchPattern)
	}
	
	// 按创建时间倒序,限制数量
	err := query.Order("create_time DESC").Limit(limit).Find(&sources).Error
	if err != nil {
		return nil, err
	}
	
	return sources, nil
}

// DeleteExpiredTemp 删除过期的临时资源
// expiryTime: 过期时间戳，早于此时间的临时资源将被删除
func (r *sourceRepository) DeleteExpiredTemp(ctx context.Context, expiryTime int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("is_time = ? AND create_time < ?", 1, expiryTime).
		Delete(&model.Source{})
	
	if result.Error != nil {
		return 0, result.Error
	}
	
	return result.RowsAffected, nil
}