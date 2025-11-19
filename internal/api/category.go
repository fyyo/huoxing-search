package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"huoxing-search/internal/model"
	"huoxing-search/internal/repository"
)

// CategoryHandler 分类处理器
type CategoryHandler struct {
	categoryRepo repository.CategoryRepository
}

// NewCategoryHandler 创建分类处理器
func NewCategoryHandler() *CategoryHandler {
	return &CategoryHandler{
		categoryRepo: repository.NewCategoryRepository(),
	}
}

// List 获取分类列表
func (h *CategoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	isType, _ := strconv.Atoi(c.DefaultQuery("is_type", "-1"))

	categories, total, err := h.categoryRepo.List(c.Request.Context(), page, pageSize, isType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("获取分类列表失败"))
		return
	}

	c.JSON(http.StatusOK, model.PageData(total, page, pageSize, categories))
}

// GetByID 根据ID获取分类
func (h *CategoryHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("无效的分类ID"))
		return
	}

	category, err := h.categoryRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NotFound("分类不存在"))
		return
	}

	c.JSON(http.StatusOK, model.Success(category))
}

// Create 创建分类
func (h *CategoryHandler) Create(c *gin.Context) {
	var category model.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	// 验证必填字段
	if category.Name == "" {
		c.JSON(http.StatusBadRequest, model.BadRequest("名称不能为空"))
		return
	}

	if err := h.categoryRepo.Create(c.Request.Context(), &category); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("创建分类失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(category))
}

// Update 更新分类
func (h *CategoryHandler) Update(c *gin.Context) {
	var category model.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	if category.CategoryID == 0 {
		c.JSON(http.StatusBadRequest, model.BadRequest("ID不能为空"))
		return
	}

	if err := h.categoryRepo.Update(c.Request.Context(), &category); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("更新分类失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(category))
}

// Delete 删除分类
func (h *CategoryHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	if err := h.categoryRepo.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("删除分类失败"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessWithMessage("删除成功", nil))
}