
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
	"sync"
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
	configRepo     repository.ConfigRepository
	processingMsgs sync.Map // 消息去重: msgID -> 处理时间
}

// NewWechatHandler 创建微信处理器
func NewWechatHandler(configRepo repository.ConfigRepository) *WechatHandler {
	handler := &WechatHandler{
		configRepo: configRepo,
	}
	// 启动清理过期消息ID的协程
	go handler.cleanupExpiredMessages()
	return handler
}

// cleanupExpiredMessages 清理过期的消息ID（每分钟清理一次）
func (h *WechatHandler) cleanupExpiredMessages() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		now := time.Now()
		h.processingMsgs.Range(func(key, value interface{}) bool {
			keyStr := key.(string)
			processTime := value.(time.Time)
			
			// 🔥 区分消息去重ID和欢迎消息标记
			// 消息去重ID格式: "user:type:content"
			// 欢迎消息标记格式: "chatbot_welcome:user"
			
			if strings.HasPrefix(keyStr, "chatbot_welcome:") {
				// ✅ 欢迎消息标记：永不清理（用户首次访问后永久记录）
				// 除非服务重启，否则不会再次发送欢迎消息
				return true
			} else {
				// 消息去重ID：5分钟后清理（允许用户重新搜索相同内容）
				if now.Sub(processTime) > 5*time.Minute {
					h.processingMsgs.Delete(key)
				}
			}
			return true
		})
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
	
	// 从数据库获取配置（统一使用 wx_* 前缀）
	ctx := context.Background()
	appID, _ := h.configRepo.Get(ctx, "wx_chat_appid")
	token, _ := h.configRepo.Get(ctx, "wx_chat_token")
	encodingAESKey, _ := h.configRepo.Get(ctx, "wx_chat_aes_key")
	systemName, _ := h.configRepo.Get(ctx, "wx_chatbot_name")
	
	// 如果encrypted为空（首次接入或测试请求）
	if encrypted == "" {
		logger.Info("收到空encrypted请求（可能是首次接入或测试）")
		// 返回成功，不报错（与PHP版本一致）
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
		})
		return
	}
	
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

	// 🔥 消息去重：生成唯一标识
	msgID := fmt.Sprintf("%s:%s:%s", msg.UserID, msg.Content.MsgType, msg.Content.Msg)
	
	// 检查是否正在处理相同消息
	if _, exists := h.processingMsgs.LoadOrStore(msgID, time.Now()); exists {
		logger.Info("⚠️ 检测到重复消息，忽略",
			zap.String("user_id", msg.UserID),
			zap.String("msg", msg.Content.Msg))
		// 立即返回成功，避免微信重试
		c.JSON(http.StatusOK, gin.H{"code": 200})
		return
	}

	// ⚡ 立即响应微信服务器（<1秒），避免触发重试
	c.JSON(http.StatusOK, gin.H{"code": 200})

	// 🚀 异步处理消息（在后台执行搜索和转存）
	go func() {
		defer func() {
			// 处理完成后3分钟删除消息ID（防止用户短时间内重复搜索相同内容）
			time.AfterFunc(3*time.Minute, func() {
				h.processingMsgs.Delete(msgID)
			})
		}()

		// 创建完整的服务（对话平台支持转存）
		cacheRepo := repository.NewCacheRepository()
		transferService := service.NewTransferService(config.GlobalConfig)
		searchService := service.NewSearchService(h.configRepo, cacheRepo, transferService)

		// 处理消息并发送回复
		h.processChatbotMessage(msg, appID, token, encodingAESKey, systemName, searchService)
	}()
}

// decryptChatbotMessage 解密对话平台消息
// 参考PHP Chatbot.php第206-260行的decrypt()和decode()函数
func (h *WechatHandler) decryptChatbotMessage(encrypted, encodingAESKey, appID string) (*ChatbotMessage, error) {
	// EncodingAESKey是43位字符，Base64解码后是32字节
	// PHP代码添加"="是为了补齐Base64，Go的StdEncoding会自动处理
	var aesKey []byte
	var err error
	
	// 先尝试直接解码
	aesKey, err = base64.StdEncoding.DecodeString(encodingAESKey)
	if err != nil {
		// 如果失败，尝试添加"="填充
		aesKey, err = base64.StdEncoding.DecodeString(encodingAESKey + "=")
		if err != nil {
			return nil, fmt.Errorf("解码AES密钥失败: %w", err)
		}
	}
	
	// 验证密钥长度
	if len(aesKey) < 32 {
		return nil, fmt.Errorf("AES密钥长度不足: %d字节，需要至少32字节", len(aesKey))
	}

	// 解码密文 (PHP第211行openssl_decrypt会自动Base64解码)
	cipherData, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("解码密文失败: %w", err)
	}

	// 检查长度
	if len(cipherData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度无效: %d", len(cipherData))
	}

	// 创建AES cipher
	block, err := aes.NewCipher(aesKey[:32])
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// IV = aesKey的前16字节 (PHP第210行: $iv = substr($this->key, 0, 16))
	iv := aesKey[:16]

	// 解密 (PHP第211行: openssl_decrypt with OPENSSL_ZERO_PADDING)
	plaintext := make([]byte, len(cipherData))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, cipherData)

	// 去除PKCS7填充 (PHP第231行: $result = $this->decode($decrypted))
	// decode函数在267-275行
	plaintext = removePKCS7Padding(plaintext)

	// 提取XML内容 (PHP第233-239行)
	if len(plaintext) < 20 { // 16字节随机数 + 4字节长度
		return nil, fmt.Errorf("解密后数据太短: %d字节", len(plaintext))
	}

	// 跳过16字节随机数 (PHP第235行)
	content := plaintext[16:]

	// 读取4字节网络字节序的XML长度 (PHP第236行)
	xmlLen := binary.BigEndian.Uint32(content[:4])
	if int(xmlLen) > len(content)-4 || xmlLen == 0 {
		return nil, fmt.Errorf("XML长度异常: %d, 可用: %d", xmlLen, len(content)-4)
	}

	// 提取XML内容 (PHP第238行)
	xmlData := content[4 : 4+xmlLen]

	// 提取AppID (PHP第239行)
	receivedAppID := string(content[4+xmlLen:])

	// 验证AppID (PHP第250行)
	if strings.TrimSpace(receivedAppID) != appID {
		logger.Warn("AppID不匹配",
			zap.String("expected", appID),
			zap.String("received", receivedAppID))
	}

	// 解析XML
	var msg ChatbotMessage
	if err := xml.Unmarshal(xmlData, &msg); err != nil {
		return nil, fmt.Errorf("解析XML失败: %w", err)
	}

	return &msg, nil
}

