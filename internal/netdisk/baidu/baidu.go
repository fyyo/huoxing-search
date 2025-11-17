package baidu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"xinyue-go/internal/model"
	"xinyue-go/internal/repository"
)

// BaiduClient 百度网盘客户端
type BaiduClient struct {
	cookie     string
	httpClient *http.Client
	bdstoken   string
	configRepo repository.ConfigRepository
}

// NewBaiduClient 创建百度网盘客户端 - 只需要cookie
func NewBaiduClient(cookie string, configRepo repository.ConfigRepository) *BaiduClient {
	return &BaiduClient{
		cookie:     cookie,
		configRepo: configRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Transfer 实现转存功能 - 参考PHP版本BaiduWork.php
func (c *BaiduClient) Transfer(ctx context.Context, shareURL, password string, expiredType int) (*model.TransferResult, error) {
	// 1. 获取bdstoken
	if err := c.getBdstoken(ctx); err != nil {
		return nil, fmt.Errorf("获取bdstoken失败: %w", err)
	}

	// 2. 验证提取码
	if password != "" {
		randsk, err := c.verifyPassCode(ctx, shareURL, password)
		if err != nil {
			return nil, fmt.Errorf("验证提取码失败: %w", err)
		}
		c.updateCookie(randsk)
	}

	// 3. 获取转存参数
	shareID, userID, fsIDs, fileNames, err := c.getTransferParams(ctx, shareURL)
	if err != nil {
		return nil, fmt.Errorf("获取转存参数失败: %w", err)
	}

	// 4. 动态获取转存目录
	folderPath, err := c.getToPdirFid(ctx, expiredType)
	if err != nil {
		return nil, fmt.Errorf("获取转存目录失败: %w", err)
	}

	// 5. 检查并创建目录
	if err := c.ensureDir(ctx, folderPath); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 6. 执行转存
	if err := c.transferFile(ctx, shareID, userID, fsIDs, folderPath); err != nil {
		return nil, fmt.Errorf("转存文件失败: %w", err)
	}

	// 7. 获取转存后的文件列表
	files, err := c.getDirList(ctx, "/"+folderPath)
	if err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %w", err)
	}

	// 8. 动态获取禁用词并过滤广告
	bannedWords := c.getBannedWords(ctx)
	
	// 9. 找到刚转存的文件
	var targetFiles []string
	var targetFsIDs []string
	allAds := true

	for _, file := range files {
		if c.isInFileNames(file.ServerFilename, fileNames) {
			filePath := "/" + folderPath + "/" + file.ServerFilename
			isAd := c.containsAdKeywords(file.ServerFilename, bannedWords)
			
			if !isAd {
				targetFiles = append(targetFiles, filePath)
				targetFsIDs = append(targetFsIDs, fmt.Sprintf("%d", file.FsID))
				allAds = false
			}
		}
	}

	if allAds || len(targetFsIDs) == 0 {
		return nil, fmt.Errorf("资源内容为空或全部为广告")
	}

	// 10. 创建分享链接
	shareLink, sharePassword, expiredType, err := c.createShare(ctx, targetFsIDs, 0)
	if err != nil {
		return nil, fmt.Errorf("创建分享失败: %w", err)
	}

	result := &model.TransferResult{
		Title:       fileNames[0],
		OriginalURL: shareURL,
		ShareURL:    shareLink,
		Password:    sharePassword,
		Success:     true,
		Message:     "转存成功",
		ExpiredType: expiredType, // 使用百度API返回的真实过期类型
	}

	return result, nil
}

// getToPdirFid 动态获取转存目录
func (c *BaiduClient) getToPdirFid(ctx context.Context, expiredType int) (string, error) {
	var configKey string
	if expiredType == 2 {
		configKey = "baidu_file_time" // 临时资源路径
	} else {
		configKey = "baidu_file" // 默认存储路径
	}
	
	folderPath, err := c.configRepo.Get(ctx, configKey)
	if err != nil {
		return "", fmt.Errorf("读取配置%s失败: %w", configKey, err)
	}
	
	if folderPath == "" {
		return "xinyue", nil // 默认目录
	}
	
	// ⚠️ 关键修复：移除开头的斜杠（如果有），因为后续会统一添加
	folderPath = strings.TrimPrefix(folderPath, "/")
	
	return folderPath, nil
}

// getBannedWords 动态获取禁用词列表
func (c *BaiduClient) getBannedWords(ctx context.Context) []string {
	bannedStr, err := c.configRepo.Get(ctx, "quark_banned")
	if err != nil || bannedStr == "" {
		return []string{}
	}
	
	return strings.Split(bannedStr, ",")
}

// getBdstoken 获取bdstoken - 兼容百度API的动态返回格式
func (c *BaiduClient) getBdstoken(ctx context.Context) error {
	url := "https://pan.baidu.com/api/gettemplatevariable"
	params := map[string]string{
		"clienttype": "0",
		"app_id":     "38824127",
		"web":        "1",
		"fields":     `["bdstoken","token","uk","isdocuser","servertime"]`,
	}

	fmt.Printf("🔍 [DEBUG] getBdstoken - 请求参数:\n")
	fmt.Printf("  URL: %s\n", url)
	fmt.Printf("  Params: %+v\n", params)
	fmt.Printf("  Cookie长度: %d字符\n", len(c.cookie))
	fmt.Printf("  Cookie前50字符: %s\n", c.cookie[:min(50, len(c.cookie))])

	// 使用map接收，因为result字段可能是对象或数组
	var result map[string]interface{}

	err := c.requestWithRetry(ctx, "GET", url, params, nil, &result)
	if err != nil {
		fmt.Printf("🔍 [DEBUG] getBdstoken - 请求失败: %v\n", err)
		return fmt.Errorf("获取bdstoken失败: %w", err)
	}

	fmt.Printf("🔍 [DEBUG] getBdstoken - 响应内容:\n")
	fmt.Printf("  完整响应: %+v\n", result)

	// 检查errno
	errno, _ := result["errno"].(float64)
	fmt.Printf("🔍 [DEBUG] getBdstoken - errno: %d\n", int(errno))
	
	if int(errno) != 0 {
		return fmt.Errorf("获取bdstoken失败,错误码: %d", int(errno))
	}

	// 尝试从result中提取bdstoken
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		fmt.Printf("🔍 [DEBUG] getBdstoken - result是对象: %+v\n", resultData)
		if bdstoken, ok := resultData["bdstoken"].(string); ok && bdstoken != "" {
			c.bdstoken = bdstoken
			fmt.Printf("🔍 [DEBUG] getBdstoken - 成功提取bdstoken: %s\n", bdstoken[:min(10, len(bdstoken))])
			return nil
		}
	} else {
		fmt.Printf("🔍 [DEBUG] getBdstoken - result不是对象，类型: %T, 值: %+v\n", result["result"], result["result"])
	}

	return fmt.Errorf("无法从响应中提取bdstoken")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// verifyPassCode 验证提取码 - 完全对齐PHP版本
func (c *BaiduClient) verifyPassCode(ctx context.Context, shareURL, password string) (string, error) {
	// 先移除URL参数（如?pwd=xxxx）
	baseURL := shareURL
	if idx := strings.Index(shareURL, "?"); idx != -1 {
		baseURL = shareURL[:idx]
	}
	
	// ⚠️ 完全对齐PHP: substr($linkUrl, 25, 23)
	// PHP固定从第25位取23个字符
	if len(baseURL) < 25 {
		return "", fmt.Errorf("分享链接格式错误: %s", baseURL)
	}
	
	// 取23个字符（如果不足23个就取到末尾）
	surl := ""
	if len(baseURL) >= 48 {
		surl = baseURL[25:48] // 取23个字符
	} else {
		surl = baseURL[25:] // 取到末尾
	}
	
	url := "https://pan.baidu.com/share/verify"
	params := map[string]string{
		"surl":       surl,
		"bdstoken":   c.bdstoken,
		"t":          fmt.Sprintf("%d", time.Now().UnixMilli()),
		"channel":    "chunlei",
		"web":        "1",
		"clienttype": "0",
		// ⚠️ PHP版本没有app_id参数
	}

	data := map[string]string{
		"pwd":       password,
		"vcode":     "",
		"vcode_str": "",
	}

	var result struct {
		Errno  int    `json:"errno"`
		Randsk string `json:"randsk"`
	}

	fmt.Printf("🔍 [DEBUG] verifyPassCode参数:\n")
	fmt.Printf("  - shareURL: %s\n", shareURL)
	fmt.Printf("  - password: %s\n", password)
	fmt.Printf("  - surl: %s (长度:%d)\n", surl, len(surl))
	fmt.Printf("  - bdstoken: %s\n", c.bdstoken[:min(10, len(c.bdstoken))])
	fmt.Printf("  - cookie长度: %d字符\n", len(c.cookie))

	err := c.requestWithRetry(ctx, "POST", url, params, data, &result)
	if err != nil {
		fmt.Printf("🔍 [DEBUG] verifyPassCode请求失败: %v\n", err)
		return "", fmt.Errorf("验证提取码失败: %w", err)
	}

	fmt.Printf("🔍 [DEBUG] verifyPassCode返回: errno=%d\n", result.Errno)
	
	if result.Errno != 0 {
		return "", fmt.Errorf("验证提取码失败,错误码: %d", result.Errno)
	}

	fmt.Printf("🔍 [DEBUG] 成功获取randsk: %s (前10字符)\n", result.Randsk[:min(10, len(result.Randsk))])
	return result.Randsk, nil
}

// getTransferParams 获取转存参数 - 参考PHP版本BaiduWork.php的parseResponse实现
func (c *BaiduClient) getTransferParams(ctx context.Context, shareURL string) (string, string, []string, []string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", shareURL, nil)
	if err != nil {
		return "", "", nil, nil, err
	}

	// ⚠️ 必须使用更新后的cookie（包含BDCLND=randsk）
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://pan.baidu.com/disk/main")
	req.Header.Set("Cookie", c.cookie) // 使用c.cookie而不是setHeaders，因为cookie已经包含了randsk
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	// ⚠️ 不要手动设置Accept-Encoding，让Go自动处理gzip解压
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, nil, err
	}

	bodyStr := string(body)
	
	fmt.Printf("🔍 [DEBUG] getTransferParams - HTML长度: %d字符\n", len(bodyStr))
	fmt.Printf("🔍 [DEBUG] getTransferParams - 是否包含shareid: %v\n", strings.Contains(bodyStr, "shareid"))
	fmt.Printf("🔍 [DEBUG] getTransferParams - 是否包含share_uk: %v\n", strings.Contains(bodyStr, "share_uk"))
	fmt.Printf("🔍 [DEBUG] getTransferParams - 是否包含fs_id: %v\n", strings.Contains(bodyStr, "fs_id"))
	
	// ⚠️ 关键修复：完全对齐PHP版本的正则表达式（BaiduWork.php 第308-312行）
	patterns := map[string]string{
		"shareid":         `"shareid":(\d+?),"`,           // 修复：添加结尾的引号和逗号
		"user_id":         `"share_uk":"(\d+?)",`,         // 修复：添加结尾的逗号
		"fs_id":           `"fs_id":(\d+?),`,              // 保持不变
		"server_filename": `"server_filename":"(.+?)",`,   // 修复：添加结尾的逗号
		"isdir":           `"isdir":(\d+?),`,              // 保持不变
	}
	
	results := make(map[string][]string)
	
	// 提取所有匹配项
	for key, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(bodyStr, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				results[key] = append(results[key], match[1])
			}
		}
	}
	
	fmt.Printf("🔍 [DEBUG] getTransferParams - 提取结果:\n")
	fmt.Printf("  shareid数量: %d\n", len(results["shareid"]))
	fmt.Printf("  user_id数量: %d\n", len(results["user_id"]))
	fmt.Printf("  fs_id数量: %d\n", len(results["fs_id"]))
	fmt.Printf("  server_filename数量: %d\n", len(results["server_filename"]))
	
	// 验证是否获取到所有必要参数
	if len(results["shareid"]) == 0 || len(results["user_id"]) == 0 ||
	   len(results["fs_id"]) == 0 || len(results["server_filename"]) == 0 {
		// 保存HTML到文件以便调试
		fmt.Printf("🔍 [DEBUG] HTML前500字符: %s\n", bodyStr[:min(500, len(bodyStr))])
		return "", "", nil, nil, fmt.Errorf("未能从HTML中提取到完整的转存参数")
	}
	
	shareID := results["shareid"][0]
	userID := results["user_id"][0]
	fsIDs := results["fs_id"]
	
	fmt.Printf("🔍 [DEBUG] getTransferParams成功 - shareID:%s, userID:%s, fsIDs数量:%d\n", shareID, userID, len(fsIDs))
	
	// 文件名去重
	fileNameMap := make(map[string]bool)
	var fileNames []string
	for _, name := range results["server_filename"] {
		if !fileNameMap[name] {
			fileNameMap[name] = true
			fileNames = append(fileNames, name)
		}
	}
	
	return shareID, userID, fsIDs, fileNames, nil
}

