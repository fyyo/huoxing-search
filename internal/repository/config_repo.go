package repository

import (
	"context"
	"fmt"
	"time"
	"huoxing-search/internal/model"
	"huoxing-search/internal/pkg/database"

	"gorm.io/gorm"
)

// ConfigRepository 配置仓储接口
type ConfigRepository interface {
	List(ctx context.Context, page, pageSize int) ([]model.Config, int64, error)
	GetByID(ctx context.Context, id int) (*model.Config, error)
	GetByName(ctx context.Context, name string) (*model.Config, error)
	GetByNames(ctx context.Context, names []string) (map[string]string, error)
	Get(ctx context.Context, name string) (string, error)
	GetInt(ctx context.Context, name string) (int, error)
	Create(ctx context.Context, config *model.Config) error
	Update(ctx context.Context, config *model.Config) error
	Delete(ctx context.Context, id int) error
	BatchDelete(ctx context.Context, ids []int) error
	BatchUpdate(ctx context.Context, configs []model.Config) error
	BatchUpsert(ctx context.Context, configs map[string]string) error
}

type configRepository struct {
	db *gorm.DB
}

// NewConfigRepository 创建配置仓储
func NewConfigRepository() ConfigRepository {
	return &configRepository{
		db: database.GetDB(),
	}
}

// List 获取配置列表
func (r *configRepository) List(ctx context.Context, page, pageSize int) ([]model.Config, int64, error) {
	var configs []model.Config
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.Config{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Order("`group` ASC, sort ASC, conf_id ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&configs).Error

	return configs, total, err
}

// GetByID 根据ID获取配置
func (r *configRepository) GetByID(ctx context.Context, id int) (*model.Config, error) {
	var config model.Config
	err := r.db.WithContext(ctx).Where("conf_id = ?", id).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByName 根据名称获取配置
func (r *configRepository) GetByName(ctx context.Context, name string) (*model.Config, error) {
	var config model.Config
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByNames 根据名称批量获取配置
func (r *configRepository) GetByNames(ctx context.Context, names []string) (map[string]string, error) {
	var configs []model.Config
	err := r.db.WithContext(ctx).Where("name IN ?", names).Find(&configs).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.Name] = config.Value
	}
	return result, nil
}

// Get 根据名称获取配置值
func (r *configRepository) Get(ctx context.Context, name string) (string, error) {
	config, err := r.GetByName(ctx, name)
	if err != nil {
		return "", err
	}
	return config.Value, nil
}

// GetInt 根据名称获取配置值(整数)
func (r *configRepository) GetInt(ctx context.Context, name string) (int, error) {
	value, err := r.Get(ctx, name)
	if err != nil {
		return 0, err
	}
	
	var intValue int
	_, err = fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		return 0, fmt.Errorf("配置值 %s 不是有效的整数: %w", name, err)
	}
	return intValue, nil
}

// Create 创建配置
func (r *configRepository) Create(ctx context.Context, config *model.Config) error {
	return r.db.WithContext(ctx).Create(config).Error
}

// Update 更新配置
func (r *configRepository) Update(ctx context.Context, config *model.Config) error {
	return r.db.WithContext(ctx).Model(config).Where("conf_id = ?", config.ConfID).Updates(config).Error
}

// Delete 删除配置
func (r *configRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Where("conf_id = ?", id).Delete(&model.Config{}).Error
}

// BatchDelete 批量删除配置
func (r *configRepository) BatchDelete(ctx context.Context, ids []int) error {
	return r.db.WithContext(ctx).Where("conf_id IN ?", ids).Delete(&model.Config{}).Error
}

// BatchUpdate 批量更新配置
func (r *configRepository) BatchUpdate(ctx context.Context, configs []model.Config) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, config := range configs {
			if err := tx.Model(&model.Config{}).
				Where("conf_id = ?", config.ConfID).
				Updates(map[string]interface{}{
					"value":       config.Value,
					"update_time": config.UpdateTime,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchUpsert 批量插入或更新配置（根据name）
func (r *configRepository) BatchUpsert(ctx context.Context, configs map[string]string) error {
	if len(configs) == 0 {
		return fmt.Errorf("配置列表为空")
	}
	
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		createCount := 0
		updateCount := 0
		
		for name, value := range configs {
			// 先查询是否存在
			var existing model.Config
			err := tx.Where("name = ?", name).First(&existing).Error
			
			if err == gorm.ErrRecordNotFound {
				// 不存在则创建，设置必要的默认值
				newConfig := model.Config{
					Name:        name,
					Value:       value,
					Title:       name,  // 标题默认使用name
					Description: "",    // 描述为空
					Group:       0,     // 默认分组：基本配置
					Type:        1,     // 默认类型：文本输入
					Options:     "",    // 选项为空
					Sort:        0,     // 默认排序
					Status:      1,     // 默认启用
					CreateTime:  now,
					UpdateTime:  now,
				}
				
				if err := tx.Create(&newConfig).Error; err != nil {
					return fmt.Errorf("创建配置 %s 失败: %w", name, err)
				}
				createCount++
				fmt.Printf("✅ [BatchUpsert] 创建配置: %s = %s (ID=%d)\n", name, value, newConfig.ConfID)
			} else if err != nil {
				return fmt.Errorf("查询配置 %s 失败: %w", name, err)
			} else {
				// 存在则更新
				result := tx.Model(&model.Config{}).
					Where("name = ?", name).
					Updates(map[string]interface{}{
						"value":       value,
						"update_time": now,
					})
				
				if result.Error != nil {
					return fmt.Errorf("更新配置 %s 失败: %w", name, result.Error)
				}
				updateCount++
				fmt.Printf("✅ [BatchUpsert] 更新配置: %s = %s (影响行数=%d)\n", name, value, result.RowsAffected)
			}
		}
		
		fmt.Printf("📊 [BatchUpsert] 完成: 创建=%d, 更新=%d, 总数=%d\n", createCount, updateCount, len(configs))
		return nil
	})
}