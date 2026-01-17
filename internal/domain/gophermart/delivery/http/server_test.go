package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockUsecase мок для интерфейса uc
type mockUsecase struct {
	RegisterFunc       func(ctx context.Context, creds *entities.UserAuth) (string, error)
	LoginFunc          func(ctx context.Context, creds *entities.UserAuth) (string, error)
	UploadOrderFunc    func(ctx context.Context, userID int, order string) error
	GetOrdersFunc      func(ctx context.Context, userID int) ([]entities.Order, error)
	UploadWithdrawFunc func(ctx context.Context, withdraw *entities.Withdraw, userID int) error
	GetWithdrawFunc    func(ctx context.Context, userID int) ([]entities.Withdraw, error)
	GetBalanceFunc     func(ctx context.Context, userID int) (*entities.Balance, error)
}

func (m *mockUsecase) Register(ctx context.Context, creds *entities.UserAuth) (string, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, creds)
	}
	return "", errors.New("not implemented")
}

func (m *mockUsecase) Login(ctx context.Context, creds *entities.UserAuth) (string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, creds)
	}
	return "", errors.New("not implemented")
}

func (m *mockUsecase) UploadOrder(ctx context.Context, userID int, order string) error {
	if m.UploadOrderFunc != nil {
		return m.UploadOrderFunc(ctx, userID, order)
	}
	return errors.New("not implemented")
}