// transferFile 转存文件 - 对齐PHP版本BaiduWork.php的transfer方法
func (c *BaiduClient) transferFile(ctx context.Context, shareID, userID string, fsIDs []string, toPath string) error {
	params := url.Values{
		"shareid":     {shareID},
		"from":        {userID},
		"ondup":       {"newcopy"},
		"async":       {"1"},
		"channel":     {"chunlei"},
		"web":         {"1"},
		"app_id":      {"250528"},
		"bdstoken":    {c.bdstoken},
		"logid":       {""},
		"clienttype":  {"0"},
	}

	// ⚠️ 关键修复：fsidlist必须是JSON数组格式，path必须以/开头
	// PHP版本: 'fsidlist' => '[' . implode(',', $fs_ids) . ']'
	// PHP版本: 'path' => '/' . $folder_path
	body := map[string]interface{}{
		"fsidlist":  "[" + strings.Join(fsIDs, ",") + "]",  // JSON数组格式
		"path":      "/" + toPath,                          // 绝对路径
	}

	fmt.Printf("🔍 [DEBUG] transferFile参数:\n")
	fmt.Printf("  - shareID: %s\n", shareID)
	fmt.Printf("  - userID: %s\n", userID)
	fmt.Printf("  - fsidlist: %s\n", body["fsidlist"])
	fmt.Printf("  - path: %s\n", body["path"])

	return c.doPost(ctx, "https://pan.baidu.com/share/transfer", params, body)
}

