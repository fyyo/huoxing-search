package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/repository"
)

// UserHandler 用户处理器
type UserHandler struct {
	userRepo repository.UserRepository
}

// NewUserHandler 创建用户处理器
func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{
		userRepo: repository.NewUserRepository(),
	}
}

// List 获取用户列表
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total, err := h.userRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("获取用户列表失败"))
		return
	}

	c.JSON(http.StatusOK, model.PageData(total, page, pageSize, users))
}

// GetByID 根据ID获取用户
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("无效的用户ID"))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NotFound("用户不存在"))
		return
	}

	c.JSON(http.StatusOK, model.Success(user))
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	// TODO: 密码加密
	if err := h.userRepo.Create(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("创建用户失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(user))
}

// Update 更新用户
func (h *UserHandler) Update(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	if user.AdminID == 0 {
		c.JSON(http.StatusBadRequest, model.BadRequest("用户ID不能为空"))
		return
	}

	if err := h.userRepo.Update(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("更新用户失败"))
		return
	}

	c.JSON(http.StatusOK, model.Success(user))
}

// Delete 批量删除用户
func (h *UserHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	// TODO: 实际批量删除逻辑
	for _, id := range req.IDs {
		if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, model.ServerError("删除用户失败"))
			return
		}
	}

	c.JSON(http.StatusOK, model.SuccessWithMessage("删除成功", nil))
}

// ResetPassword 重置用户密码
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req struct {
		UserID   uint64 `json:"user_id" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequest("参数错误"))
		return
	}

	// TODO: 实际更新密码逻辑（需要加密）
	// 这里简化处理
	user := &model.User{
		AdminID:  uint(req.UserID),
		Password: req.Password, // 实际应该加密
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerError("重置密码失败"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessWithMessage("密码重置成功", nil))
}