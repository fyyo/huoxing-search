package xunlei

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

// XunleiClient 迅雷网盘客户端
type XunleiClient struct {
	refreshToken string
	accessToken  string
	userID       string
	httpClient   *http.Client
	configRepo   repository.ConfigRepository
}

// NewXunleiClient 创建迅雷网盘客户端 - 只需要refreshToken
func NewXunleiClient(refreshToken string, configRepo repository.ConfigRepository) *XunleiClient {
	return &XunleiClient{
		refreshToken: refreshToken,
		configRepo:   configRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Transfer 实现转存功能 - 参考PHP版本XunleiPan.php的transfer方法
func (c *XunleiClient) Transfer(ctx context.Context, shareURL, password string, expiredType int) (*model.TransferResult, error) {
	// 1. 刷新access token
	if err := c.refreshAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("刷新token失败: %w", err)
	}

	// 2. 从分享链接提取share_id
	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return nil, fmt.Errorf("提取share_id失败: %w", err)
	}

	// 3. 获取分享详情
	shareInfo, err := c.getShareInfo(ctx, shareID, password)
	if err != nil {
		return nil, fmt.Errorf("获取分享详情失败: %w", err)
	}

	// 4. 动态获取转存目录 - 参考PHP版本XunleiPan.php第402-405行
	folderID, err := c.getToPdirFid(ctx, expiredType)
	if err != nil {
		return nil, fmt.Errorf("获取转存目录失败: %w", err)
	}

	// 5. 转存文件到自己的网盘
	fileIDs := make([]string, 0, len(shareInfo.Files))
	for _, file := range shareInfo.Files {
		fileIDs = append(fileIDs, file.FileID)
	}

	if err := c.saveFiles(ctx, shareID, fileIDs, folderID); err != nil {
		return nil, fmt.Errorf("转存文件失败: %w", err)
	}

	// 6. 创建新的分享链接
	newShareURL, newPassword, err := c.createShare(ctx, fileIDs, expiredType)
	if err != nil {
		return nil, fmt.Errorf("创建分享失败: %w", err)
	}

	result := &model.TransferResult{
		Title:       shareInfo.Title,
		OriginalURL: shareURL,
		ShareURL:    newShareURL,
		Password:    newPassword,
		Success:     true,
		Message:     "转存成功",
	}

	return result, nil
}

// getToPdirFid 动态获取转存目录 - 参考PHP版本XunleiPan.php第402-405行
func (c *XunleiClient) getToPdirFid(ctx context.Context, expiredType int) (string, error) {
	var configKey string
	if expiredType == 2 {
		configKey = "xunlei_file_time" // 临时资源路径
	} else {
		configKey = "xunlei_file" // 默认存储路径
	}
	
	folderID, err := c.configRepo.Get(ctx, configKey)
	if err != nil {
		return "", fmt.Errorf("读取配置%s失败: %w", configKey, err)
	}
	
	if folderID == "" {
		return "", nil // 默认根目录
	}
	
	return folderID, nil
}

// refreshAccessToken 刷新访问令牌
func (c *XunleiClient) refreshAccessToken(ctx context.Context) error {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": c.refreshToken,
		"client_id":     "Xqp0kJBXWhwaTpB6",
		"client_secret": "",
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		UserID       string `json:"user_id"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.xpan.xunlei.com/oauth/token", body, &result); err != nil {
		return err
	}

	c.accessToken = result.AccessToken
	c.refreshToken = result.RefreshToken
	c.userID = result.UserID

	return nil
}

// extractShareID 从分享链接提取share_id
func (c *XunleiClient) extractShareID(shareURL string) (string, error) {
	// 迅雷网盘分享链接格式: https://pan.xunlei.com/s/abc123
	parts := strings.Split(shareURL, "/s/")
	if len(parts) < 2 {
		return "", fmt.Errorf("无效的分享链接")
	}
	return strings.TrimSpace(parts[1]), nil
}

// getShareInfo 获取分享详情
func (c *XunleiClient) getShareInfo(ctx context.Context, shareID, password string) (*ShareInfo, error) {
	url := fmt.Sprintf("https://api.xpan.xunlei.com/drive/v1/share/%s", shareID)
	
	body := map[string]string{
		"share_pwd": password,
	}

	var result struct {
		ShareInfo ShareInfo `json:"share_info"`
	}

	if err := c.doRequest(ctx, "POST", url, body, &result); err != nil {
		return nil, err
	}

	return &result.ShareInfo, nil
}

// saveFiles 转存文件
func (c *XunleiClient) saveFiles(ctx context.Context, shareID string, fileIDs []string, toFolderID string) error {
	body := map[string]interface{}{
		"share_id":       shareID,
		"file_id_list":   fileIDs,
		"to_parent_id":   toFolderID,
		"to_drive_id":    c.userID,
	}

	var result struct {
		TaskID string `json:"task_id"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.xpan.xunlei.com/drive/v1/share/save", body, &result); err != nil {
		return err
	}

	// 等待转存任务完成
	return c.waitForTask(ctx, result.TaskID)
}