// getDirList 获取目录列表
func (c *BaiduClient) getDirList(ctx context.Context, dir string) ([]FileInfo, error) {
	params := url.Values{
		"order":      {"name"},
		"desc":       {"0"},
		"showempty":  {"0"},
		"web":        {"1"},
		"page":       {"1"},
		"num":        {"100"},
		"dir":        {dir},
		"t":          {fmt.Sprintf("%d", time.Now().UnixMilli())},
		"channel":    {"chunlei"},
		"app_id":     {"250528"},
		"bdstoken":   {c.bdstoken},
		"logid":      {""},
		"clienttype": {"0"},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://pan.baidu.com/api/list?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Errno int        `json:"errno"`
		List  []FileInfo `json:"list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Errno != 0 {
		return nil, fmt.Errorf("获取目录列表失败,错误码: %d", result.Errno)
	}

	return result.List, nil
}

// ensureDir 确保目录存在
func (c *BaiduClient) ensureDir(ctx context.Context, path string) error {
	// 先尝试列出目录
	_, err := c.getDirList(ctx, "/"+path)
	if err == nil {
		return nil // 目录已存在
	}

	// 目录不存在,创建它
	return c.createDir(ctx, path)
}

// createDir 创建目录
func (c *BaiduClient) createDir(ctx context.Context, path string) error {
	params := url.Values{
		"a":          {"commit"},
		"channel":    {"chunlei"},
		"web":        {"1"},
		"app_id":     {"250528"},
		"bdstoken":   {c.bdstoken},
		"logid":      {""},
		"clienttype": {"0"},
	}

	body := map[string]interface{}{
		"path":   "/" + path,
		"isdir":  1,
		"block_list": "[]",
	}

	return c.doPost(ctx, "https://pan.baidu.com/api/create", params, body)
}

// createShare 创建分享 - 修复:使用form-urlencoded编码，并返回过期类型
func (c *BaiduClient) createShare(ctx context.Context, fsIDs []string, period int) (string, string, int, error) {
	params := url.Values{
		"channel":    {"chunlei"},
		"web":        {"1"},
		"app_id":     {"250528"},
		"bdstoken":   {c.bdstoken},
		"logid":      {""},
		"clienttype": {"0"},
	}

	password := "6666" // 固定提取码
	body := map[string]interface{}{
		"fid_list":       "[" + strings.Join(fsIDs, ",") + "]",
		"schannel":       4,
		"channel_list":   "[]",
		"period":         period,
		"pwd":            password,
	}

	// ✅ 使用doPost方法，自动处理form编码
	if err := c.doPost(ctx, "https://pan.baidu.com/share/set", params, body); err != nil {
		return "", "", 0, err
	}

	// ⚠️ doPost只检查errno，我们需要获取link和expiredType，所以需要独立实现
	// 将body转换为form编码
	formData := url.Values{}
	for key, value := range body {
		formData.Set(key, fmt.Sprintf("%v", value))
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://pan.baidu.com/share/set?"+params.Encode(),
		strings.NewReader(formData.Encode()))
	if err != nil {
		return "", "", 0, err
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, err
	}

	fmt.Printf("🔍 [DEBUG] createShare响应:\n")
	fmt.Printf("  - 状态码: %d\n", resp.StatusCode)
	fmt.Printf("  - 响应体: %s\n", string(respBody))

	var result struct {
		Errno       int    `json:"errno"`
		Link        string `json:"link"`
		ExpiredType int    `json:"expiredType"` // 0=永久 1=7天 2=1天
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", 0, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	if result.Errno != 0 {
		return "", "", 0, fmt.Errorf("创建分享失败,错误码: %d", result.Errno)
	}

	return result.Link + "?pwd=" + password, password, result.ExpiredType, nil
}

// 辅助方法

func (c *BaiduClient) setHeaders(req *http.Request) {
	// ⚠️ 对齐PHP版本的Headers，但不设置Accept-Encoding让Go自动处理gzip
	req.Header.Set("Host", "pan.baidu.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Referer", "https://pan.baidu.com")
	// ⚠️ 不要手动设置Accept-Encoding，让Go的http.Client自动处理gzip解压
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7,en-GB;q=0.6,ru;q=0.5")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", c.cookie)
}

// updateCookie 更新Cookie - 完全对齐PHP版本的updateCookie方法
func (c *BaiduClient) updateCookie(randsk string) {
	// PHP版本: 将cookie解析为字典，更新BDCLND，再重组
	// 参考 BaiduWork.php 第278-302行
	
	// 1. 拆分cookie为键值对
	cookieMap := make(map[string]string)
	pairs := strings.Split(c.cookie, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			cookieMap[parts[0]] = parts[1]
		}
	}
	
	// 2. 更新或添加BDCLND
	cookieMap["BDCLND"] = randsk
	
	// 3. 重新构建cookie字符串
	var cookieParts []string
	for key, value := range cookieMap {
		cookieParts = append(cookieParts, key+"="+value)
	}
	
	c.cookie = strings.Join(cookieParts, "; ")
	
	fmt.Printf("🔍 [DEBUG] updateCookie成功 - 新cookie长度: %d字符\n", len(c.cookie))
}

func (c *BaiduClient) extractValue(text, startStr, endStr string) string {
	start := strings.Index(text, startStr)
	if start == -1 {
		return ""
	}
	start += len(startStr)
	end := strings.Index(text[start:], endStr)
	if end == -1 {
		return ""
	}
	return strings.Trim(text[start:start+end], `"`)
}

func (c *BaiduClient) isInFileNames(name string, names []string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func (c *BaiduClient) containsAdKeywords(filename string, bannedWords []string) bool {
	lower := strings.ToLower(filename)
	for _, keyword := range bannedWords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (c *BaiduClient) doPost(ctx context.Context, apiURL string, params url.Values, body interface{}) error {
	// ⚠️ 关键修复：使用form-urlencoded编码，完全对齐PHP版本
	// PHP: curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($data));
	
	// 将body转换为url.Values
	formData := url.Values{}
	if bodyMap, ok := body.(map[string]interface{}); ok {
		for key, value := range bodyMap {
			formData.Set(key, fmt.Sprintf("%v", value))
		}
	}
	
	// 使用form编码
	encodedBody := formData.Encode()
	
	fmt.Printf("🔍 [DEBUG] doPost详情:\n")
	fmt.Printf("  - URL: %s?%s\n", apiURL, params.Encode())
	fmt.Printf("  - Body编码: %s\n", encodedBody)
	fmt.Printf("  - Cookie长度: %d\n", len(c.cookie))
	
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL+"?"+params.Encode(), strings.NewReader(encodedBody))
	if err != nil {
		return err
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	fmt.Printf("🔍 [DEBUG] doPost响应:\n")
	fmt.Printf("  - 状态码: %d\n", resp.StatusCode)
	fmt.Printf("  - 响应体: %s\n", string(respBody))

	var result struct {
		Errno int `json:"errno"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	if result.Errno != 0 {
		return fmt.Errorf("请求失败,错误码: %d", result.Errno)
	}

	return nil
}

// requestWithRetry 带重试机制的HTTP请求 - 参考PHP原版
func (c *BaiduClient) requestWithRetry(ctx context.Context, method, url string, params, data map[string]string, result interface{}) error {
	maxRetries := 3
	
	for retry := 0; retry < maxRetries; retry++ {
		// 如果是重试，添加随机延迟 (1-2秒)
		if retry > 0 {
			delay := time.Duration(1000+rand.Intn(1000)) * time.Millisecond
			time.Sleep(delay)
		}
		
		err := c.doRequest(ctx, method, url, params, data, result)
		if err == nil {
			return nil
		}
		
		// 如果是最后一次重试，返回错误
		if retry == maxRetries-1 {
			return err
		}
	}
	
	return fmt.Errorf("请求失败，已重试%d次", maxRetries)
}

// doRequest 执行HTTP请求
func (c *BaiduClient) doRequest(ctx context.Context, method, urlStr string, params, data map[string]string, result interface{}) error {
	// 构建URL参数
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		urlStr += "?" + values.Encode()
	}
	
	var req *http.Request
	var err error
	
	if method == "POST" && data != nil {
		// POST请求，数据放在body中
		values := url.Values{}
		for k, v := range data {
			values.Add(k, v)
		}
		req, err = http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(values.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		// GET请求
		req, err = http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			return err
		}
	}
	
	c.setHeaders(req)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	// 检查是否返回HTML(验证码页面)
	bodyStr := string(body)
	if strings.Contains(bodyStr, "<html") || strings.Contains(bodyStr, "<!DOCTYPE") {
		return fmt.Errorf("触发百度安全验证，请稍后重试或更新Cookie")
	}
	
	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}
	
	return nil
}

// 数据结构

type FileInfo struct {
	FsID           int64  `json:"fs_id"`
	ServerFilename string `json:"server_filename"`
	IsDir          int    `json:"isdir"`
}

// GetName 获取网盘名称
func (c *BaiduClient) GetName() string {
	return "百度网盘"
}

// IsConfigured 检查是否已配置 - 实时从数据库读取
func (c *BaiduClient) IsConfigured() bool {
	// 先检查初始化时的cookie
	if c.cookie != "" {
		return true
	}
	
	// 如果初始化时没有cookie，尝试从数据库读取最新配置
	if c.configRepo != nil {
		ctx := context.Background()
		conf, err := c.configRepo.GetByName(ctx, "baidu_cookie")
		if err == nil && conf != nil && conf.Value != "" {
			// 更新内存中的cookie
			c.cookie = conf.Value
			return true
		}
	}
	
	return false
}

// DeleteDirectory 删除指定目录
func (c *BaiduClient) DeleteDirectory(ctx context.Context, dirPath string) error {
	// 1. 获取bdstoken
	if err := c.getBdstoken(ctx); err != nil {
		return fmt.Errorf("获取bdstoken失败: %w", err)
	}
	
	// 2. 列出根目录找到目标目录
	files, err := c.getDirList(ctx, "/")
	if err != nil {
		return fmt.Errorf("列出根目录失败: %w", err)
	}
	
	var targetPath string
	for _, file := range files {
		if file.ServerFilename == dirPath && file.IsDir == 1 {
			targetPath = "/" + dirPath
			break
		}
	}
	
	if targetPath == "" {
		return fmt.Errorf("目录不存在: %s", dirPath)
	}
	
	// 3. 删除目录
	params := url.Values{
		"opera":      {"delete"},
		"async":      {"0"},
		"onnest":     {"fail"},
		"channel":    {"chunlei"},
		"web":        {"1"},
		"app_id":     {"250528"},
		"bdstoken":   {c.bdstoken},
		"logid":      {""},
		"clienttype": {"0"},
	}
	
	body := map[string]interface{}{
		"filelist": "[\"" + targetPath + "\"]",
	}
	
	return c.doPost(ctx, "https://pan.baidu.com/api/filemanager", params, body)
}

// CreateDirectory 创建指定目录
func (c *BaiduClient) CreateDirectory(ctx context.Context, dirPath string) error {
	// 1. 获取bdstoken
	if err := c.getBdstoken(ctx); err != nil {
		return fmt.Errorf("获取bdstoken失败: %w", err)
	}
	
	// 2. 创建目录
	return c.createDir(ctx, dirPath)
}

// TestConnection 测试百度网盘连接 - 兼容百度API的动态返回格式
func (c *BaiduClient) TestConnection(ctx context.Context) error {
	url := "https://pan.baidu.com/api/gettemplatevariable"
	params := map[string]string{
		"clienttype": "0",
		"app_id":     "38824127",
		"web":        "1",
		"fields":     `["bdstoken","token","uk","isdocuser","servertime"]`,
	}

	// 使用map接收，因为result字段可能是对象或数组
	var result map[string]interface{}

	err := c.doRequest(ctx, "GET", url, params, nil, &result)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}

	// 检查errno
	errno, ok := result["errno"].(float64)
	if !ok {
		return fmt.Errorf("响应格式错误，无法获取errno")
	}

	if int(errno) != 0 {
		if int(errno) == -6 {
			return fmt.Errorf("Cookie已过期或无效，请重新登录")
		}
		return fmt.Errorf("连接失败，错误码: %d", int(errno))
	}

	// result字段可能是对象或数组，只要errno=0就认为连接成功
	// 尝试提取bdstoken验证（可选）
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		if bdstoken, ok := resultData["bdstoken"].(string); ok && bdstoken != "" {
			return nil // 成功获取到bdstoken
		}
	}

	// 即使result是数组或其他格式，只要errno=0就认为Cookie有效
	return nil
}