package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"xinyue-go/internal/model"
	"xinyue-go/internal/service"
)

// SearchHandler 搜索API处理器
type SearchHandler struct {
	searchService *service.SearchService
}

// NewSearchHandler 创建搜索处理器实例
func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

// Search 搜索接口
// @Summary 搜索资源
// @Description 根据关键词搜索网盘资源
// @Tags 搜索
// @Accept json
// @Produce json
// @Param request body model.SearchRequest true "搜索请求"
// @Success 200 {object} model.Response{data=model.SearchResponse}
// @Router /api/search [post]
func (h *SearchHandler) Search(c *gin.Context) {
	var req model.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	// 调用搜索服务
	result, err := h.searchService.Search(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: "搜索失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: result.Message,
		Data:    result,
	})
}

// ClearCache 清除搜索缓存
// @Summary 清除搜索缓存
// @Description 清除指定关键词的搜索缓存
// @Tags 搜索
// @Accept json
// @Produce json
// @Param keyword query string false "关键词"
// @Param pan_type query int false "网盘类型"
// @Success 200 {object} model.Response
// @Router /api/search/cache [delete]
func (h *SearchHandler) ClearCache(c *gin.Context) {
	keyword := c.Query("keyword")
	panType := 0
	if pt := c.Query("pan_type"); pt != "" {
		if _, err := c.GetQuery("pan_type"); err {
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    400,
				Message: "网盘类型参数错误",
			})
			return
		}
	}

	var err error
	if keyword == "" {
		// 清除所有缓存
		err = h.searchService.ClearAllCache(c.Request.Context())
	} else {
		// 清除指定关键词的缓存
		err = h.searchService.ClearCache(c.Request.Context(), keyword, panType)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: "清除缓存失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "缓存清除成功",
	})
}