func (m *mockUsecase) GetOrders(ctx context.Context, userID int) ([]entities.Order, error) {
	if m.GetOrdersFunc != nil {
		return m.GetOrdersFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUsecase) UploadWithdraw(ctx context.Context, withdraw *entities.Withdraw, userID int) error {
	if m.UploadWithdrawFunc != nil {
		return m.UploadWithdrawFunc(ctx, withdraw, userID)
	}
	return errors.New("not implemented")
}

func (m *mockUsecase) GetWithdraw(ctx context.Context, userID int) ([]entities.Withdraw, error) {
	if m.GetWithdrawFunc != nil {
		return m.GetWithdrawFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUsecase) GetBalance(ctx context.Context, userID int) (*entities.Balance, error) {
	if m.GetBalanceFunc != nil {
		return m.GetBalanceFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func setupTestServer(mockUC *mockUsecase) (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	cfg := &config.Model{
		HTTP: config.HTTPConfig{
			Host:        "localhost:8080",
			SecretToken: "test-secret-token",
		},
	}

	server := &Server{
		logger: logger,
		serv:   gin.New(),
		cfg:    cfg,
		uc:     mockUC,
	}

	return server, server.serv
}

func TestServer_Register_Success(t *testing.T) {
	expectedToken := "test-token"
	mockUC := &mockUsecase{
		RegisterFunc: func(ctx context.Context, creds *entities.UserAuth) (string, error) {
			assert.Equal(t, "testuser", creds.Login)
			assert.Equal(t, "testpass", creds.Password)
			return expectedToken, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/register", server.Register)

	body := `{"login":"testuser","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, expectedToken, rec.Header().Get("Authorization"))
}

func TestServer_Register_BadRequest(t *testing.T) {
	mockUC := &mockUsecase{}
	server, engine := setupTestServer(mockUC)
	engine.POST("/register", server.Register)

	body := `{"login":"testuser"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestServer_Register_Conflict(t *testing.T) {
	mockUC := &mockUsecase{
		RegisterFunc: func(ctx context.Context, creds *entities.UserAuth) (string, error) {
			return "", entities.ErrAlreadyInUse
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/register", server.Register)

	body := `{"login":"testuser","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestServer_Register_InternalError(t *testing.T) {
	mockUC := &mockUsecase{
		RegisterFunc: func(ctx context.Context, creds *entities.UserAuth) (string, error) {
			return "", errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/register", server.Register)

	body := `{"login":"testuser","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_Login_Success(t *testing.T) {
	expectedToken := "test-token"
	mockUC := &mockUsecase{
		LoginFunc: func(ctx context.Context, creds *entities.UserAuth) (string, error) {
			assert.Equal(t, "testuser", creds.Login)
			assert.Equal(t, "testpass", creds.Password)
			return expectedToken, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/login", server.Login)

	body := `{"login":"testuser","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, expectedToken, rec.Header().Get("Authorization"))
}

func TestServer_Login_Unauthorized(t *testing.T) {
	mockUC := &mockUsecase{
		LoginFunc: func(ctx context.Context, creds *entities.UserAuth) (string, error) {
			return "", entities.ErrWrongCredentials
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/login", server.Login)

	body := `{"login":"testuser","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestServer_UploadOrder_Success(t *testing.T) {
	orderNumber := "9278923470"
	mockUC := &mockUsecase{
		UploadOrderFunc: func(ctx context.Context, userID int, order string) error {
			assert.Equal(t, 123, userID)
			assert.Equal(t, orderNumber, order)
			return nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadOrder(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(orderNumber))
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestServer_UploadOrder_InvalidOrderNumber(t *testing.T) {
	mockUC := &mockUsecase{}
	server, engine := setupTestServer(mockUC)
	engine.POST("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadOrder(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString("12345"))
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestServer_UploadOrder_AlreadyInUse(t *testing.T) {
	orderNumber := "9278923470"
	mockUC := &mockUsecase{
		UploadOrderFunc: func(ctx context.Context, userID int, order string) error {
			return entities.ErrAlreadyInUse
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadOrder(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(orderNumber))
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_UploadOrder_PermissionDenied(t *testing.T) {
	orderNumber := "9278923470"
	mockUC := &mockUsecase{
		UploadOrderFunc: func(ctx context.Context, userID int, order string) error {
			return entities.ErrPermissionDenied
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadOrder(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(orderNumber))
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestServer_GetOrders_Success(t *testing.T) {
	expectedOrders := []entities.Order{
		{
			Number:     "9278923470",
			Status:     "PROCESSED",
			Accrual:    floatPtr(500.0),
			UploadedAt: time.Now(),
		},
	}

	mockUC := &mockUsecase{
		GetOrdersFunc: func(ctx context.Context, userID int) ([]entities.Order, error) {
			assert.Equal(t, 123, userID)
			return expectedOrders, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetOrders(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var orders []entities.Order
	err := json.Unmarshal(rec.Body.Bytes(), &orders)
	require.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, expectedOrders[0].Number, orders[0].Number)
}

func TestServer_GetOrders_NoContent(t *testing.T) {
	mockUC := &mockUsecase{
		GetOrdersFunc: func(ctx context.Context, userID int) ([]entities.Order, error) {
			return []entities.Order{}, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetOrders(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestServer_GetOrders_InternalError(t *testing.T) {
	mockUC := &mockUsecase{
		GetOrdersFunc: func(ctx context.Context, userID int) ([]entities.Order, error) {
			return nil, errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetOrders(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_UploadWithdraw_Success(t *testing.T) {
	withdraw := entities.Withdraw{
		OrderNumber: "9278923470",
		Sum:         100.5,
	}

	mockUC := &mockUsecase{
		UploadWithdrawFunc: func(ctx context.Context, w *entities.Withdraw, userID int) error {
			assert.Equal(t, 123, userID)
			assert.Equal(t, withdraw.OrderNumber, w.OrderNumber)
			assert.Equal(t, withdraw.Sum, w.Sum)
			return nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/withdraw", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadWithdraw(c)
	})

	body, _ := json.Marshal(withdraw)
	req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_UploadWithdraw_InvalidOrderNumber(t *testing.T) {
	mockUC := &mockUsecase{}
	server, engine := setupTestServer(mockUC)
	engine.POST("/withdraw", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadWithdraw(c)
	})

	withdraw := entities.Withdraw{
		OrderNumber: "12345",
		Sum:         100.5,
	}
	body, _ := json.Marshal(withdraw)
	req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestServer_UploadWithdraw_EmptyBalance(t *testing.T) {
	withdraw := entities.Withdraw{
		OrderNumber: "9278923470",
		Sum:         100.5,
	}

	mockUC := &mockUsecase{
		UploadWithdrawFunc: func(ctx context.Context, w *entities.Withdraw, userID int) error {
			return entities.ErrEmptyBalance
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/withdraw", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadWithdraw(c)
	})

	body, _ := json.Marshal(withdraw)
	req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPaymentRequired, rec.Code)
}

func TestServer_GetBalance_Success(t *testing.T) {
	expectedBalance := &entities.Balance{
		Current:   500.5,
		Withdrawn: 100.0,
	}

	mockUC := &mockUsecase{
		GetBalanceFunc: func(ctx context.Context, userID int) (*entities.Balance, error) {
			assert.Equal(t, 123, userID)
			return expectedBalance, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/balance", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetBalance(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var balance entities.Balance
	err := json.Unmarshal(rec.Body.Bytes(), &balance)
	require.NoError(t, err)
	assert.Equal(t, expectedBalance.Current, balance.Current)
	assert.Equal(t, expectedBalance.Withdrawn, balance.Withdrawn)
}

func TestServer_GetBalance_InternalError(t *testing.T) {
	mockUC := &mockUsecase{
		GetBalanceFunc: func(ctx context.Context, userID int) (*entities.Balance, error) {
			return nil, errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/balance", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetBalance(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_GetWithdraw_Success(t *testing.T) {
	expectedWithdraws := []entities.Withdraw{
		{
			OrderNumber: "9278923470",
			Sum:         100.5,
			UploadedAt:  time.Now(),
		},
	}

	mockUC := &mockUsecase{
		GetWithdrawFunc: func(ctx context.Context, userID int) ([]entities.Withdraw, error) {
			assert.Equal(t, 123, userID)
			return expectedWithdraws, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/withdrawals", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetWithdraw(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var withdraws []entities.Withdraw
	err := json.Unmarshal(rec.Body.Bytes(), &withdraws)
	require.NoError(t, err)
	assert.Len(t, withdraws, 1)
	assert.Equal(t, expectedWithdraws[0].OrderNumber, withdraws[0].OrderNumber)
}

func TestServer_GetWithdraw_NoContent(t *testing.T) {
	mockUC := &mockUsecase{
		GetWithdrawFunc: func(ctx context.Context, userID int) ([]entities.Withdraw, error) {
			return []entities.Withdraw{}, nil
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/withdrawals", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetWithdraw(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestServer_GetWithdraw_InternalError(t *testing.T) {
	mockUC := &mockUsecase{
		GetWithdrawFunc: func(ctx context.Context, userID int) ([]entities.Withdraw, error) {
			return nil, errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.GET("/withdrawals", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.GetWithdraw(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestValidateOrderNumber_Valid(t *testing.T) {
	validNumbers := []string{
		"9278923470",
		"12345678903",
		"346436439",
		"0000000000",
	}

	for _, num := range validNumbers {
		t.Run(num, func(t *testing.T) {
			assert.True(t, validateOrderNumber(num))
		})
	}
}

func TestValidateOrderNumber_Invalid(t *testing.T) {
	invalidNumbers := []string{
		"12345",
		"123",
		"abc",
		"",
		"12345678901",
		"1",
	}

	for _, num := range invalidNumbers {
		t.Run(num, func(t *testing.T) {
			assert.False(t, validateOrderNumber(num))
		})
	}
}

func TestValidateOrderNumber_WithSpaces(t *testing.T) {
	validNumber := "9278 9234 70"
	assert.True(t, validateOrderNumber(validNumber))
}

func TestValidateOrderNumber_WithNonNumeric(t *testing.T) {
	validNumber := "9278-9234-70"
	assert.True(t, validateOrderNumber(validNumber))
}

func TestServer_UploadOrder_InternalError(t *testing.T) {
	orderNumber := "9278923470"
	mockUC := &mockUsecase{
		UploadOrderFunc: func(ctx context.Context, userID int, order string) error {
			return errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/orders", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadOrder(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(orderNumber))
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_UploadWithdraw_InternalError(t *testing.T) {
	withdraw := entities.Withdraw{
		OrderNumber: "9278923470",
		Sum:         100.5,
	}

	mockUC := &mockUsecase{
		UploadWithdrawFunc: func(ctx context.Context, w *entities.Withdraw, userID int) error {
			return errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/withdraw", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadWithdraw(c)
	})

	body, _ := json.Marshal(withdraw)
	req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_UploadWithdraw_BadJSON(t *testing.T) {
	mockUC := &mockUsecase{}
	server, engine := setupTestServer(mockUC)
	engine.POST("/withdraw", func(c *gin.Context) {
		c.Set("userID", float64(123))
		server.UploadWithdraw(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_Login_BadRequest(t *testing.T) {
	mockUC := &mockUsecase{}
	server, engine := setupTestServer(mockUC)
	engine.POST("/login", server.Login)

	body := `{"login":"testuser"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestServer_Login_InternalError(t *testing.T) {
	mockUC := &mockUsecase{
		LoginFunc: func(ctx context.Context, creds *entities.UserAuth) (string, error) {
			return "", errors.New("database error")
		},
	}

	server, engine := setupTestServer(mockUC)
	engine.POST("/login", server.Login)

	body := `{"login":"testuser","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Вспомогательная функция для создания указателя на float64
func floatPtr(f float64) *float64 {
	return &f
}
