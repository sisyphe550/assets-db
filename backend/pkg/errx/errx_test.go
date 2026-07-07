package errx

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBizError_Error(t *testing.T) {
	e := New(42201, 422, "资产当前不可领用")
	if e.Error() != "[42201] 资产当前不可领用" {
		t.Errorf("unexpected Error(): %s", e.Error())
	}
}

func TestToHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantHTTP   int
		wantMsg    string
	}{
		{"nil", nil, 0, 200, "ok"},
		{"biz error", ErrInvalidState, 42201, 422, "业务状态不允许"},
		{"not found", ErrUserNotFound, 40402, 404, "用户不存在"},
		{"unknown error", errors.New("unknown"), 50001, 500, "服务器内部错误"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, httpStatus, msg := ToHTTPError(tt.err)
			if code != tt.wantCode || httpStatus != tt.wantHTTP || msg != tt.wantMsg {
				t.Errorf("ToHTTPError() = (%d, %d, %s), want (%d, %d, %s)",
					code, httpStatus, msg, tt.wantCode, tt.wantHTTP, tt.wantMsg)
			}
		})
	}
}

func TestToGRPCError(t *testing.T) {
	err := ToGRPCError(ErrInvalidState)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestFromGRPCError(t *testing.T) {
	orig := ErrInvalidState
	grpcErr := ToGRPCError(orig)
	be := FromGRPCError(grpcErr)
	if be.Code != orig.Code {
		t.Errorf("code %d != %d", be.Code, orig.Code)
	}
}

func TestAllErrorCodes(t *testing.T) {
	// 验证所有错误码都已注册到 codeToHTTP
	codes := []*BizError{
		ErrInvalidParam, ErrInvalidPagination, ErrInvalidTimeFormat,
		ErrUnauthenticated, ErrTokenRevoked,
		ErrForbidden, ErrDeptAccessDenied, ErrNotAssigned,
		ErrNotFound, ErrUserNotFound, ErrAssetNotFound,
		ErrWorkflowNotFound, ErrTaskNotFound, ErrExportJobNotFound,
		ErrConflict, ErrDuplicateOpen, ErrDuplicateKey,
		ErrInvalidState, ErrAlreadyArchived, ErrInvalidTimeRange,
		ErrExportNotReady, ErrOpenWorkflow,
		ErrInternal, ErrServiceUnavailable,
	}
	for _, e := range codes {
		if _, ok := codeToHTTP[e.Code]; !ok {
			t.Errorf("error code %d not registered in codeToHTTP", e.Code)
		}
	}
}

func TestHTTPCodeFromGRPC(t *testing.T) {
	tests := []struct {
		grpcCode codes.Code
		httpCode int
	}{
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.NotFound, http.StatusNotFound},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.FailedPrecondition, http.StatusUnprocessableEntity},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.Internal, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if got := httpCodeFromGRPC(tt.grpcCode); got != tt.httpCode {
			t.Errorf("httpCodeFromGRPC(%v) = %d, want %d", tt.grpcCode, got, tt.httpCode)
		}
	}
}

func TestGrpcCodeMapping(t *testing.T) {
	httpToGrpc := map[int]codes.Code{
		http.StatusBadRequest:          codes.InvalidArgument,
		http.StatusUnauthorized:        codes.Unauthenticated,
		http.StatusForbidden:           codes.PermissionDenied,
		http.StatusNotFound:            codes.NotFound,
		http.StatusConflict:            codes.AlreadyExists,
		http.StatusUnprocessableEntity: codes.FailedPrecondition,
		http.StatusServiceUnavailable:  codes.Unavailable,
		http.StatusInternalServerError: codes.Internal,
	}
	for httpCode, grpcCode := range httpToGrpc {
		if got := grpcCodeFromHTTP(httpCode); got != grpcCode {
			t.Errorf("grpcCodeFromHTTP(%d) = %v, want %v", httpCode, got, grpcCode)
		}
	}
}
