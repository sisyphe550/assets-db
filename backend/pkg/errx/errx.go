// Package errx 统一错误码定义与 HTTP/gRPC 映射
// 完整错误码矩阵见 doc/06-error-codes.md
package errx

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BizError 业务错误
type BizError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	HTTP    int    `json:"-"`
}

func (e *BizError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// New 创建 BizError
func New(code int, httpStatus int, message string) *BizError {
	return &BizError{Code: code, HTTP: httpStatus, Message: message}
}

// ==========================================
// 全局错误码常量（与 06-error-codes.md §2 对齐）
// ==========================================

var (
	// 400xx 请求参数错误
	ErrInvalidParam       = New(40001, http.StatusBadRequest, "请求参数无效")
	ErrInvalidPagination  = New(40002, http.StatusBadRequest, "分页参数无效")
	ErrInvalidTimeFormat  = New(40003, http.StatusBadRequest, "时间格式无效")

	// 401xx 认证失败
	ErrUnauthenticated = New(40101, http.StatusUnauthorized, "未登录或凭证无效")
	ErrTokenRevoked    = New(40102, http.StatusUnauthorized, "凭证已撤销")

	// 403xx 权限/越权
	ErrForbidden         = New(40301, http.StatusForbidden, "无操作权限")
	ErrDeptAccessDenied  = New(40302, http.StatusForbidden, "无权访问该数据")
	ErrNotAssigned       = New(40303, http.StatusForbidden, "未指派参与该盘点任务")

	// 404xx 资源不存在
	ErrNotFound         = New(40401, http.StatusNotFound, "资源不存在")
	ErrUserNotFound     = New(40402, http.StatusNotFound, "用户不存在")
	ErrAssetNotFound    = New(40403, http.StatusNotFound, "资产不存在")
	ErrWorkflowNotFound = New(40404, http.StatusNotFound, "工单不存在")
	ErrTaskNotFound     = New(40405, http.StatusNotFound, "盘点任务不存在")
	ErrExportJobNotFound = New(40406, http.StatusNotFound, "导出任务不存在")

	// 409xx 冲突/重复
	ErrConflict       = New(40901, http.StatusConflict, "操作冲突")
	ErrDuplicateOpen  = New(40902, http.StatusConflict, "该资产已有进行中的申请")
	ErrDuplicateKey   = New(40903, http.StatusConflict, "唯一键冲突")

	// 422xx 业务状态不允许
	ErrInvalidState     = New(42201, http.StatusUnprocessableEntity, "业务状态不允许")
	ErrAlreadyArchived  = New(42202, http.StatusUnprocessableEntity, "工单已归档")
	ErrInvalidTimeRange = New(42203, http.StatusUnprocessableEntity, "时间窗设置无效")
	ErrExportNotReady   = New(42204, http.StatusUnprocessableEntity, "导出任务未完成")
	ErrOpenWorkflow     = New(42205, http.StatusUnprocessableEntity, "存在进行中的审批，无法修改资产")

	// 500xx 服务器内部错误
	ErrInternal = New(50001, http.StatusInternalServerError, "服务器内部错误")

	// 503xx 依赖不可用
	ErrServiceUnavailable = New(50301, http.StatusServiceUnavailable, "服务依赖不可用")
)

// ==========================================
// 错误码 → HTTP 状态码映射
// ==========================================

var codeToHTTP = map[int]int{}

func init() {
	for _, e := range []*BizError{
		ErrInvalidParam, ErrInvalidPagination, ErrInvalidTimeFormat,
		ErrUnauthenticated, ErrTokenRevoked,
		ErrForbidden, ErrDeptAccessDenied, ErrNotAssigned,
		ErrNotFound, ErrUserNotFound, ErrAssetNotFound,
		ErrWorkflowNotFound, ErrTaskNotFound, ErrExportJobNotFound,
		ErrConflict, ErrDuplicateOpen, ErrDuplicateKey,
		ErrInvalidState, ErrAlreadyArchived, ErrInvalidTimeRange,
		ErrExportNotReady, ErrOpenWorkflow,
		ErrInternal, ErrServiceUnavailable,
	} {
		codeToHTTP[e.Code] = e.HTTP
	}
}

// ==========================================
// 公共方法
// ==========================================

// ToHTTPError 从任意 error 提取 (code, httpStatus, message)
func ToHTTPError(err error) (int, int, string) {
	if err == nil {
		return 0, http.StatusOK, "ok"
	}
	if be, ok := err.(*BizError); ok {
		return be.Code, be.HTTP, be.Message
	}
	// 未知错误 → 50001
	return ErrInternal.Code, ErrInternal.HTTP, ErrInternal.Message
}

// ToGRPCError 从任意 error 创建 gRPC status error
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}
	be, ok := err.(*BizError)
	if !ok {
		be = ErrInternal
	}
	return status.Error(grpcCodeFromHTTP(be.HTTP), fmt.Sprintf("%d:%s", be.Code, be.Message))
}

// FromGRPCError 从 gRPC status error 恢复 BizError
func FromGRPCError(err error) *BizError {
	st, ok := status.FromError(err)
	if !ok {
		return ErrInternal
	}
	code := httpCodeFromGRPC(st.Code())
	// 尝试解析消息中的业务错误码
	var bizCode int
	var msg string
	if _, scanErr := fmt.Sscanf(st.Message(), "%d:", &bizCode); scanErr == nil {
		msg = st.Message()[len(fmt.Sprintf("%d:", bizCode)):]
	} else {
		bizCode = code
		msg = st.Message()
	}
	return New(bizCode, code, msg)
}

// ==========================================
// gRPC ↔ HTTP 映射（06-error-codes.md §4）
// ==========================================

func grpcCodeFromHTTP(httpStatus int) codes.Code {
	switch httpStatus {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusUnprocessableEntity:
		return codes.FailedPrecondition
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

func httpCodeFromGRPC(c codes.Code) int {
	switch c {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.FailedPrecondition:
		return http.StatusUnprocessableEntity
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
