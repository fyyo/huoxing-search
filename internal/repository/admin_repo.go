package repository

import (
	"context"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/database"

	"gorm.io/gorm"
)

// AdminRepository 管理员仓储接口
type AdminRepository interface {
	Create(ctx context.Context, admin *model.Admin) error
	Update(ctx context.Context, admin *model.Admin) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.Admin, error)
	GetByUsername(ctx context.Context, username string) (*model.Admin, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]*model.Admin, int64, error)
	UpdatePassword(ctx context.Context, id uint, password string) error
	UpdateStatus(ctx context.Context, id uint, status int) error
	UpdateLoginInfo(ctx context.Context, id uint, ip string) error
}

type adminRepository struct {
	db *gorm.DB
}

// NewAdminRepository 创建管理员仓储实例
func NewAdminRepository() AdminRepository {
	return &adminRepository{
		db: database.GetDB(),
	}
}

// Create 创建管理员
func (r *adminRepository) Create(ctx context.Context, admin *model.Admin) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

// Update 更新管理员
func (r *adminRepository) Update(ctx context.Context, admin *model.Admin) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

// Delete 删除管理员
func (r *adminRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Admin{}, id).Error
}

// GetByID 根据ID获取管理员
func (r *adminRepository) GetByID(ctx context.Context, id uint) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.WithContext(ctx).Where("admin_id = ?", id).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByUsername 根据用户名获取管理员
func (r *adminRepository) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// List 获取管理员列表
func (r *adminRepository) List(ctx context.Context, page, pageSize int, keyword string) ([]*model.Admin, int64, error) {
	var admins []*model.Admin
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Admin{})

	// 关键词搜索
	if keyword != "" {
		db = db.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("admin_id DESC").Find(&admins).Error; err != nil {
		return nil, 0, err
	}

	return admins, total, nil
}

// UpdatePassword 更新密码
func (r *adminRepository) UpdatePassword(ctx context.Context, id uint, password string) error {
	return r.db.WithContext(ctx).Model(&model.Admin{}).
		Where("admin_id = ?", id).
		Update("password", password).Error
}

// UpdateStatus 更新状态
func (r *adminRepository) UpdateStatus(ctx context.Context, id uint, status int) error {
	return r.db.WithContext(ctx).Model(&model.Admin{}).
		Where("admin_id = ?", id).
		Update("status", status).Error
}

// UpdateLoginInfo 更新登录信息
func (r *adminRepository) UpdateLoginInfo(ctx context.Context, id uint, ip string) error {
	return r.db.WithContext(ctx).Model(&model.Admin{}).
		Where("admin_id = ?", id).
		Updates(map[string]interface{}{
			"last_login_time": gorm.Expr("UNIX_TIMESTAMP()"),
			"last_login_ip":   ip,
		}).Error
}