package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/pkg/jwt"
	"xinyue-go/internal/pkg/logger"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	expiration := cfg.JWT.Expiration
	if expiration == 0 {
		expiration = cfg.JWT.ExpireHours
	}
	jwtService := jwt.NewJWTService(cfg.JWT.Secret, expiration)

	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.Unauthorized("未提供认证token"))
			c.Abort()
			return
		}

		// 验证格式: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, model.Unauthorized("token格式错误"))
			c.Abort()
			return
		}

		token := parts[1]

		// 验证token
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			// 安全地截取token用于日志
			tokenPreview := token
			if len(token) > 20 {
				tokenPreview = token[:20] + "..."
			}
			logger.Warn("token验证失败",
				zap.Error(err),
				zap.String("token", tokenPreview),
			)
			c.JSON(http.StatusUnauthorized, model.Unauthorized("token无效或已过期"))
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件 (必须在AuthMiddleware之后使用)
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.Unauthorized("未登录"))
			c.Abort()
			return
		}

		// role = 0 表示管理员
		if roleInt, ok := role.(int); !ok || roleInt != 0 {
			logger.Warn("非管理员尝试访问管理接口",
				zap.Any("user_id", c.GetInt64("user_id")),
				zap.Any("role", role),
			)
			c.JSON(http.StatusForbidden, model.Forbidden("需要管理员权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件 (如果有token则验证,没有则跳过)
func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	expiration := cfg.JWT.Expiration
	if expiration == 0 {
		expiration = cfg.JWT.ExpireHours
	}
	jwtService := jwt.NewJWTService(cfg.JWT.Secret, expiration)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		claims, err := jwtService.ValidateToken(token)
		if err == nil {
			// token有效,存入上下文
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
		}

		c.Next()
	}
}