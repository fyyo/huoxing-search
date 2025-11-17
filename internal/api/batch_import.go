package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xinyue-go/internal/model"
	"xinyue-go/internal/repository"
)

// BatchImportHandler 批量导入处理器
type BatchImportHandler struct {
	sourceRepo repository.SourceRepository
}

// NewBatchImportHandler 创建批量导入处理器
func NewBatchImportHandler() *BatchImportHandler {
	return &BatchImportHandler{
		sourceRepo: repository.NewSourceRepository(),
	}
}

// ImportRequest 批量导入请求
type ImportRequest struct {
	Content string `json:"content" binding:"required"` // 批量链接内容，每行一个
	PanType int    `json:"pan_type"`                   // 网盘类型：0=夸克 2=百度 3=阿里 4=UC 5=迅雷
	IsTime  int    `json:"is_time"`                    // 是否临时：0=否 1=是
	Status  int    `json:"status"`                     // 状态：0=禁用 1=启用
}

// ImportResponse 导入响应
type ImportResponse struct {
	Total   int      `json:"total"`   // 总数
	Success int      `json:"success"` // 成功数
	Failed  int      `json:"failed"`  // 失败数
	Errors  []string `json:"errors"`  // 错误信息
}

// Import 批量导入
func (h *BatchImportHandler) Import(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 分割内容，每行一个链接
	lines := strings.Split(req.Content, "\n")
	response := ImportResponse{
		Total:  0,
		Success: 0,
		Failed: 0,
		Errors: []string{},
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		response.Total++

		// 解析链接
		url, title := parseLine(line)
		if url == "" {
			response.Failed++
			response.Errors = append(response.Errors, "无效的链接: "+line)
			continue
		}

		// 检查链接是否已存在
		existing, _ := h.sourceRepo.GetByURL(c.Request.Context(), url)
		if existing != nil {
			response.Failed++
			response.Errors = append(response.Errors, "链接已存在: "+url)
			continue
		}

		// 创建资源
		source := &model.Source{
			Title:      title,
			URL:        url,
			Content:    url,
			IsType:     req.PanType,
			IsTime:     req.IsTime,
			Status:     req.Status,
			CreateTime: time.Now().Unix(),
			UpdateTime: time.Now().Unix(),
		}

		if err := h.sourceRepo.Create(c.Request.Context(), source); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, "导入失败: "+url+" - "+err.Error())
			continue
		}

		response.Success++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "导入完成",
		"data":    response,
	})
}

// parseLine 解析一行内容，返回URL和标题
func parseLine(line string) (url string, title string) {
	// 支持格式：
	// 1. 纯链接：https://pan.quark.cn/s/xxx
	// 2. 标题|链接：速度与激情|https://pan.quark.cn/s/xxx
	// 3. 链接|标题：https://pan.quark.cn/s/xxx|速度与激情

	parts := strings.Split(line, "|")
	if len(parts) == 1 {
		// 纯链接
		url = strings.TrimSpace(parts[0])
		title = extractTitleFromURL(url)
	} else if len(parts) >= 2 {
		// 有标题的情况
		part1 := strings.TrimSpace(parts[0])
		part2 := strings.TrimSpace(parts[1])

		// 判断哪个是URL
		if strings.HasPrefix(part1, "http://") || strings.HasPrefix(part1, "https://") {
			url = part1
			title = part2
		} else {
			url = part2
			title = part1
		}
	}

	// 如果没有标题，使用URL作为标题
	if title == "" {
		title = extractTitleFromURL(url)
	}

	return url, title
}

// extractTitleFromURL 从URL中提取标题
func extractTitleFromURL(url string) string {
	// 从URL中提取最后一段作为标题
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}

// GetTemplate 获取导入模板
func (h *BatchImportHandler) GetTemplate(c *gin.Context) {
	template := `# 批量导入模板

# 支持以下格式：
# 1. 纯链接（每行一个）：
https://pan.quark.cn/s/abc123
https://pan.baidu.com/s/def456

# 2. 标题|链接：
速度与激情1-10合集|https://pan.quark.cn/s/abc123
三体全集|https://pan.baidu.com/s/def456

# 3. 链接|标题：
https://pan.quark.cn/s/abc123|速度与激情1-10合集
https://pan.baidu.com/s/def456|三体全集

# 注意事项：
# - 每行一个资源
# - 空行会被忽略
# - 以 # 开头的行会被当作注释忽略
`

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"template": template,
		},
	})
}