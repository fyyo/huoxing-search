package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"huoxing-search/internal/model"
	"huoxing-search/internal/repository"
)

// AdminManagementHandler 管理员管理处理器
type AdminManagementHandler struct {
	repo repository.AdminRepository
}

// NewAdminManagementHandler 创建管理员管理处理器
func NewAdminManagementHandler() *AdminManagementHandler {
	return &AdminManagementHandler{
		repo: repository.NewAdminRepository(),
	}
}

// List 获取管理员列表
func (h *AdminManagementHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	admins, total, err := h.repo.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取管理员列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"items": admins,
			"total": total,
			"page":  page,
			"size":  pageSize,
		},
	})
}

// GetByID 根据ID获取管理员
func (h *AdminManagementHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	admin, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "管理员不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    admin,
	})
}

// Create 创建管理员
func (h *AdminManagementHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Mobile   string `json:"mobile"`
		Status   int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 检查用户名是否已存在
	existingAdmin, _ := h.repo.GetByUsername(c.Request.Context(), req.Username)
	if existingAdmin != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户名已存在",
		})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "密码加密失败",
		})
		return
	}

	admin := &model.Admin{
		Username: req.Username,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Email:    req.Email,
		Mobile:   req.Mobile,
		Status:   req.Status,
	}

	if err := h.repo.Create(c.Request.Context(), admin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建管理员失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    admin,
	})
}

// Update 更新管理员
func (h *AdminManagementHandler) Update(c *gin.Context) {
	var req struct {
		AdminID  uint   `json:"admin_id" binding:"required"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Mobile   string `json:"mobile"`
		Status   int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 获取现有管理员
	admin, err := h.repo.GetByID(c.Request.Context(), req.AdminID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "管理员不存在",
		})
		return
	}

	// 更新字段
	if req.Username != "" {
		admin.Username = req.Username
	}
	if req.Nickname != "" {
		admin.Nickname = req.Nickname
	}
	if req.Email != "" {
		admin.Email = req.Email
	}
	if req.Mobile != "" {
		admin.Mobile = req.Mobile
	}
	admin.Status = req.Status

	if err := h.repo.Update(c.Request.Context(), admin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新管理员失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    admin,
	})
}

// Delete 删除管理员
func (h *AdminManagementHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	for _, id := range req.IDs {
		if err := h.repo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除管理员失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// ResetPassword 重置密码
func (h *AdminManagementHandler) ResetPassword(c *gin.Context) {
	var req struct {
		AdminID     uint   `json:"admin_id" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "密码加密失败",
		})
		return
	}

	if err := h.repo.UpdatePassword(c.Request.Context(), req.AdminID, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "重置密码失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "重置密码成功",
	})
}

// UpdateStatus 更新状态
func (h *AdminManagementHandler) UpdateStatus(c *gin.Context) {
	var req struct {
		AdminID uint `json:"admin_id" binding:"required"`
		Status  int  `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), req.AdminID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新状态成功",
	})
}