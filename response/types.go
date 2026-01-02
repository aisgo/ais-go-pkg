package response

/* ========================================================================
 * Response Types - 响应类型定义
 * ========================================================================
 * 职责: 定义标准 API 响应结构
 * ======================================================================== */

// Result 标准 API 响应结构
type Result struct {
	Code int         `json:"code" example:"200" doc:"响应状态码"`
	Msg  string      `json:"msg" example:"success" doc:"响应消息"`
	Data interface{} `json:"data" doc:"响应数据"`
}

// PageResult 分页响应结构
type PageResult struct {
	List     interface{} `json:"list" doc:"数据列表"`
	Total    int64       `json:"total" example:"100" doc:"总记录数"`
	Page     int         `json:"page" example:"1" doc:"当前页码"`
	PageSize int         `json:"page_size" example:"10" doc:"每页大小"`
}

// Response Result 的别名，用于 Swagger 文档
type Response = Result
