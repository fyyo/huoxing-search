package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"huoxing-search/internal/model"
	"huoxing-search/internal/pkg/config"
	"huoxing-search/internal/pkg/logger"
	"huoxing-search/internal/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(cfg),
	}
}

// Login 登录接口
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误: "+err.Error()))
		return
	}

	logger.Info("收到登录请求", zap.String("username", req.Username))

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		logger.Error("登录失败", 
			zap.String("username", req.Username),
			zap.Error(err),
		)
		
		// 根据错误类型返回不同的消息
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusUnauthorized, model.Unauthorized("用户不存在"))
		case service.ErrPasswordIncorrect:
			c.JSON(http.StatusUnauthorized, model.Unauthorized("密码错误"))
		case service.ErrUserDisabled:
			c.JSON(http.StatusForbidden, model.Forbidden("用户已被禁用"))
		default:
			c.JSON(http.StatusInternalServerError, model.ServerError("登录失败"))
		}
		return
	}

	c.JSON(http.StatusOK, model.Success(resp))
}

// Register 注册接口
func (h *AuthHandler) Register(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误: "+err.Error()))
		return
	}

	// 验证必填字段
	if user.Username == "" || user.Password == "" {
		c.JSON(http.StatusBadRequest, model.BadRequest("用户名和密码不能为空"))
		return
	}

	// 默认启用
	if user.Status == 0 {
		user.Status = 1
	}

	logger.Info("收到注册请求", zap.String("username", user.Username))

	if err := h.authService.Register(c.Request.Context(), &user); err != nil {
		logger.Error("注册失败",
			zap.String("username", user.Username),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, model.ServerError("注册失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessWithMessage("注册成功", gin.H{
		"admin_id": user.AdminID,
		"username": user.Username,
	}))
}

// RefreshToken 刷新token接口
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, model.Unauthorized("未提供token"))
		return
	}

	// 移除 "Bearer " 前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	newToken, err := h.authService.RefreshToken(c.Request.Context(), token)
	if err != nil {
		logger.Error("刷新token失败", zap.Error(err))
		c.JSON(http.StatusUnauthorized, model.Unauthorized("token刷新失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"token": newToken,
	}))
}

// GetUserInfo 获取当前用户信息
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.Unauthorized("未登录"))
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, model.Success(gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
	}))
}

// Logout 登出接口
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: 可以将token加入黑名单
	c.JSON(http.StatusOK, model.SuccessWithMessage("登出成功", nil))
}