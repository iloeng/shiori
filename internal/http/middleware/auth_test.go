package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-shiori/shiori/internal/http/response"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/go-shiori/shiori/internal/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAuthenticationRequiredMiddleware(t *testing.T) {
	t.Run("test unauthorized", func(t *testing.T) {
		g := testutil.NewGin()
		g.Use(AuthenticationRequired())
		g.Handle("GET", "/", func(c *gin.Context) {
			response.Send(c, http.StatusOK, nil)
		})
		w := testutil.PerformRequest(g, "GET", "/")
		require.Equal(t, http.StatusUnauthorized, w.Code)
		// This ensures we are aborting the request and not sending more data
		require.Equal(t, `{"ok":false,"message":null}`, w.Body.String())
	})

	t.Run("test authorized", func(t *testing.T) {
		g := testutil.NewGin()
		// Fake a logged in user in the context, which is the way the AuthMiddleware works.
		g.Use(func(ctx *gin.Context) {
			ctx.Set(model.ContextAccountKey, "test")
		})
		g.Use(AuthenticationRequired())
		g.GET("/", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := testutil.PerformRequest(g, "GET", "/")
		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthMiddleware(t *testing.T) {
	ctx := context.TODO()
	logger := logrus.New()
	_, deps := testutil.GetTestConfigurationAndDependencies(t, ctx, logger)
	middleware := AuthMiddleware(deps)

	t.Run("test no authorization method", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.Use(middleware)
		router.ServeHTTP(w, req)

		_, exists := c.Get("account")
		require.False(t, exists)
	})

	t.Run("test authorization header", func(t *testing.T) {
		account := testutil.GetValidAccount().ToDTO()
		token, err := deps.Domains.Auth.CreateTokenForAccount(&account, time.Now().Add(time.Minute))
		require.NoError(t, err)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.Header.Set(model.AuthorizationHeader, model.AuthorizationTokenType+" "+token)
		middleware(c)
		_, exists := c.Get(model.ContextAccountKey)
		require.True(t, exists)
	})

	t.Run("test authorization cookie", func(t *testing.T) {
		account := model.AccountDTO{Username: "shiori"}
		token, err := deps.Domains.Auth.CreateTokenForAccount(&account, time.Now().Add(time.Minute))
		require.NoError(t, err)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:   "token",
			Value:  token,
			MaxAge: int(time.Now().Add(time.Minute).Unix()),
		})
		middleware(c)
		_, exists := c.Get(model.ContextAccountKey)
		require.True(t, exists)
	})
}

func TestAdminRequiredMiddleware(t *testing.T) {
	t.Run("test unauthorized", func(t *testing.T) {
		g := testutil.NewGin()
		g.Use(AdminRequired())
		g.Handle("GET", "/", func(c *gin.Context) {
			response.Send(c, http.StatusOK, nil)
		})
		w := testutil.PerformRequest(g, "GET", "/")
		require.Equal(t, http.StatusForbidden, w.Code)
		// This ensures we are aborting the request and not sending more data
		require.Equal(t, `{"ok":false,"message":null}`, w.Body.String())
	})

	t.Run("test user but not admin", func(t *testing.T) {
		g := testutil.NewGin()
		// Fake a logged in admin in the context, which is the way the AuthMiddleware works.
		g.Use(func(ctx *gin.Context) {
			ctx.Set(model.ContextAccountKey, &model.AccountDTO{
				Owner: model.Ptr(false),
			})
		})
		g.Use(AdminRequired())
		g.GET("/", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := testutil.PerformRequest(g, "GET", "/")
		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("test authorized", func(t *testing.T) {
		g := testutil.NewGin()
		// Fake a logged in admin in the context, which is the way the AuthMiddleware works.
		g.Use(func(ctx *gin.Context) {
			ctx.Set(model.ContextAccountKey, &model.AccountDTO{
				Owner: model.Ptr(true),
			})
		})
		g.Use(AdminRequired())
		g.GET("/", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := testutil.PerformRequest(g, "GET", "/")
		require.Equal(t, http.StatusOK, w.Code)
	})
}
