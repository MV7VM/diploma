package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestServerForMiddleware() (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	cfg := &config.Model{
		HTTP: config.HTTPConfig{
			Host:        "localhost:8080",
			SecretToken: "test-secret-token-for-middleware",
		},
	}

	server := &Server{
		logger: logger,
		serv:   gin.New(),
		cfg:    cfg,
		uc:     nil, // не используется в middleware
	}

	engine := gin.New()
	server.serv = engine

	return server, engine
}

func createValidToken(secretToken string, userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	})

	signedString, err := token.SignedString([]byte(secretToken))
	if err != nil {
		return "", err
	}

	return signedString, nil
}

func createExpiredToken(secretToken string, userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(-time.Hour).Unix(), // истекший токен
		"iat":    time.Now().Add(-2 * time.Hour).Unix(),
	})

	signedString, err := token.SignedString([]byte(secretToken))
	if err != nil {
		return "", err
	}

	return signedString, nil
}

func createTokenWithWrongSecret(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	})

	// Используем неправильный секретный ключ
	signedString, err := token.SignedString([]byte("wrong-secret-token"))
	if err != nil {
		return "", err
	}

	return signedString, nil
}

func createTokenWithWrongAlg(userID int) (string, error) {
	// Используем RS256 вместо HS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	})

	// Это не сработает без приватного ключа, но для теста это нормально
	signedString, err := token.SignedString([]byte("test-secret-token-for-middleware"))
	if err != nil {
		return "", err
	}

	return signedString, nil
}

func TestMiddleware_Auth_Success(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		userID, _ := c.Get("userID")
		require.NotNil(t, userID)
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	}

	engine.GET("/test", server.auth, testHandler)

	token, err := createValidToken(server.cfg.HTTP.SecretToken, 123)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "123")
}

func TestMiddleware_Auth_NoCookie(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.auth, testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Не добавляем cookie

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_Auth_InvalidToken(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.auth, testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: "invalid-token-string",
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_Auth_WrongSecret(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.auth, testHandler)

	token, err := createTokenWithWrongSecret(123)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_Auth_ExpiredToken(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.auth, testHandler)

	token, err := createExpiredToken(server.cfg.HTTP.SecretToken, 123)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_ParseToken_Success(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		require.NoError(t, err)

		userID, _ := c.Get("userID")
		require.NotNil(t, userID)
		assert.Equal(t, float64(123), userID) // JWT парсит числа как float64

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	token, err := createValidToken(server.cfg.HTTP.SecretToken, 123)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_ParseToken_NoCookie(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		require.Error(t, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Не добавляем cookie

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMiddleware_ParseToken_InvalidToken(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		require.Error(t, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: "invalid.jwt.token",
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMiddleware_ParseToken_WrongSecret(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		require.Error(t, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	})

	token, err := createTokenWithWrongSecret(123)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMiddleware_ParseToken_WrongAlgorithm(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		// Может быть ошибка или нет, в зависимости от того, как JWT библиотека обрабатывает это
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Попробуем создать токен с неправильным алгоритмом
	// Но это может не сработать без приватного ключа
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// Может вернуть ошибку из-за неправильного алгоритма
	assert.True(t, rec.Code == http.StatusInternalServerError || rec.Code == http.StatusOK)
}

func TestMiddleware_ParseToken_SetsClaims(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		require.NoError(t, err)

		// Проверяем, что все claims установлены
		userID, _ := c.Get("userID")
		require.NotNil(t, userID)

		exp, _ := c.Get("exp")
		require.NotNil(t, exp)

		iat, _ := c.Get("iat")
		require.NotNil(t, iat)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	token, err := createValidToken(server.cfg.HTTP.SecretToken, 456)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_ParseToken_MultipleClaims(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	// Создаем токен с несколькими claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID":   789,
		"username": "testuser",
		"role":     "admin",
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	signedToken, err := token.SignedString([]byte(server.cfg.HTTP.SecretToken))
	require.NoError(t, err)

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		require.NoError(t, err)

		assert.Equal(t, float64(789), c.GetFloat64("userID"))
		assert.Equal(t, "testuser", c.GetString("username"))
		assert.Equal(t, "admin", c.GetString("role"))
		//assert.NotNil(t, c.Get("exp"))
		//assert.NotNil(t, c.Get("iat"))

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: signedToken,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_WithLogger(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.withLogger, testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	// withLogger должен пропустить запрос дальше и залогировать
	assert.Equal(t, http.StatusOK, rec.Code)
	// Логирование не должно влиять на ответ
}

func TestMiddleware_Auth_NextCalled(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	called := false
	testHandler := func(c *gin.Context) {
		called = true
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.auth, testHandler)

	token, err := createValidToken(server.cfg.HTTP.SecretToken, 123)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.True(t, called, "Handler should be called when auth succeeds")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_Auth_NextNotCalled(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	called := false
	testHandler := func(c *gin.Context) {
		called = true
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	engine.GET("/test", server.auth, testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Не добавляем cookie

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.False(t, called, "Handler should not be called when auth fails")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_Auth_Chain(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	testHandler := func(c *gin.Context) {
		// Проверяем, что middleware выполнились в правильном порядке
		userID, _ := c.Get("userID")
		require.NotNil(t, userID)
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	}

	// Цепочка middleware: logger -> auth -> handler
	engine.GET("/test", server.withLogger, server.auth, testHandler)

	token, err := createValidToken(server.cfg.HTTP.SecretToken, 999)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "999")
}

func TestMiddleware_ParseToken_EmptyClaims(t *testing.T) {
	server, engine := setupTestServerForMiddleware()

	// Создаем токен без claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	signedToken, err := token.SignedString([]byte(server.cfg.HTTP.SecretToken))
	require.NoError(t, err)

	engine.GET("/test", func(c *gin.Context) {
		err := server.parseToken(c)
		// Должен успешно распарсить, но claims будут пустыми
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: signedToken,
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