// waitForTask 等待任务完成
func (c *XunleiClient) waitForTask(ctx context.Context, taskID string) error {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		url := fmt.Sprintf("https://api.xpan.xunlei.com/drive/v1/task/%s", taskID)
		
		var result struct {
			Status string `json:"status"`
		}

		if err := c.doRequest(ctx, "GET", url, nil, &result); err != nil {
			return err
		}

		if result.Status == "completed" {
			return nil
		}

		if result.Status == "failed" {
			return fmt.Errorf("任务失败")
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf("任务超时")
}

// createShare 创建分享 - 参考PHP版本XunleiPan.php第462-493行
func (c *XunleiClient) createShare(ctx context.Context, fileIDs []string, expiredType int) (string, string, error) {
	// 根据expiredType设置过期天数 - 参考PHP版本第465-468行
	expirationDays := "-1" // 永久
	if expiredType == 2 {
		expirationDays = "2" // 2天
	}
	
	body := map[string]interface{}{
		"file_ids": fileIDs,
		"share_to": "copy",
		"params": map[string]interface{}{
			"subscribe_push":    "false",
			"WithPassCodeInLink": "true",
		},
		"title":           "云盘资源分享",
		"restore_limit":   "-1",
		"expiration_days": expirationDays,
	}

	var result struct {
		ShareURL string `json:"share_url"`
		PassCode string `json:"pass_code"`
	}

	if err := c.doRequest(ctx, "POST", "https://api-pan.xunlei.com/drive/v1/share", body, &result); err != nil {
		return "", "", err
	}

	// 拼接密码到URL - 参考PHP第346行
	shareURLWithPwd := result.ShareURL + "?pwd=" + result.PassCode
	return shareURLWithPwd, result.PassCode, nil
}

// doRequest 执行HTTP请求
func (c *XunleiClient) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
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

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
		}
	}

	return nil
}

// 数据结构

type ShareInfo struct {
	Title string        `json:"title"`
	Files []XunleiFile  `json:"files"`
}

type XunleiFile struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// GetName 获取网盘名称
func (c *XunleiClient) GetName() string {
	return "迅雷网盘"
}

// IsConfigured 检查是否已配置 - 实时从数据库读取
func (c *XunleiClient) IsConfigured() bool {
	// 先检查初始化时的refreshToken
	if c.refreshToken != "" {
		return true
	}
	
	// 如果初始化时没有refreshToken，尝试从数据库读取最新配置
	if c.configRepo != nil {
		ctx := context.Background()
		conf, err := c.configRepo.GetByName(ctx, "xunlei_cookie")
		if err == nil && conf != nil && conf.Value != "" {
			// 更新内存中的refreshToken
			c.refreshToken = conf.Value
			return true
		}
	}
	
	return false
}

// DeleteDirectory 删除指定目录
func (c *XunleiClient) DeleteDirectory(ctx context.Context, dirPath string) error {
	// 1. 刷新token
	if err := c.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("刷新token失败: %w", err)
	}
	
	// 2. 列出根目录找到目标目录
	body := map[string]interface{}{
		"parent_id": "",
		"page":      1,
		"per_page":  100,
	}
	
	var result struct {
		Files []XunleiFile `json:"files"`
	}
	
	if err := c.doRequest(ctx, "POST", "https://api-pan.xunlei.com/drive/v1/files", body, &result); err != nil {
		return fmt.Errorf("列出根目录失败: %w", err)
	}
	
	var targetFileID string
	for _, file := range result.Files {
		if file.FileName == dirPath {
			targetFileID = file.FileID
			break
		}
	}
	
	if targetFileID == "" {
		return fmt.Errorf("目录不存在: %s", dirPath)
	}
	
	// 3. 删除目录
	deleteBody := map[string]interface{}{
		"file_ids": []string{targetFileID},
	}
	
	var deleteResult map[string]interface{}
	return c.doRequest(ctx, "POST", "https://api-pan.xunlei.com/drive/v1/files/trash", deleteBody, &deleteResult)
}

// CreateDirectory 创建指定目录
func (c *XunleiClient) CreateDirectory(ctx context.Context, dirPath string) error {
	// 1. 刷新token
	if err := c.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("刷新token失败: %w", err)
	}
	
	// 2. 创建目录
	body := map[string]interface{}{
		"parent_id": "",
		"name":      dirPath,
		"kind":      "drive#folder",
	}
	
	var result map[string]interface{}
	return c.doRequest(ctx, "POST", "https://api-pan.xunlei.com/drive/v1/files", body, &result)
}

// TestConnection 测试迅雷网盘连接
func (c *XunleiClient) TestConnection(ctx context.Context) error {
	// 测试策略：刷新token，如果成功说明refreshToken有效
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": c.refreshToken,
		"client_id":     "Xqp0kJBXWhwaTpB6",
		"client_secret": "",
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		UserID       string `json:"user_id"`
		ErrorCode    string `json:"error_code"`
		ErrorMsg     string `json:"error_msg"`
	}

	if err := c.doRequest(ctx, "POST", "https://api.xpan.xunlei.com/oauth/token", body, &result); err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}

	if result.ErrorCode != "" {
		if result.ErrorCode == "invalid_grant" {
			return fmt.Errorf("RefreshToken已过期或无效，请重新获取")
		}
		return fmt.Errorf("连接失败: %s (错误码:%s)", result.ErrorMsg, result.ErrorCode)
	}

	if result.AccessToken == "" {
		return fmt.Errorf("获取AccessToken失败")
	}

	return nil
}