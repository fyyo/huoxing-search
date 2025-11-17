package quark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"xinyue-go/internal/model"
	"xinyue-go/internal/pkg/logger"
	"xinyue-go/internal/repository"
)

// QuarkClient 夸克网盘客户端
type QuarkClient struct {
	cookie      string
	httpClient  *http.Client
	configRepo  repository.ConfigRepository // 用于动态读取配置
}

// NewQuarkClient 创建夸克网盘客户端 - 接受configRepo参数
func NewQuarkClient(cookie string, configRepo repository.ConfigRepository) *QuarkClient {
	return &QuarkClient{
		cookie:     cookie,
		configRepo: configRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Transfer 实现转存功能 - 添加expiredType参数
func (c *QuarkClient) Transfer(ctx context.Context, shareURL, password string, expiredType int) (*model.TransferResult, error) {
	// 如果expiredType无效，使用默认值1(1天)
	if expiredType < 1 || expiredType > 4 {
		expiredType = 1
	}

	// 1. 从分享链接提取 pwd_id
	pwdID, err := c.extractPwdID(shareURL)
	if err != nil {
		return nil, fmt.Errorf("提取pwd_id失败: %w", err)
	}

	// 2. 获取 stoken
	stokenResp, err := c.getStoken(ctx, pwdID, password)
	if err != nil {
		return nil, fmt.Errorf("获取stoken失败: %w", err)
	}

	stoken := strings.ReplaceAll(stokenResp.Stoken, " ", "+")
	title := stokenResp.Title

	// 3. 获取分享详情
	shareDetail, err := c.getShareDetail(ctx, pwdID, stoken)
	if err != nil {
		return nil, fmt.Errorf("获取分享详情失败: %w", err)
	}

	// 提取文件列表
	fidList := make([]string, 0, len(shareDetail.List))
	fidTokenList := make([]string, 0, len(shareDetail.List))
	for _, file := range shareDetail.List {
		fidList = append(fidList, file.Fid)
		fidTokenList = append(fidTokenList, file.ShareFidToken)
	}

	// 4. 转存到自己的网盘（传入expiredType用于选择目录）
	saveResp, err := c.saveToMyDrive(ctx, pwdID, stoken, fidList, fidTokenList, expiredType)
	if err != nil {
		return nil, fmt.Errorf("转存失败: %w", err)
	}

	// 5. 等待转存任务完成
	saveTask, err := c.waitForTask(ctx, saveResp.TaskID, 50)
	if err != nil {
		return nil, fmt.Errorf("等待转存任务失败: %w", err)
	}

	savedFids := saveTask.SaveAs.SaveAsTopFids

	// 6. 清理广告文件
	bannedWords := c.getBannedWords(ctx)
	if len(bannedWords) > 0 && len(savedFids) > 0 {
		if err := c.cleanBannedFiles(ctx, savedFids[0], bannedWords); err != nil {
			logger.Warn("清理广告文件失败", zap.Error(err))
		}
	}

	// 7. 分享转存后的文件（传入expiredType）
	shareResp, err := c.shareFiles(ctx, savedFids, title, expiredType)
	if err != nil {
		return nil, fmt.Errorf("分享文件失败: %w", err)
	}

	// 8. 等待分享任务完成
	shareTask, err := c.waitForTask(ctx, shareResp.TaskID, 50)
	if err != nil {
		return nil, fmt.Errorf("等待分享任务失败: %w", err)
	}

	// 9. 获取分享密码和链接
	shareInfo, err := c.getSharePassword(ctx, shareTask.ShareID)
	if err != nil {
		return nil, fmt.Errorf("获取分享链接失败: %w", err)
	}

	// 构造返回结果
	result := &model.TransferResult{
		Title:       title,
		OriginalURL: shareURL,
		ShareURL:    shareInfo.ShareURL,
		Password:    shareInfo.PassCode,
		Success:     true,
		Message:     "转存成功",
	}

	return result, nil
}

// GetFiles 获取文件列表
func (c *QuarkClient) GetFiles(ctx context.Context, pdirFid string) ([]map[string]interface{}, error) {
	params := url.Values{
		"pr":               {"ucpro"},
		"fr":               {"pc"},
		"uc_param_str":     {""},
		"pdir_fid":         {pdirFid},
		"_page":            {"1"},
		"_size":            {"50"},
		"_fetch_total":     {"1"},
		"_fetch_sub_dirs":  {"0"},
		"_sort":            {"file_type:asc,updated_at:desc"},
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    struct {
			List []map[string]interface{} `json:"list"`
		} `json:"data"`
	}

	err := c.doRequest(ctx, "GET", "https://drive-pc.quark.cn/1/clouddrive/file/sort", params, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("获取文件列表失败: %s", result.Message)
	}

	return result.Data.List, nil
}

// extractPwdID 从分享链接提取 pwd_id
func (c *QuarkClient) extractPwdID(shareURL string) (string, error) {
	// 示例: https://pan.quark.cn/s/abc123def456
	parts := strings.Split(shareURL, "/s/")
	if len(parts) < 2 {
		return "", fmt.Errorf("无效的分享链接")
	}
	pwdID := strings.TrimSpace(parts[1])
	if pwdID == "" {
		return "", fmt.Errorf("无法提取pwd_id")
	}
	return pwdID, nil
}

// getStoken 获取 stoken
func (c *QuarkClient) getStoken(ctx context.Context, pwdID, passcode string) (*StokenResponse, error) {
	params := url.Values{
		"pr":           {"ucpro"},
		"fr":           {"pc"},
		"uc_param_str": {""},
	}

	body := map[string]string{
		"passcode": passcode,
		"pwd_id":   pwdID,
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    StokenResponse `json:"data"`
	}

	err := c.doRequest(ctx, "POST", "https://drive-pc.quark.cn/1/clouddrive/share/sharepage/token", params, body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("获取stoken失败: %s", result.Message)
	}

	return &result.Data, nil
}

// getShareDetail 获取分享详情
func (c *QuarkClient) getShareDetail(ctx context.Context, pwdID, stoken string) (*ShareDetailResponse, error) {
	params := url.Values{
		"pr":            {"ucpro"},
		"fr":            {"pc"},
		"uc_param_str":  {""},
		"pwd_id":        {pwdID},
		"stoken":        {stoken},
		"pdir_fid":      {"0"},
		"force":         {"0"},
		"_page":         {"1"},
		"_size":         {"100"},
		"_fetch_banner": {"1"},
		"_fetch_share":  {"1"},
		"_fetch_total":  {"1"},
		"_sort":         {"file_type:asc,updated_at:desc"},
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    ShareDetailResponse `json:"data"`
	}

	err := c.doRequest(ctx, "GET", "https://drive-pc.quark.cn/1/clouddrive/share/sharepage/detail", params, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("获取分享详情失败: %s", result.Message)
	}

	return &result.Data, nil
}

// saveToMyDrive 转存到自己的网盘
func (c *QuarkClient) saveToMyDrive(ctx context.Context, pwdID, stoken string, fidList, fidTokenList []string, expiredType int) (*TaskResponse, error) {
	params := url.Values{
		"entry":        {"update_share"},
		"pr":           {"ucpro"},
		"fr":           {"pc"},
		"uc_param_str": {""},
	}

	// 动态读取转存目录配置（参考PHP版本逻辑）
	toPdirFid, err := c.getToPdirFid(ctx, expiredType)
	if err != nil {
		logger.Warn("读取转存目录配置失败，使用默认值", zap.Error(err))
		toPdirFid = "0" // 默认根目录
	}

	body := map[string]interface{}{
		"fid_list":       fidList,
		"fid_token_list": fidTokenList,
		"to_pdir_fid":    toPdirFid,
		"pwd_id":         pwdID,
		"stoken":         stoken,
		"pdir_fid":       "0",
		"scene":          "link",
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    TaskResponse `json:"data"`
	}

	err = c.doRequest(ctx, "POST", "https://drive-pc.quark.cn/1/clouddrive/share/sharepage/save", params, body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != 200 {
		if result.Message == "capacity limit[{0}]" {
			return nil, fmt.Errorf("容量不足")
		}
		return nil, fmt.Errorf("转存失败: %s", result.Message)
	}

	return &result.Data, nil
}

// shareFiles 分享文件
func (c *QuarkClient) shareFiles(ctx context.Context, fidList []string, title string, expiredType int) (*TaskResponse, error) {
	// 动态读取广告文件ID配置
	adFid, _ := c.configRepo.GetByName(ctx, "quark_file")
	if adFid != nil && adFid.Value != "" && adFid.Value != "0" {
		fidList = append(fidList, adFid.Value)
	}

	params := url.Values{
		"pr":           {"ucpro"},
		"fr":           {"pc"},
		"uc_param_str": {""},
	}

	// 过期类型验证: 1=1天, 2=7天, 3=30天, 4=永久
	// 如果expiredType为0或无效，默认使用1(1天)
	if expiredType < 1 || expiredType > 4 {
		originalValue := expiredType // 记录原始值
		expiredType = 1 // 默认1天
		logger.Warn("无效的过期类型，使用默认值1(1天)", zap.Int("original", originalValue))
	}

	body := map[string]interface{}{
		"fid_list":     fidList,
		"expired_type": expiredType,
		"title":        title,
		"url_type":     1,
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    TaskResponse `json:"data"`
	}

	err := c.doRequest(ctx, "POST", "https://drive-pc.quark.cn/1/clouddrive/share", params, body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("分享创建失败。%s", result.Message)
	}

	return &result.Data, nil
}

// waitForTask 等待任务完成
func (c *QuarkClient) waitForTask(ctx context.Context, taskID string, maxRetries int) (*TaskStatusResponse, error) {
	for i := 0; i < maxRetries; i++ {
		params := url.Values{
			"pr":           {"ucpro"},
			"fr":           {"pc"},
			"uc_param_str": {""},
			"task_id":      {taskID},
			"retry_index":  {fmt.Sprintf("%d", i)},
		}

		var result struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
			Data    TaskStatusResponse `json:"data"`
		}

		err := c.doRequest(ctx, "GET", "https://drive-pc.quark.cn/1/clouddrive/task", params, nil, &result)
		if err != nil {
			logger.Warn("查询任务状态失败", zap.Error(err), zap.Int("retry", i))
			time.Sleep(time.Second)
			continue
		}

		if result.Status != 200 {
			return nil, fmt.Errorf("查询任务失败: %s", result.Message)
		}

		// status == 2 表示任务完成
		if result.Data.Status == 2 {
			return &result.Data, nil
		}

		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("任务超时")
}

// getSharePassword 获取分享密码
func (c *QuarkClient) getSharePassword(ctx context.Context, shareID string) (*SharePasswordResponse, error) {
	params := url.Values{
		"pr":           {"ucpro"},
		"fr":           {"pc"},
		"uc_param_str": {""},
	}

	body := map[string]string{
		"share_id": shareID,
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    SharePasswordResponse `json:"data"`
	}

	err := c.doRequest(ctx, "POST", "https://drive-pc.quark.cn/1/clouddrive/share/password", params, body, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("获取分享密码失败: %s", result.Message)
	}

	return &result.Data, nil
}

// cleanBannedFiles 清理包含禁用词的文件
func (c *QuarkClient) cleanBannedFiles(ctx context.Context, pdirFid string, bannedWords []string) error {
	files, err := c.GetFiles(ctx, pdirFid)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	toDelete := make([]string, 0)
	for _, file := range files {
		fileName, ok := file["file_name"].(string)
		if !ok {
			continue
		}

		for _, banned := range bannedWords {
			if strings.Contains(fileName, banned) {
				if fid, ok := file["fid"].(string); ok {
					toDelete = append(toDelete, fid)
				}
				break
			}
		}
	}

	// 如果所有文件都要删除,删除整个文件夹
	if len(toDelete) == len(files) {
		return c.deleteFiles(ctx, []string{pdirFid})
	}

	// 否则只删除匹配的文件
	if len(toDelete) > 0 {
		return c.deleteFiles(ctx, toDelete)
	}

	return nil
}

// deleteFiles 删除文件
func (c *QuarkClient) deleteFiles(ctx context.Context, fidList []string) error {
	params := url.Values{
		"pr":           {"ucpro"},
		"fr":           {"pc"},
		"uc_param_str": {""},
	}

	body := map[string]interface{}{
		"action_type":  2,
		"exclude_fids": []string{},
		"filelist":     fidList,
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}

	err := c.doRequest(ctx, "POST", "https://drive-pc.quark.cn/1/clouddrive/file/delete", params, body, &result)
	if err != nil {
		return err
	}

	if result.Status != 200 {
		return fmt.Errorf("删除文件失败: %s", result.Message)
	}

	return nil
}

// doRequest 执行HTTP请求
func (c *QuarkClient) doRequest(ctx context.Context, method, urlStr string, params url.Values, body interface{}, result interface{}) error {
	// 构建完整URL
	if len(params) > 0 {
		urlStr += "?" + params.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Referer", "https://pan.quark.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
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

// 响应结构体定义

type StokenResponse struct {
	Stoken string `json:"stoken"`
	Title  string `json:"title"`
}

type ShareDetailResponse struct {
	Share struct {
		Title string `json:"title"`
	} `json:"share"`
	List []struct {
		Fid            string `json:"fid"`
		ShareFidToken  string `json:"share_fid_token"`
	} `json:"list"`
}

type TaskResponse struct {
	TaskID string `json:"task_id"`
}

type TaskStatusResponse struct {
	Status  int    `json:"status"`
	ShareID string `json:"share_id"`
	SaveAs  struct {
		SaveAsTopFids []string `json:"save_as_top_fids"`
	} `json:"save_as"`
}

type SharePasswordResponse struct {
	ShareURL string `json:"share_url"`
	PassCode string `json:"passcode"`
}

// getToPdirFid 根据过期类型动态获取转存目录（参考PHP版本逻辑）
func (c *QuarkClient) getToPdirFid(ctx context.Context, expiredType int) (string, error) {
	// 如果expiredType == 2，使用临时目录，否则使用默认目录
	configName := "quark_file" // 默认目录
	if expiredType == 2 {
		configName = "quark_file_time" // 临时目录
	}

	conf, err := c.configRepo.GetByName(ctx, configName)
	if err != nil {
		return "0", err // 返回根目录作为默认值
	}

	if conf == nil || conf.Value == "" || conf.Value == "0" {
		return "0", nil // 使用根目录
	}

	return conf.Value, nil
}

// getBannedWords 动态读取禁用词配置
func (c *QuarkClient) getBannedWords(ctx context.Context) []string {
	conf, err := c.configRepo.GetByName(ctx, "quark_banned")
	if err != nil || conf == nil || conf.Value == "" {
		return []string{}
	}

	// 按逗号分割
	words := strings.Split(conf.Value, ",")
	result := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" {
			result = append(result, word)
		}
	}
	return result
}

// parseIntOrDefault 解析整数，失败返回默认值
func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return defaultVal
	}
	return val
}

// GetName 获取网盘名称
func (c *QuarkClient) GetName() string {
	return "夸克网盘"
}

// IsConfigured 检查是否已配置 - 实时从数据库读取
func (c *QuarkClient) IsConfigured() bool {
	// 先检查初始化时的cookie
	if c.cookie != "" {
		return true
	}
	
	// 如果初始化时没有cookie，尝试从数据库读取最新配置
	if c.configRepo != nil {
		ctx := context.Background()
		conf, err := c.configRepo.GetByName(ctx, "quark_cookie")
		if err == nil && conf != nil && conf.Value != "" {
			// 更新内存中的cookie
			c.cookie = conf.Value
			return true
		}
	}
	
	return false
}

// DeleteDirectory 删除指定目录
func (c *QuarkClient) DeleteDirectory(ctx context.Context, dirPath string) error {
	// 1. 获取目录信息，获取fid
	files, err := c.GetFiles(ctx, "0") // 先列出根目录
	if err != nil {
		return fmt.Errorf("列出根目录失败: %w", err)
	}
	
	var targetFid string
	for _, file := range files {
		if fileName, ok := file["file_name"].(string); ok && fileName == dirPath {
			if fid, ok := file["fid"].(string); ok {
				targetFid = fid
				break
			}
		}
	}
	
	if targetFid == "" {
		return fmt.Errorf("目录不存在: %s", dirPath)
	}
	
	// 2. 删除目录
	return c.deleteFiles(ctx, []string{targetFid})
}

// CreateDirectory 创建指定目录
func (c *QuarkClient) CreateDirectory(ctx context.Context, dirPath string) error {
	params := url.Values{
		"pr":           {"ucpro"},
		"fr":           {"pc"},
		"uc_param_str": {""},
	}
	
	body := map[string]interface{}{
		"pdir_fid":   "0", // 在根目录创建
		"file_name":  dirPath,
		"dir_path":   "",
		"dir_init_lock": false,
	}
	
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Fid string `json:"fid"`
		} `json:"data"`
	}
	
	err := c.doRequest(ctx, "POST", "https://drive-pc.quark.cn/1/clouddrive/file", params, body, &result)
	if err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	
	if result.Status != 200 {
		return fmt.Errorf("创建目录失败: %s", result.Message)
	}
	
	return nil
}

// TestConnection 测试夸克网盘连接
func (c *QuarkClient) TestConnection(ctx context.Context) error {
	// 测试策略：调用获取文件列表API，验证cookie是否有效
	params := url.Values{
		"pr":               {"ucpro"},
		"fr":               {"pc"},
		"uc_param_str":     {""},
		"pdir_fid":         {"0"}, // 根目录
		"_page":            {"1"},
		"_size":            {"1"}, // 只取1条数据
		"_fetch_total":     {"0"},
		"_fetch_sub_dirs":  {"0"},
		"_sort":            {"file_type:asc,updated_at:desc"},
	}

	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}

	err := c.doRequest(ctx, "GET", "https://drive-pc.quark.cn/1/clouddrive/file/sort", params, nil, &result)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}

	if result.Status != 200 {
		if result.Status == 401 {
			return fmt.Errorf("Cookie已过期或无效，请重新获取")
		}
		return fmt.Errorf("连接失败: %s (状态码:%d)", result.Message, result.Status)
	}

	return nil
}