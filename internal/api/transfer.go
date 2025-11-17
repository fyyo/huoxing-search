package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/pkg/logger"
	"xinyue-go/internal/service"
)

// TransferHandler 转存处理器
type TransferHandler struct {
	transferService service.TransferService
}

// NewTransferHandler 创建转存处理器
func NewTransferHandler(cfg *config.Config) *TransferHandler {
	return &TransferHandler{
		transferService: service.NewTransferService(cfg),
	}
}

// Transfer 转存接口
func (h *TransferHandler) Transfer(c *gin.Context) {
	var req model.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误: "+err.Error()))
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, model.BadRequest("转存项不能为空"))
		return
	}

	logger.Info("收到转存请求",
		zap.Int("items_count", len(req.Items)),
		zap.Int("pan_type", req.PanType),
		zap.Int("max_count", req.MaxCount),
	)

	resp, err := h.transferService.BatchTransfer(c.Request.Context(), &req)
	if err != nil {
		logger.Error("转存失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ServerError("转存失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(resp))
}

// TransferAndSave 转存并保存接口
func (h *TransferHandler) TransferAndSave(c *gin.Context) {
	var req model.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误: "+err.Error()))
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, model.BadRequest("转存项不能为空"))
		return
	}

	logger.Info("收到转存并保存请求",
		zap.Int("items_count", len(req.Items)),
		zap.Int("pan_type", req.PanType),
		zap.Int("max_count", req.MaxCount),
	)

	resp, err := h.transferService.TransferAndSave(c.Request.Context(), &req)
	if err != nil {
		logger.Error("转存并保存失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ServerError("转存并保存失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(resp))
}