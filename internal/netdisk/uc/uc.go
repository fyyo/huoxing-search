package uc

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

// UCClient UC网盘客户端
type UCClient struct {
	cookie     string
	httpClient *http.Client
	configRepo repository.ConfigRepository
}

// NewUCClient 创建UC网盘客户端 - 只需要cookie
func NewUCClient(cookie string, configRepo repository.ConfigRepository) *UCClient {
	return &UCClient{
		cookie:     cookie,
		configRepo: configRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Transfer 实现转存功能 - 参考PHP版本UcPan.php的transfer方法
func (c *UCClient) Transfer(ctx context.Context, shareURL, password string, expiredType int) (*model.TransferResult, error) {
	// 1. 从分享链接提取share_id
	shareID, err := c.extractShareID(shareURL)
	if err != nil {
		return nil, fmt.Errorf("提取share_id失败: %w", err)
	}

	// 2. 获取分享详情
	shareInfo, err := c.getShareInfo(ctx, shareID, password)
	if err != nil {
		return nil, fmt.Errorf("获取分享详情失败: %w", err)
	}

	// 3. 动态获取转存目录 - 参考PHP版本UcPan.php第224-227行
	folderID, err := c.getToPdirFid(ctx, expiredType)
	if err != nil {
		return nil, fmt.Errorf("获取转存目录失败: %w", err)
	}

	// 4. 转存文件到自己的网盘
	fileIDs := make([]string, 0, len(shareInfo.Files))
	for _, file := range shareInfo.Files {
		fileIDs = append(fileIDs, file.FileID)
	}

	if err := c.saveFiles(ctx, shareID, fileIDs, folderID); err != nil {
		return nil, fmt.Errorf("转存文件失败: %w", err)
	}

	// 5. 创建新的分享链接
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

// getToPdirFid 动态获取转存目录 - 参考PHP版本UcPan.php第224-227行
func (c *UCClient) getToPdirFid(ctx context.Context, expiredType int) (string, error) {
	var configKey string
	if expiredType == 2 {
		configKey = "uc_file_time" // 临时资源路径
	} else {
		configKey = "uc_file" // 默认存储路径
	}
	
	folderID, err := c.configRepo.Get(ctx, configKey)
	if err != nil {
		return "", fmt.Errorf("读取配置%s失败: %w", configKey, err)
	}
	
	if folderID == "" {
		return "0", nil // 默认根目录
	}
	
	return folderID, nil
}

// extractShareID 从分享链接提取share_id
func (c *UCClient) extractShareID(shareURL string) (string, error) {
	// UC网盘分享链接格式类似: https://drive.uc.cn/s/abc123
	parts := strings.Split(shareURL, "/s/")
	if len(parts) < 2 {
		return "", fmt.Errorf("无效的分享链接")
	}
	return strings.TrimSpace(parts[1]), nil
}

// getShareInfo 获取分享详情
func (c *UCClient) getShareInfo(ctx context.Context, shareID, password string) (*ShareInfo, error) {
	body := map[string]interface{}{
		"share_id": shareID,
		"password": password,
	}

	var result struct {
		Code int       `json:"code"`
		Data ShareInfo `json:"data"`
		Msg  string    `json:"msg"`
	}

	if err := c.doRequest(ctx, "POST", "https://drive.uc.cn/api/share/detail", body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("获取分享详情失败: %s", result.Msg)
	}

	return &result.Data, nil
}

// saveFiles 转存文件
func (c *UCClient) saveFiles(ctx context.Context, shareID string, fileIDs []string, toFolderID string) error {
	body := map[string]interface{}{
		"share_id":     shareID,
		"file_ids":     fileIDs,
		"to_folder_id": toFolderID,
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := c.doRequest(ctx, "POST", "https://drive.uc.cn/api/share/save", body, &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("转存文件失败: %s", result.Msg)
	}

	return nil
}

// createShare 创建分享 - 参考PHP版本UcPan.php第252-269行
func (c *UCClient) createShare(ctx context.Context, fileIDs []string, expiredType int) (string, string, error) {
	password := "6666"
	body := map[string]interface{}{
		"file_ids":     fileIDs,
		"expired_type": expiredType, // 使用传入的过期类型
		"password":     password,
		"title":        "云盘资源分享",
		"url_type":     1,
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ShareURL string `json:"share_url"`
		} `json:"data"`
	}

	if err := c.doRequest(ctx, "POST", "https://pc-api.uc.cn/1/clouddrive/share", body, &result); err != nil {
		return "", "", err
	}

	if result.Code != 0 {
		return "", "", fmt.Errorf("创建分享失败: %s", result.Msg)
	}

	return result.Data.ShareURL, password, nil
}

// doRequest 执行HTTP请求
func (c *UCClient) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Cookie", c.cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
	}

	return nil
}

// 数据结构

type ShareInfo struct {
	Title string     `json:"title"`
	Files []UCFile   `json:"files"`
}

type UCFile struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// GetName 获取网盘名称
func (c *UCClient) GetName() string {
	return "UC网盘"
}

// IsConfigured 检查是否已配置 - 实时从数据库读取
func (c *UCClient) IsConfigured() bool {
	// 先检查初始化时的cookie
	if c.cookie != "" {
		return true
	}
	
	// 如果初始化时没有cookie，尝试从数据库读取最新配置
	if c.configRepo != nil {
		ctx := context.Background()
		conf, err := c.configRepo.GetByName(ctx, "uc_cookie")
		if err == nil && conf != nil && conf.Value != "" {
			// 更新内存中的cookie
			c.cookie = conf.Value
			return true
		}
	}
	
	return false
}

// DeleteDirectory 删除指定目录
func (c *UCClient) DeleteDirectory(ctx context.Context, dirPath string) error {
	// 1. 列出根目录找到目标目录
	body := map[string]interface{}{
		"folder_id": "0",
		"page":      1,
		"size":      100,
	}
	
	var result struct {
		Code int `json:"code"`
		Data struct {
			Files []UCFile `json:"files"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	
	if err := c.doRequest(ctx, "POST", "https://drive.uc.cn/api/file/list", body, &result); err != nil {
		return fmt.Errorf("列出根目录失败: %w", err)
	}
	
	if result.Code != 0 {
		return fmt.Errorf("列出根目录失败: %s", result.Msg)
	}
	
	var targetFileID string
	for _, file := range result.Data.Files {
		if file.FileName == dirPath {
			targetFileID = file.FileID
			break
		}
	}
	
	if targetFileID == "" {
		return fmt.Errorf("目录不存在: %s", dirPath)
	}
	
	// 2. 删除目录
	deleteBody := map[string]interface{}{
		"file_ids": []string{targetFileID},
	}
	
	var deleteResult struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	
	if err := c.doRequest(ctx, "POST", "https://drive.uc.cn/api/file/delete", deleteBody, &deleteResult); err != nil {
		return err
	}
	
	if deleteResult.Code != 0 {
		return fmt.Errorf("删除目录失败: %s", deleteResult.Msg)
	}
	
	return nil
}

// CreateDirectory 创建指定目录
func (c *UCClient) CreateDirectory(ctx context.Context, dirPath string) error {
	body := map[string]interface{}{
		"parent_id": "0",
		"name":      dirPath,
	}
	
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	
	if err := c.doRequest(ctx, "POST", "https://drive.uc.cn/api/folder/create", body, &result); err != nil {
		return err
	}
	
	if result.Code != 0 {
		return fmt.Errorf("创建目录失败: %s", result.Msg)
	}
	
	return nil
}

// TestConnection 测试UC网盘连接
func (c *UCClient) TestConnection(ctx context.Context) error {
	// 测试策略：调用获取用户信息API，验证cookie是否有效
	body := map[string]interface{}{
		"folder_id": "0",
		"page":      1,
		"size":      1,
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Files []UCFile `json:"files"`
		} `json:"data"`
	}

	if err := c.doRequest(ctx, "POST", "https://drive.uc.cn/api/file/list", body, &result); err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}

	if result.Code != 0 {
		if result.Code == 401 {
			return fmt.Errorf("Cookie已过期或无效，请重新获取")
		}
		return fmt.Errorf("连接失败: %s (错误码:%d)", result.Msg, result.Code)
	}

	return nil
}