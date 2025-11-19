
package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"huoxing-search/internal/model"
	"huoxing-search/internal/pkg/config"
	"huoxing-search/internal/pkg/logger"
	"huoxing-search/internal/repository"
	"huoxing-search/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WechatHandler 微信处理器
type WechatHandler struct {
	configRepo repository.ConfigRepository
}

// NewWechatHandler 创建微信处理器
func NewWechatHandler(configRepo repository.ConfigRepository) *WechatHandler {
	return &WechatHandler{
		configRepo: configRepo,
	}
}

// ============ 微信对话开放平台 ============

// ChatbotMessage 对话平台消息结构
type ChatbotMessage struct {
	XMLName xml.Name `xml:"xml"`
	AppID   string   `xml:"appid"`
	UserID  string   `xml:"userid"`
	Channel string   `xml:"channel"`
	Content struct {
		MsgType string `xml:"msgtype"`
		Msg     string `xml:"msg"`
	} `xml:"content"`
}

// ChatbotResponse 对话平台响应结构
type ChatbotResponse struct {
	XMLName xml.Name `xml:"xml"`
	AppID   string   `xml:"appid"`
	OpenID  string   `xml:"openid"`
	Msg     string   `xml:"msg"`
	Channel string   `xml:"channel"`
}

// ChatbotCallback 处理微信对话开放平台回调
// 回调地址: https://your-domain.com/api/wechat/chatbot/callback
func (h *WechatHandler) ChatbotCallback(c *gin.Context) {
	// 优先从query获取，如果没有则从body获取
	encrypted := c.Query("encrypted")
	if encrypted == "" {
		// 尝试从POST body中获取
		var body struct {
			Encrypted string `json:"encrypted"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			encrypted = body.Encrypted
		}
	}
	
	if encrypted == "" {
		logger.Error("缺少encrypted参数")
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "缺少encrypted参数"})
		return
	}

	// 从数据库获取配置（统一使用 wx_* 前缀）
	ctx := context.Background()
	appID, _ := h.configRepo.Get(ctx, "wx_chat_appid")
	token, _ := h.configRepo.Get(ctx, "wx_chat_token")
	encodingAESKey, _ := h.configRepo.Get(ctx, "wx_chat_aes_key")
	systemName, _ := h.configRepo.Get(ctx, "wx_chatbot_name")
	
	if appID == "" || token == "" || encodingAESKey == "" {
		logger.Error("微信对话平台配置不完整")
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "配置不完整"})
		return
	}

	if systemName == "" {
		systemName = "心悦搜索"
	}

	// 解密消息
	msg, err := h.decryptChatbotMessage(encrypted, encodingAESKey, appID)
	if err != nil {
		logger.Error("解密消息失败", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"code": 200})
		return
	}

	// 创建完整的服务（对话平台支持转存，响应时间无严格限制）
	cacheRepo := repository.NewCacheRepository()
	transferService := service.NewTransferService(config.GlobalConfig)
	searchService := service.NewSearchService(h.configRepo, cacheRepo, transferService)

	// 处理消息并发送回复
	h.processChatbotMessage(msg, appID, token, encodingAESKey, systemName, searchService)

	c.JSON(http.StatusOK, gin.H{"code": 200})
}

// decryptChatbotMessage 解密对话平台消息
func (h *WechatHandler) decryptChatbotMessage(encrypted, encodingAESKey, appID string) (*ChatbotMessage, error) {
	// Base64解码EncodingAESKey
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("解码AES密钥失败: %w", err)
	}

	// Base64解码加密数据
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("解码加密数据失败: %w", err)
	}

	// AES解密
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("密文太短")
	}

	iv := key[:16]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除PKCS7填充
	plaintext := pkcs7Unpad(ciphertext)

	// 提取XML内容
	// 格式: 16位随机字符串 + 4字节消息长度 + XML内容 + AppID
	if len(plaintext) < 20 {
		return nil, fmt.Errorf("解密后数据太短")
	}

	content := plaintext[16:]
	xmlLen := binary.BigEndian.Uint32(content[:4])
	xmlContent := content[4 : 4+xmlLen]

	// 解析XML
	var msg ChatbotMessage
	if err := xml.Unmarshal(xmlContent, &msg); err != nil {
		return nil, fmt.Errorf("解析XML失败: %w", err)
	}

	return &msg, nil
}

// processChatbotMessage 处理对话平台消息
func (h *WechatHandler) processChatbotMessage(msg *ChatbotMessage, appID, token, encodingAESKey, systemName string, searchService *service.SearchService) {
	ctx := context.Background()

	// 如果不是文本消息,发送欢迎语
	if msg.Content.MsgType != "text" || msg.Content.Msg == "" {
		welcomeMsg := buildWelcomeMessage(systemName)
		h.sendChatbotMessage(msg, welcomeMsg, appID, token, encodingAESKey)
		return
	}

	message := strings.TrimSpace(msg.Content.Msg)

	// 检查是否是搜索命令
	if strings.HasPrefix(message, "搜") || strings.HasPrefix(message, "全网搜") {
		var keyword string

		if strings.HasPrefix(message, "全网搜") {
			keyword = strings.TrimPrefix(message, "全网搜")
		} else {
			keyword = strings.TrimPrefix(message, "搜剧")
			keyword = strings.TrimPrefix(keyword, "搜")
		}

		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			h.sendChatbotMessage(msg, "请输入要搜索的关键词哦~", appID, token, encodingAESKey)
			return
		}

		// 先发送"正在搜索"提示
		h.sendChatbotMessage(msg, "正在深入搜索,请稍等...", appID, token, encodingAESKey)

		// 执行搜索（v1.0.10: MaxCount 从 5 改为 10）
		results, err := searchService.Search(ctx, model.SearchRequest{
			Keyword:  keyword,
			PanType:  0, // 默认夸克
			MaxCount: 10,
		})

		if err != nil {
			logger.Error("搜索失败", zap.Error(err))
			h.sendChatbotMessage(msg, "搜索出错了,请稍后再试~", appID, token, encodingAESKey)
			return
		}

		// v1.0.8: 添加 nil 检查
		if results == nil || results.Results == nil {
			h.sendChatbotMessage(msg, "搜索出错了,请稍后再试~", appID, token, encodingAESKey)
			return
		}

		// 构建回复消息
		replyMsg := buildSearchResultMessage(keyword, results.Results)
		
		// v1.0.8: 添加 2000 字符限制
		if len(replyMsg) > 2000 {
			replyMsg = replyMsg[:1997] + "..."
		}
		
		h.sendChatbotMessage(msg, replyMsg, appID, token, encodingAESKey)
	} else {
		// v1.0.8: 非搜索命令时自动发送欢迎消息
		welcomeMsg := buildWelcomeMessage(systemName)
		h.sendChatbotMessage(msg, welcomeMsg, appID, token, encodingAESKey)
	}
}

// buildWelcomeMessage 构建欢迎消息
func buildWelcomeMessage(systemName string) string {
	msg := fmt.Sprintf("欢迎来到 %s！🍿🎬 这里是影迷的梦幻天堂,准备好享受每一部影视的精彩时刻吧!\n\n", systemName)
	msg += "🔍 使用指南 🔍\n\n"
	msg += "1. 搜剧命令\n"
	msg += "回复 \"搜剧+剧名\",免费获取最全的影视资源。\n"
	msg += "示例：<a href='weixin://bizmsgmenu?msgmenucontent=搜剧我被美女包围了&msgmenuid=搜剧我被美女包围了'>搜剧我被美女包围了</a>\n\n"
	msg += "2. 全网搜\n"
	msg += "回复 \"全网搜+关键词\",快速找到全网资源!\n"
	msg += "示例：<a href='weixin://bizmsgmenu?msgmenucontent=全网搜学剪辑&msgmenuid=全网搜学剪辑'>全网搜学剪辑</a>\n\n"
	msg += "赶快准备好你的爆米花,和我们一起开启下一场视觉盛宴吧!🎥"
	return msg
}

// buildSearchResultMessage 构建搜索结果消息
func buildSearchResultMessage(keyword string, results []model.SearchResult) string {
	msg := fmt.Sprintf("🔍 %s 丨搜索结果\n", keyword)

	if len(results) == 0 {
		msg += "\n 未找到,可换个关键词尝试哦~\n"
		msg += " ⚠️宁少写,不多写、错写~"
		return msg
	}

	for _, item := range results {
		msg += "\n　\n"
		msg += fmt.Sprintf("\n%s\n<a href='%s'>%s</a>", item.Title, item.URL, item.URL)
	}

	msg += "\n　\n"
	if keyword != "" {
		msg += fmt.Sprintf("不是短剧?请尝试：<a href='weixin://bizmsgmenu?msgmenucontent=全网搜%s&msgmenuid=全网搜%s'>全网搜%s</a>", keyword, keyword, keyword)
	}

	msg += "\n 欢迎观看!如果喜欢可以喊你的朋友一起来哦"
	return msg
}

// sendChatbotMessage 发送对话平台消息
func (h *WechatHandler) sendChatbotMessage(msg *ChatbotMessage, content, appID, token, encodingAESKey string) {
	// 构建XML响应
	response := ChatbotResponse{
		AppID:   msg.AppID,
		OpenID:  msg.UserID,
		Msg:     content,
		Channel: msg.Channel,
	}

	xmlData, err := xml.Marshal(response)
	if err != nil {
		logger.Error("构建XML响应失败", zap.Error(err))
		return
	}

	// 加密响应
	encrypted, err := encryptChatbotMessage(string(xmlData), appID, encodingAESKey)
	if err != nil {
		logger.Error("加密响应失败", zap.Error(err))
		return
	}

	// 发送到微信服务器
	url := fmt.Sprintf("https://chatbot.weixin.qq.com/openapi/sendmsg/%s", token)
	payload := fmt.Sprintf(`{"encrypt":"%s"}`, encrypted)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		logger.Error("发送消息到微信服务器失败", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	logger.Info("消息发送成功", zap.String("user_id", msg.UserID))
}

// encryptChatbotMessage 加密对话平台消息
func encryptChatbotMessage(text, appID, encodingAESKey string) (string, error) {
	// Base64解码EncodingAESKey
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return "", fmt.Errorf("解码AES密钥失败: %w", err)
	}

	// 生成16位随机字符串
	random := getRandomStr(16)

	// 构建明文: 16位随机字符串 + 4字节消息长度 + 消息内容 + AppID
	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(text)))
	plaintext := []byte(random)
	plaintext = append(plaintext, msgLen...)
	plaintext = append(plaintext, []byte(text)...)
	plaintext = append(plaintext, []byte(appID)...)

	// PKCS7填充
	plaintext = pkcs7Pad(plaintext, 32)

	// AES加密
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	iv := key[:16]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	// Base64编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// ============ 微信公众号 ============

// OfficialAccountVerify 验证微信公众号服务器配置(GET请求)
func (h *WechatHandler) OfficialAccountVerify(c *gin.Context) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	// 从数据库获取Token（统一使用 wx_* 前缀）
	ctx := context.Background()
	token, _ := h.configRepo.Get(ctx, "wx_official_token")
	if token == "" {
		logger.Error("微信公众号Token未配置")
		c.String(http.StatusBadRequest, "Token未配置")
		return
	}

	// 验证签名
	if verifySignature(signature, timestamp, nonce, token) {
		c.String(http.StatusOK, echostr)
	} else {
		c.String(http.StatusUnauthorized, "签名验证失败")
	}
}

// OfficialAccountCallback 处理微信公众号消息回调(POST请求)
func (h *WechatHandler) OfficialAccountCallback(c *gin.Context) {
	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("读取请求体失败", zap.Error(err))
		c.String(http.StatusBadRequest, "读取请求失败")
		return
	}

	// 解析XML消息
	var msg struct {
		XMLName      xml.Name `xml:"xml"`
		ToUserName   string   `xml:"ToUserName"`
		FromUserName string   `xml:"FromUserName"`
		CreateTime   int64    `xml:"CreateTime"`
		MsgType      string   `xml:"MsgType"`
		Content      string   `xml:"Content"`
		MsgId        int64    `xml:"MsgId"`
	}

	if err := xml.Unmarshal(body, &msg); err != nil {
		logger.Error("解析XML失败", zap.Error(err))
		c.String(http.StatusBadRequest, "解析失败")
		return
	}

	// 只处理文本消息
	if msg.MsgType != "text" {
		c.String(http.StatusOK, "success")
		return
	}

	// 检查是否包含"搜"关键字
	if !strings.Contains(msg.Content, "搜") {
		c.String(http.StatusOK, "success")
		return
	}

	// 提取搜索关键词
	keyword := strings.ReplaceAll(msg.Content, "搜剧", "")
	keyword = strings.ReplaceAll(keyword, "搜", "")
	keyword = strings.TrimSpace(keyword)

	if keyword == "" {
		c.String(http.StatusOK, "success")
		return
	}

	// 创建搜索服务（公众号禁用转存，避免超过5秒响应限制）
	cacheRepo := repository.NewCacheRepository()
	searchService := service.NewSearchService(h.configRepo, cacheRepo, nil)

	// 执行搜索（公众号禁用转存，直接返回原始链接）
	ctx := context.Background()
	results, err := searchService.Search(ctx, model.SearchRequest{
		Keyword:  keyword,
		PanType:  0, // 默认夸克
		MaxCount: 5,
	})

	if err != nil {
		logger.Error("搜索失败", zap.Error(err))
		c.String(http.StatusOK, "success")
		return
	}

	// v1.0.8: 添加 nil 检查
	if results == nil || results.Results == nil {
		c.String(http.StatusOK, "success")
		return
	}

	// 构建回复内容
	var replyContent string
	if len(results.Results) == 0 {
		replyContent = "未找到,减少关键词尝试搜索。"
	} else {
		for _, item := range results.Results {
			if replyContent != "" {
				replyContent += "\n" + item.Title + "\n" + item.URL + "\n --------------------"
			} else {
				replyContent = item.Title + "\n" + item.URL + "\n --------------------"
			}
		}
		replyContent += "\n 步骤：点击上方链接-打开网盘-点立即查看-点右下角保存-打开文件-按文件名排序即可从第一集开始-自动-全集播放"
	}

	// 构建回复XML
	replyXML := fmt.Sprintf(`<xml>
<ToUserName><![CDATA[%s]]></ToUserName>
<FromUserName><![CDATA[%s]]></FromUserName>
<CreateTime>%d</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[%s]]></Content>
</xml>`, msg.FromUserName, msg.ToUserName, time.Now().Unix(), replyContent)

	c.Data(http.StatusOK, "application/xml", []byte(replyXML))
}

// verifySignature 验证微信签名
func verifySignature(signature, timestamp, nonce, token string) bool {
	// 将token、timestamp、nonce三个参数进行字典序排序
	arr := []string{token, timestamp, nonce}
	sort.Strings(arr)

	// 将三个参数字符串拼接成一个字符串进行sha1加密
	str := strings.Join(arr, "")
	h := sha1.New()
	h.Write([]byte(str))
	hashCode := fmt.Sprintf("%x", h.Sum(nil))

	// 将加密后的字符串与signature对比
	return hashCode == signature
}

// ============ 辅助函数 ============

// pkcs7Pad PKCS7填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// pkcs7Unpad PKCS7去填充
func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	padding := int(data[length-1])
	if padding < 1 || padding > 32 {
		return data
	}
	return data[:length-padding]
}

// getRandomStr 生成随机字符串
func getRandomStr(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}