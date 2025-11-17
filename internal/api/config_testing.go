package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"xinyue-go/internal/netdisk"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/repository"
)

// ConfigTestHandler 配置测试处理器
type ConfigTestHandler struct {
	cfg        *config.Config
	configRepo repository.ConfigRepository
}

// NewConfigTestHandler 创建配置测试处理器
func NewConfigTestHandler(cfg *config.Config) *ConfigTestHandler {
	return &ConfigTestHandler{
		cfg:        cfg,
		configRepo: repository.NewConfigRepository(),
	}
}

// TestNetdiskConnection 测试网盘连接
// POST /api/admin/test/netdisk
// Body: {"netdisk": "quark|baidu|aliyun|uc|xunlei"}
func (h *ConfigTestHandler) TestNetdiskConnection(c *gin.Context) {
	var req struct {
		Netdisk string `json:"netdisk" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 映射网盘名称到PanType
	panTypeMap := map[string]int{
		"quark":  0,
		"baidu":  2,
		"aliyun": 3,
		"uc":     4,
		"xunlei": 5,
	}

	panType, ok := panTypeMap[req.Netdisk]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的网盘类型: " + req.Netdisk,
		})
		return
	}

	// 创建网盘管理器
	manager := netdisk.NewNetdiskManager(h.cfg)

	// 获取网盘客户端
	client, err := manager.GetClient(panType)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "获取网盘客户端失败: " + err.Error(),
			"success": false,
		})
		return
	}

	// 检查是否已配置
	if !client.IsConfigured() {
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": "网盘未配置，请先配置Cookie/Token等信息",
			"success": false,
		})
		return
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": fmt.Sprintf("%s网盘连接测试失败: %s", client.GetName(), err.Error()),
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": fmt.Sprintf("%s网盘连接测试成功！", client.GetName()),
		"success": true,
	})
}

// TestWechatConnection 测试微信配置连接
// POST /api/admin/test/wechat
// Body: {"type": "chatbot|official"}
func (h *ConfigTestHandler) TestWechatConnection(c *gin.Context) {
	var req struct {
		Type string `json:"type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	switch req.Type {
	case "chatbot":
		// 测试对话开放平台配置
		appID, _ := h.configRepo.Get(ctx, "wechat_chatbot_app_id")
		token, _ := h.configRepo.Get(ctx, "wechat_chatbot_token")
		encodingAESKey, _ := h.configRepo.Get(ctx, "wechat_chatbot_encoding_aes_key")

		if appID == "" || token == "" || encodingAESKey == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    400,
				"message": "微信对话开放平台配置不完整，请检查AppID、Token和EncodingAESKey",
				"success": false,
			})
			return
		}

		// 验证配置格式
		if len(encodingAESKey) != 43 {
			c.JSON(http.StatusOK, gin.H{
				"code":    400,
				"message": "EncodingAESKey格式错误，应为43位字符",
				"success": false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "微信对话开放平台配置验证成功！请在微信后台测试实际回调",
			"success": true,
			"data": gin.H{
				"app_id":  appID,
				"token":   token[:4] + "****",
				"aes_key": encodingAESKey[:4] + "****",
				"callback_url_format": "https://yourdomain.com/api/wechat/chatbot/callback",
			},
		})

	case "official":
		// 测试公众号配置
		token, _ := h.configRepo.Get(ctx, "wechat_official_token")

		if token == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    400,
				"message": "微信公众号Token未配置",
				"success": false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "微信公众号配置验证成功！请在微信后台填写回调URL进行验证",
			"success": true,
			"data": gin.H{
				"token":              token[:4] + "****",
				"callback_url_format": "https://yourdomain.com/api/wechat/official/callback",
			},
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的类型: " + req.Type,
		})
	}
}