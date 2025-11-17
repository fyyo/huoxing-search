package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"xinyue-go/internal/model"
	"xinyue-go/internal/repository"
)

// AliyunClient 阿里云盘客户端
type AliyunClient struct {
	refreshToken string
	accessToken  string
	driveID      string
	httpClient   *http.Client
	configRepo   repository.ConfigRepository
}

// NewAliyunClient 创建阿里云盘客户端 - 只需要refreshToken
func NewAliyunClient(refreshToken string, configRepo repository.ConfigRepository) *AliyunClient {
	return &AliyunClient{
		refreshToken: refreshToken,
		configRepo:   configRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Transfer 实现转存功能 - 参考PHP版本AlipanPan.php的transfer方法
func (c *AliyunClient) Transfer(ctx context.Context, shareURL, password string, expiredType int) (*model.TransferResult, error) {
	// 1. 刷新access token
	if err := c.refreshAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("刷新token失败: %w", err)
	}

	// 2. 从分享链接提取share_id
	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return nil, fmt.Errorf("提取share_id失败: %w", err)
	}

	// 3. 获取分享token
	shareToken, err := c.getShareToken(ctx, shareID, password)
	if err != nil {
		return nil, fmt.Errorf("获取分享token失败: %w", err)
	}

	// 4. 获取分享文件列表
	files, err := c.getShareFileList(ctx, shareID, shareToken)
	if err != nil {
		return nil, fmt.Errorf("获取分享文件列表失败: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("分享链接中没有文件")
	}

	// 5. 动态获取转存目录 - 参考PHP版本第101-104行
	folderID, err := c.getToPdirFid(ctx, expiredType)
	if err != nil {
		return nil, fmt.Errorf("获取转存目录失败: %w", err)
	}

	// 6. 转存文件到自己的网盘
	fileIDs := make([]string, 0, len(files))
	for _, file := range files {
		fileIDs = append(fileIDs, file.FileID)
	}

	if err := c.saveFiles(ctx, shareID, shareToken, fileIDs, folderID); err != nil {
		return nil, fmt.Errorf("转存文件失败: %w", err)
	}

	// 7. 创建新的分享链接
	newShareURL, newPassword, err := c.createShare(ctx, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("创建分享失败: %w", err)
	}

	result := &model.TransferResult{
		Title:       files[0].Name,
		OriginalURL: shareURL,
		ShareURL:    newShareURL,
		Password:    newPassword,
		Success:     true,
		Message:     "转存成功",
	}

	return result, nil
}

// getToPdirFid 动态获取转存目录 - 参考PHP版本AlipanPan.php第101-104行
func (c *AliyunClient) getToPdirFid(ctx context.Context, expiredType int) (string, error) {
	var configKey string
	if expiredType == 2 {
		configKey = "ali_file_time" // 临时资源路径
	} else {
		configKey = "ali_file" // 默认存储路径
	}
	
	folderID, err := c.configRepo.Get(ctx, configKey)
	if err != nil {
		return "", fmt.Errorf("读取配置%s失败: %w", configKey, err)
	}
	
	if folderID == "" {
		return "root", nil // 默认值
	}
	
	return folderID, nil
}

// refreshAccessToken 刷新访问令牌
func (c *AliyunClient) refreshAccessToken(ctx context.Context) error {
	body := map[string]string{
		"refresh_token": c.refreshToken,
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		DriveID      string `json:"default_drive_id"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.aliyundrive.com/token/refresh", body, &result); err != nil {
		return err
	}

	c.accessToken = result.AccessToken
	c.refreshToken = result.RefreshToken
	c.driveID = result.DriveID

	return nil
}

// extractShareID 从分享链接提取share_id
func (c *AliyunClient) extractShareID(shareURL string) (string, error) {
	// 示例: https://www.aliyundrive.com/s/abc123def
	parts := strings.Split(shareURL, "/s/")
	if len(parts) < 2 {
		return "", fmt.Errorf("无效的分享链接")
	}
	return strings.TrimSpace(parts[1]), nil
}

// getShareToken 获取分享token
func (c *AliyunClient) getShareToken(ctx context.Context, shareID, password string) (string, error) {
	body := map[string]string{
		"share_id":   shareID,
		"share_pwd":  password,
	}

	var result struct {
		ShareToken string `json:"share_token"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.aliyundrive.com/v2/share_link/get_share_token", body, &result); err != nil {
		return "", err
	}

	return result.ShareToken, nil
}

// getShareFileList 获取分享文件列表
func (c *AliyunClient) getShareFileList(ctx context.Context, shareID, shareToken string) ([]AliyunFile, error) {
	body := map[string]interface{}{
		"share_id":        shareID,
		"parent_file_id":  "root",
		"limit":           100,
		"order_by":        "name",
		"order_direction": "ASC",
	}

	headers := map[string]string{
		"X-Share-Token": shareToken,
	}

	var result struct {
		Items []AliyunFile `json:"items"`
	}

	if err := c.doRequestWithHeaders(ctx, "POST", "https://api.aliyundrive.com/adrive/v3/file/list", body, headers, &result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// saveFiles 转存文件
func (c *AliyunClient) saveFiles(ctx context.Context, shareID, shareToken string, fileIDs []string, toDriveID string) error {
	body := map[string]interface{}{
		"share_id":       shareID,
		"file_id_list":   fileIDs,
		"to_parent_file_id": toDriveID,
		"to_drive_id":    c.driveID,
		"auto_rename":    true,
	}

	headers := map[string]string{
		"X-Share-Token": shareToken,
	}

	var result map[string]interface{}
	return c.doRequestWithHeaders(ctx, "POST", "https://api.aliyundrive.com/adrive/v2/file/copy", body, headers, &result)
}

// createShare 创建分享
func (c *AliyunClient) createShare(ctx context.Context, fileIDs []string) (string, string, error) {
	body := map[string]interface{}{
		"drive_id":      c.driveID,
		"file_id_list":  fileIDs,
		"share_pwd":     "6666",
		"expiration":    "", // 永久有效
	}

	var result struct {
		ShareURL string `json:"share_url"`
		SharePwd string `json:"share_pwd"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.aliyundrive.com/adrive/v2/share_link/create", body, &result); err != nil {
		return "", "", err
	}

	return result.ShareURL, result.SharePwd, nil
}

// doRequest 执行HTTP请求
func (c *AliyunClient) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	return c.doRequestWithHeaders(ctx, method, url, body, nil, result)
}

// doRequestWithHeaders 执行HTTP请求(带自定义headers)
func (c *AliyunClient) doRequestWithHeaders(ctx context.Context, method, url string, body interface{}, extraHeaders map[string]string, result interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置基础headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	// 添加额外headers
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败,状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
	}

	return nil
}

// AliyunFile 阿里云盘文件信息
type AliyunFile struct {
	FileID string `json:"file_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// GetName 获取网盘名称
func (c *AliyunClient) GetName() string {
	return "阿里云盘"
}

// IsConfigured 检查是否已配置 - 实时从数据库读取
func (c *AliyunClient) IsConfigured() bool {
	// 先检查初始化时的refreshToken
	if c.refreshToken != "" {
		return true
	}
	
	// 如果初始化时没有refreshToken，尝试从数据库读取最新配置
	if c.configRepo != nil {
		ctx := context.Background()
		conf, err := c.configRepo.GetByName(ctx, "Authorization")
		if err == nil && conf != nil && conf.Value != "" {
			// 更新内存中的refreshToken
			c.refreshToken = conf.Value
			return true
		}
	}
	
	return false
}

// DeleteDirectory 删除指定目录
func (c *AliyunClient) DeleteDirectory(ctx context.Context, dirPath string) error {
	// 1. 刷新token
	if err := c.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("刷新token失败: %w", err)
	}
	
	// 2. 列出根目录找到目标目录
	body := map[string]interface{}{
		"drive_id":        c.driveID,
		"parent_file_id":  "root",
		"limit":           100,
		"order_by":        "name",
		"order_direction": "ASC",
	}
	
	var result struct {
		Items []AliyunFile `json:"items"`
	}
	
	if err := c.doRequest(ctx, "POST", "https://api.aliyundrive.com/adrive/v3/file/list", body, &result); err != nil {
		return fmt.Errorf("列出根目录失败: %w", err)
	}
	
	var targetFileID string
	for _, file := range result.Items {
		if file.Name == dirPath && file.Type == "folder" {
			targetFileID = file.FileID
			break
		}
	}
	
	if targetFileID == "" {
		return fmt.Errorf("目录不存在: %s", dirPath)
	}
	
	// 3. 删除目录
	deleteBody := map[string]interface{}{
		"drive_id": c.driveID,
		"file_id":  targetFileID,
	}
	
	var deleteResult map[string]interface{}
	return c.doRequest(ctx, "POST", "https://api.aliyundrive.com/v2/recyclebin/trash", deleteBody, &deleteResult)
}

// CreateDirectory 创建指定目录
func (c *AliyunClient) CreateDirectory(ctx context.Context, dirPath string) error {
	// 1. 刷新token
	if err := c.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("刷新token失败: %w", err)
	}
	
	// 2. 创建目录
	body := map[string]interface{}{
		"drive_id":         c.driveID,
		"parent_file_id":   "root",
		"name":             dirPath,
		"type":             "folder",
		"check_name_mode":  "refuse",
	}
	
	var result map[string]interface{}
	return c.doRequest(ctx, "POST", "https://api.aliyundrive.com/adrive/v2/file/createWithFolders", body, &result)
}

// TestConnection 测试阿里云盘连接
func (c *AliyunClient) TestConnection(ctx context.Context) error {
	// 测试策略：刷新token，如果成功说明refreshToken有效
	body := map[string]string{
		"refresh_token": c.refreshToken,
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		DriveID      string `json:"default_drive_id"`
		Code         string `json:"code"`
		Message      string `json:"message"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.aliyundrive.com/token/refresh", body, &result); err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}

	if result.Code != "" {
		if result.Code == "RefreshTokenExpired" {
			return fmt.Errorf("RefreshToken已过期，请重新获取")
		}
		if result.Code == "InvalidParameter.RefreshToken" {
			return fmt.Errorf("RefreshToken无效，请检查配置")
		}
		return fmt.Errorf("连接失败: %s (错误码:%s)", result.Message, result.Code)
	}

	if result.AccessToken == "" {
		return fmt.Errorf("获取AccessToken失败")
	}

	return nil
}