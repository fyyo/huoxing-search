package api

import (
	"net/http"
	"strconv"
	"time"
	"huoxing-search/internal/model"
	"huoxing-search/internal/repository"

	"github.com/gin-gonic/gin"
)

// SystemConfigHandler 系统配置处理器
type SystemConfigHandler struct {
	repo repository.ConfigRepository
}

// NewSystemConfigHandler 创建系统配置处理器
func NewSystemConfigHandler() *SystemConfigHandler {
	return &SystemConfigHandler{
		repo: repository.NewConfigRepository(),
	}
}

// List 获取配置列表
func (h *SystemConfigHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.repo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// GetByID 根据ID获取配置
func (h *SystemConfigHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	config, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    config,
	})
}

// Create 创建配置
func (h *SystemConfigHandler) Create(c *gin.Context) {
	var req model.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 验证必填字段
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "配置名称不能为空",
		})
		return
	}

	if err := h.repo.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    req,
	})
}

// Update 更新配置
func (h *SystemConfigHandler) Update(c *gin.Context) {
	var req model.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 支持通过name或conf_id更新
	if req.ConfID == 0 && req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "ID或Name不能同时为空",
		})
		return
	}

	// 如果没有提供ID，通过name查找
	if req.ConfID == 0 && req.Name != "" {
		existing, err := h.repo.GetByName(c.Request.Context(), req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "配置不存在: " + err.Error(),
			})
			return
		}
		req.ConfID = existing.ConfID
	}

	// 设置更新时间
	req.UpdateTime = time.Now().Unix()

	if err := h.repo.Update(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    req,
	})
}

// BatchUpdate 批量更新配置
func (h *SystemConfigHandler) BatchUpdate(c *gin.Context) {
	var req struct {
		Configs []model.Config `json:"configs" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 设置更新时间
	now := time.Now().Unix()
	for i := range req.Configs {
		req.Configs[i].UpdateTime = now
	}

	if err := h.repo.BatchUpdate(c.Request.Context(), req.Configs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "批量更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
	})
}

// BatchUpsert 批量插入或更新配置（根据name）
func (h *SystemConfigHandler) BatchUpsert(c *gin.Context) {
	var req map[string]interface{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 调试日志：打印接收到的原始数据
	println("📥 [BatchUpsert] 接收到的数据:")
	for key, value := range req {
		println("  ", key, "=", value, "(类型:", getTypeName(value), ")")
	}

	if len(req) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "配置不能为空",
		})
		return
	}

	// 转换为 map[string]string
	strMap := make(map[string]string)

	// 检查是否是数组格式: {"configs": [{"name": "xx", "value": "yy"}, ...]}
	if configsArray, ok := req["configs"].([]interface{}); ok {
		// 数组格式（微信配置页面使用）
		println("📦 [BatchUpsert] 检测到数组格式，开始解析...")
		for i, item := range configsArray {
			if configMap, ok := item.(map[string]interface{}); ok {
				name, nameOk := configMap["name"].(string)
				value, valueOk := configMap["value"].(string)
				
				if nameOk && name != "" {
					if !valueOk {
						value = "" // value不是字符串时设为空
					}
					strMap[name] = value
					println("  ✅ [", i, "]", name, "=", value)
				}
			}
		}
	} else {
		// 直接键值对格式: {"name1": "value1", "name2": "value2"}
		println("📦 [BatchUpsert] 检测到键值对格式，开始解析...")
		for key, value := range req {
			// 跳过 configs 键（如果存在）
			if key == "configs" {
				continue
			}
			
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			case []interface{}:
				if len(v) > 0 {
					if str, ok := v[0].(string); ok {
						strValue = str
					}
				}
			case nil:
				strValue = ""
			default:
				strValue = ""
			}

			if key != "" {
				strMap[key] = strValue
				println("  ✅", key, "=", strValue)
			}
		}
	}

	println("📊 [BatchUpsert] 转换后的配置数量:", len(strMap))

	if len(strMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "没有有效的配置数据，请检查前端数据格式",
		})
		return
	}

	if err := h.repo.BatchUpsert(c.Request.Context(), strMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "批量保存失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "保存成功",
	})
}

// getTypeName 获取值的类型名称（用于调试）
func getTypeName(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch v.(type) {
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// Delete 删除配置
func (h *SystemConfigHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if err := h.repo.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// GetByName 根据名称获取配置
func (h *SystemConfigHandler) GetByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "配置名称不能为空",
		})
		return
	}

	config, err := h.repo.GetByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    config,
	})
}