package service

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/config"
	"xinyue-go/internal/pkg/jwt"
	"xinyue-go/internal/pkg/logger"
	"xinyue-go/internal/repository"
)

var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrPasswordIncorrect = errors.New("密码错误")
	ErrUserDisabled      = errors.New("用户已被禁用")
)

// AuthService 认证服务接口
type AuthService interface {
	Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error)
	Register(ctx context.Context, user *model.User) error
	RefreshToken(ctx context.Context, token string) (string, error)
	GetUserByToken(ctx context.Context, token string) (*model.User, error)
}

type authService struct {
	userRepo   repository.UserRepository
	jwtService *jwt.JWTService
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config) AuthService {
	expiration := cfg.JWT.Expiration
	if expiration == 0 {
		expiration = cfg.JWT.ExpireHours
	}
	return &authService{
		userRepo:   repository.NewUserRepository(),
		jwtService: jwt.NewJWTService(cfg.JWT.Secret, expiration),
	}
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// 查询用户
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		logger.Error("查询用户失败", zap.Error(err))
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, ErrUserDisabled
	}

	// 验证密码
	if !s.verifyPassword(req.Password, user.Password) {
		return nil, ErrPasswordIncorrect
	}

	// 生成token
	token, err := s.jwtService.GenerateToken(int64(user.AdminID), user.Username, 0) // 管理员角色固定为0
	if err != nil {
		logger.Error("生成token失败", zap.Error(err))
		return nil, err
	}

	// 更新最后登录时间
	if err := s.userRepo.UpdateLastLogin(ctx, uint64(user.AdminID)); err != nil {
		logger.Warn("更新最后登录时间失败", zap.Error(err))
	}

	logger.Info("用户登录成功",
		zap.String("username", user.Username),
		zap.Uint("admin_id", user.AdminID),
	)

	return &model.LoginResponse{
		Token:    token,
		UserInfo: user,
	}, nil
}

// Register 用户注册
func (s *authService) Register(ctx context.Context, user *model.User) error {
	// 检查用户名是否已存在
	existUser, err := s.userRepo.GetByUsername(ctx, user.Username)
	if err != nil {
		return err
	}
	if existUser != nil {
		return errors.New("用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	user.CreateTime = time.Now().Unix()
	user.UpdateTime = time.Now().Unix()

	// 创建用户
	if err := s.userRepo.Create(ctx, user); err != nil {
		logger.Error("创建用户失败", zap.Error(err))
		return err
	}

	logger.Info("用户注册成功",
		zap.String("username", user.Username),
		zap.Uint("admin_id", user.AdminID),
	)

	return nil
}

// RefreshToken 刷新token
func (s *authService) RefreshToken(ctx context.Context, token string) (string, error) {
	newToken, err := s.jwtService.RefreshToken(token)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

// GetUserByToken 根据token获取用户信息
func (s *authService) GetUserByToken(ctx context.Context, token string) (*model.User, error) {
	claims, err := s.jwtService.ParseToken(token)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, uint64(claims.UserID))
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive() {
		return nil, ErrUserDisabled
	}

	return user, nil
}

// verifyPassword 验证密码
func (s *authService) verifyPassword(inputPassword, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
	return err == nil
}