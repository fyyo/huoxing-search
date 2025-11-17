package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/repository"
)

// SourceHandler 资源处理器
type SourceHandler struct {
	sourceRepo repository.SourceRepository
}

// NewSourceHandler 创建资源处理器
func NewSourceHandler(cfg *config.Config) *SourceHandler {
	return &SourceHandler{
		sourceRepo: repository.NewSourceRepository(),
	}
}

// List 获取资源列表（支持搜索）
func (h *SourceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	isType, _ := strconv.Atoi(c.DefaultQuery("is_type", "-1"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "1"))
	keyword := c.Query("keyword")

	var sources []*model.Source
	var total int64
	var err error

	// 如果有关键词，使用搜索
	if keyword != "" {
		sources, total, err = h.sourceRepo.Search(c.Request.Context(), keyword, page, pageSize)
	} else {
		sources, total, err = h.sourceRepo.List(c.Request.Context(), page, pageSize, isType, status)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("获取资源列表失败"))
		return
	}

	c.JSON(http.StatusOK, model.PageData(total, page, pageSize, sources))
}

// GetByID 根据ID获取资源
func (h *SourceHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("无效的资源ID"))
		return
	}

	source, err := h.sourceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NotFound("资源不存在"))
		return
	}

	c.JSON(http.StatusOK, model.Success(source))
}

// Create 创建资源
func (h *SourceHandler) Create(c *gin.Context) {
	var source model.Source
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	if err := h.sourceRepo.Create(c.Request.Context(), &source); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("创建资源失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(source))
}

// Update 更新资源
func (h *SourceHandler) Update(c *gin.Context) {
	var source model.Source
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	if source.SourceID == 0 {
		c.JSON(http.StatusBadRequest, model.BadRequest("资源ID不能为空"))
		return
	}

	if err := h.sourceRepo.Update(c.Request.Context(), &source); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("更新资源失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(source))
}

// Delete 批量删除资源
func (h *SourceHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	// TODO: 实际批量删除逻辑
	for _, id := range req.IDs {
		if err := h.sourceRepo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, model.ServerError("删除资源失败"))
			return
		}
	}

	c.JSON(http.StatusOK, model.SuccessWithMessage("删除成功", nil))
}