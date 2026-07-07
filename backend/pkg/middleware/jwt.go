// Package middleware 通用 HTTP/gRPC 中间件
// 包含 JWT 校验、黑名单检查、部门数据隔离
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// ==========================================
// Context Keys
// ==========================================

type ctxKey string

const (
	UIDKey       ctxKey = "uid"
	RoleLevelKey ctxKey = "role_level"
	DeptIDKey    ctxKey = "dept_id"
	JtiKey       ctxKey = "jti"
)

// ==========================================
// JWT Claims
// ==========================================

// FamsClaims JWT 载荷
type FamsClaims struct {
	jwt.RegisteredClaims
	UID       int64 `json:"uid"`
	RoleLevel int16 `json:"role_level"`
	DeptID    int64 `json:"dept_id"`
}

// ==========================================
// JWT 中间件
// ==========================================

// JWTAuth 返回 JWT 鉴权 HTTP 中间件
// accessSecret: Access Token 密钥
// rds: Redis 客户端（黑名单检查，可为 nil 跳过）
func JWTAuth(accessSecret string, rds *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := parseAndValidate(r, accessSecret, rds)
			if err != nil {
				code, httpStatus, msg := errx.ToHTTPError(err)
				writeJSON(w, httpStatus, code, msg)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UIDKey, claims.UID)
			ctx = context.WithValue(ctx, RoleLevelKey, claims.RoleLevel)
			ctx = context.WithValue(ctx, DeptIDKey, claims.DeptID)
			ctx = context.WithValue(ctx, JtiKey, claims.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseAndValidate(r *http.Request, accessSecret string, rds *redis.Client) (*FamsClaims, error) {
	// 1. 提取 Bearer Token
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return nil, errx.ErrUnauthenticated
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	// 2. 解析并验证签名/过期
	claims := &FamsClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(accessSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errx.ErrUnauthenticated
	}

	// 3. Redis 黑名单检查
	if rds != nil && claims.ID != "" {
		blacklisted, _ := rds.Exists(r.Context(), "fams:auth:blacklist:"+claims.ID).Result()
		if blacklisted > 0 {
			return nil, errx.ErrTokenRevoked
		}
	}

	return claims, nil
}

// ==========================================
// Refresh Token 解析（不检查黑名单，用于 refresh 接口）
// ==========================================

// ParseRefreshToken 解析 Refresh Token
func ParseRefreshToken(tokenStr, refreshSecret string) (*FamsClaims, error) {
	claims := &FamsClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(refreshSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errx.ErrUnauthenticated
	}
	return claims, nil
}

// ==========================================
// Token 生成
// ==========================================

// GenerateTokens 生成 Access + Refresh Token 对
func GenerateTokens(uid int64, roleLevel int16, deptID int64, accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) (accessToken, refreshToken, jti string, err error) {
	jti = generateJTI()
	now := time.Now()

	accessClaims := &FamsClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
		},
		UID:       uid,
		RoleLevel: roleLevel,
		DeptID:    deptID,
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(accessSecret))
	if err != nil {
		return "", "", "", err
	}

	refreshClaims := &FamsClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti + "_refresh",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
		},
		UID:       uid,
		RoleLevel: roleLevel,
		DeptID:    deptID,
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(refreshSecret))
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, jti, nil
}

func generateJTI() string {
	// 使用时间戳 + 随机后缀，正式环境可用 uuid
	return "jti_" + time.Now().Format("20060102150405.000000")
}

// ==========================================
// Context 辅助函数
// ==========================================

// GetUID 从 context 取用户 ID
func GetUID(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(UIDKey).(int64)
	return v, ok
}

// GetRoleLevel 从 context 取角色级别
func GetRoleLevel(ctx context.Context) (int16, bool) {
	v, ok := ctx.Value(RoleLevelKey).(int16)
	return v, ok
}

// GetDeptID 从 context 取部门 ID
func GetDeptID(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(DeptIDKey).(int64)
	return v, ok
}

// GetJti 从 context 取 JWT Token ID
func GetJti(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(JtiKey).(string)
	return v, ok
}

// ==========================================
// 工具函数
// ==========================================

func writeJSON(w http.ResponseWriter, httpStatus, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	// 简单 JSON 写入（避免循环依赖，实际 handler 使用框架统一包装）
	body := `{"code":` + itoa(code) + `,"message":"` + msg + `","data":null}`
	w.Write([]byte(body))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
