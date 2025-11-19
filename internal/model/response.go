package model

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应
type PageResponse struct {
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Data    interface{} `json:"data"`
}

// 响应码常量
const (
	CodeSuccess      = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

// Success 成功响应
func Success(data interface{}) Response {
	return Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	}
}

// SuccessWithMessage 成功响应(自定义消息)
func SuccessWithMessage(message string, data interface{}) Response {
	return Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	}
}

// Error 错误响应
func Error(code int, message string) Response {
	return Response{
		Code:    code,
		Message: message,
	}
}

// BadRequest 400错误
func BadRequest(message string) Response {
	return Response{
		Code:    CodeBadRequest,
		Message: message,
	}
}

// Unauthorized 401错误
func Unauthorized(message string) Response {
	return Response{
		Code:    CodeUnauthorized,
		Message: message,
	}
}

// Forbidden 403错误
func Forbidden(message string) Response {
	return Response{
		Code:    CodeForbidden,
		Message: message,
	}
}

// NotFound 404错误
func NotFound(message string) Response {
	return Response{
		Code:    CodeNotFound,
		Message: message,
	}
}

// ServerError 500错误
func ServerError(message string) Response {
	return Response{
		Code:    CodeServerError,
		Message: message,
	}
}

// PageData 分页数据响应
func PageData(total int64, page, perPage int, data interface{}) Response {
	return Response{
		Code:    CodeSuccess,
		Message: "success",
		Data: PageResponse{
			Total:   total,
			Page:    page,
			PerPage: perPage,
			Data:    data,
		},
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword  string `json:"keyword" binding:"required"`
	PanType  int    `json:"pan_type"` // 0=夸克 2=百度 3=阿里 4=UC 5=迅雷
	MaxCount int    `json:"max_count"` // 最大返回数量
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Total   int            `json:"total"`
	Results []SearchResult `json:"results"`
	Message string         `json:"message,omitempty"`
}

// TransferRequest 转存请求
type TransferRequest struct {
	Items       []SearchResult `json:"items" binding:"required"`
	PanType     int            `json:"pan_type"`     // 0=夸克 2=百度 3=阿里 4=UC 5=迅雷
	MaxCount    int            `json:"max_count"`    // 最多转存成功数量
	MaxDisplay  int            `json:"max_display"`  // 最大展示数量(转存+未转存)
	ExpiredType int            `json:"expired_type"` // 过期类型: 0=永久 2=临时2天
}

// TransferResponse 转存响应
type TransferResponse struct {
	Total       int              `json:"total"`
	Success     int              `json:"success"`
	Failed      int              `json:"failed"`
	Results     []TransferResult `json:"results"`
}