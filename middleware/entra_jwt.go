package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type EntraJWTConfig struct {
	TenantID string
	Issuer   string
	Audience string
	JWKSURL  string // 例如: https://login.microsoftonline.com/<TENANT_ID>/discovery/v2.0/keys
}

func NewEntraJWTMiddleware(cfg EntraJWTConfig) (gin.HandlerFunc, func(), error) {
	// keyfunc/v3 用 context 来结束后台 refresh goroutine（推荐用 NewDefaultCtx）。:contentReference[oaicite:2]{index=2}
	ctx, cancel := context.WithCancel(context.Background())

	k, err := keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
	if err != nil {
		cancel()
		return nil, nil, err
	}

	cleanup := func() {
		// 取消 context -> 结束 refresh goroutine
		cancel()
	}

	mw := func(c *gin.Context) {
		// 1) 取 Authorization: Bearer <token>
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		rawToken := strings.TrimSpace(parts[1])
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "empty bearer token"})
			return
		}

		// 2) 解析并验证 token（签名 + iss + aud + exp/nbf）
		token, err := jwt.Parse(
			rawToken,
			k.Keyfunc, // ✅ 这是 jwt/v5.Keyfunc
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
			jwt.WithLeeway(30*time.Second),
		)
		if err != nil || token == nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// 3) 可选：校验 tid（确保是你公司租户）
		if cfg.TenantID != "" {
			if tid, _ := claims["tid"].(string); tid == "" || tid != cfg.TenantID {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "wrong tenant"})
				return
			}
		}

		// 4) 放到 context，controller 里可取
		c.Set("claims", claims)

		if oid, _ := claims["oid"].(string); oid != "" {
			c.Set("user_oid", oid)
		}
		if upn, _ := claims["preferred_username"].(string); upn != "" {
			c.Set("user_upn", upn)
		}

		c.Next()
	}

	return mw, cleanup, nil
}

// 如果你要做 scope 授权（scp 里包含 "access_as_user"）
func RequireScope(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
			return
		}

		claims, ok := v.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}

		scp, _ := claims["scp"].(string) // 例如 "access_as_user other_scope"
		if scp == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing scp"})
			return
		}

		for _, s := range strings.Split(scp, " ") {
			if s == required {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
	}
}