// removePKCS7Padding 去除PKCS7填充
// 对应PHP Chatbot.php第267-275行的decode()函数
func removePKCS7Padding(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}

	// PHP第270行: $pad = ord(substr($text, -1));
	paddingLen := int(data[length-1])

	// PHP第271-273行: if ($pad < 1 || $pad > 32) { $pad = 0; }
	if paddingLen < 1 || paddingLen > 32 {
		paddingLen = 0
	}

	// PHP第274行: return substr($text, 0, (strlen($text) - $pad));
	return data[:length-paddingLen]
}

// processChatbotMessage 处理对话平台消息
func (h *WechatHandler) processChatbotMessage(msg *ChatbotMessage, appID, token, encodingAESKey, systemName string, searchService *service.SearchService) {
	ctx := context.Background()

	// 如果不是文本消息或消息为空，发送欢迎消息（但每个用户只发一次）
	if msg.Content.MsgType != "text" || strings.TrimSpace(msg.Content.Msg) == "" {
		// 检查该用户是否已经收到过欢迎消息
		welcomeKey := fmt.Sprintf("chatbot_welcome:%s", msg.UserID)
		if _, alreadySent := h.processingMsgs.Load(welcomeKey); !alreadySent {
			logger.Info("✅ 首次访问，发送欢迎消息",
				zap.String("user_id", msg.UserID),
				zap.String("msg_type", msg.Content.MsgType))
			
			// 发送欢迎消息
			welcomeMsg := buildWelcomeMessage(systemName)
			h.sendChatbotMessage(msg, welcomeMsg, appID, token, encodingAESKey)
			
			// ✅ 永久记录（除非服务重启，否则该用户不会再收到欢迎消息）
			h.processingMsgs.Store(welcomeKey, time.Now())
		} else {
			logger.Info("⏭️ 用户已收到欢迎消息，忽略后续空消息",
				zap.String("user_id", msg.UserID))
		}
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

		// 执行搜索（限制10条避免消息太长）
		results, err := searchService.Search(ctx, model.SearchRequest{
			Keyword:  keyword,
			PanType:  0, // 默认夸克
			MaxCount: 10, // 微信对话平台最多返回10条
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

	// 执行搜索（公众号禁用转存，限制10条避免消息太长）
	ctx := context.Background()
	results, err := searchService.Search(ctx, model.SearchRequest{
		Keyword:  keyword,
		PanType:  0, // 默认夸克
		MaxCount: 10, // 微信公众号最多返回10条
	})

	if err != nil {
		logger.Error("搜索失败", zap.Error(err))
		// 返回错误提示给用户
		replyContent := "搜索出错了，请稍后再试~"
		replyXML := fmt.Sprintf(`<xml>
<ToUserName><![CDATA[%s]]></ToUserName>
<FromUserName><![CDATA[%s]]></FromUserName>
<CreateTime>%d</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[%s]]></Content>
</xml>`, msg.FromUserName, msg.ToUserName, time.Now().Unix(), replyContent)
		c.Data(http.StatusOK, "application/xml", []byte(replyXML))
		return
	}

	// v1.0.8: 添加 nil 检查
	if results == nil || results.Results == nil {
		replyContent := "搜索出错了，请稍后再试~"
		replyXML := fmt.Sprintf(`<xml>
<ToUserName><![CDATA[%s]]></ToUserName>
<FromUserName><![CDATA[%s]]></FromUserName>
<CreateTime>%d</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[%s]]></Content>
</xml>`, msg.FromUserName, msg.ToUserName, time.Now().Unix(), replyContent)
		c.Data(http.StatusOK, "application/xml", []byte(replyXML))
		return
	}

	// 构建回复内容（限制最多10条）
	var replyContent string
	if len(results.Results) == 0 {
		replyContent = "未找到,减少关键词尝试搜索。"
	} else {
		// 限制最多返回10条结果
		maxResults := 10
		if len(results.Results) > maxResults {
			results.Results = results.Results[:maxResults]
		}
		
		for i, item := range results.Results {
			if i > 0 {
				replyContent += "\n"
			}
			replyContent += item.Title + "\n" + item.URL
			if i < len(results.Results)-1 {
				replyContent += "\n--------------------"
			}
		}
		replyContent += "\n\n步骤：点击上方链接-打开网盘-点立即查看-点右下角保存-打开文件-按文件名排序即可从第一集开始-自动-全集播放"
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

// getRandomStr 生成随机字符串
func getRandomStr(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}