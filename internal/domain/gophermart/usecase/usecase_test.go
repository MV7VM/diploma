package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockRepo мок для интерфейса repo
type mockRepo struct {
	RegisterUserFunc            func(ctx context.Context, auth *entities.UserAuth) (int, error)
	LoginUserFunc               func(ctx context.Context, auth *entities.UserAuth) (int, error)
	UploadOrderFunc             func(ctx context.Context, userID int, order string) error
	GetOrdersFunc               func(ctx context.Context, userID int) ([]entities.Order, error)
	UploadWithdrawFunc          func(ctx context.Context, withdraw *entities.Withdraw, userID int) error
	GetWithdrawFunc             func(ctx context.Context, userID int) ([]entities.Withdraw, error)
	UpdateOrderFunc             func(ctx context.Context, order *entities.Accrual) error
	GetAllUnProcessedOrdersFunc func(ctx context.Context) ([]string, error)
}

func (m *mockRepo) RegisterUser(ctx context.Context, auth *entities.UserAuth) (int, error) {
	if m.RegisterUserFunc != nil {
		return m.RegisterUserFunc(ctx, auth)
	}
	return 0, errors.New("not implemented")
}

func (m *mockRepo) LoginUser(ctx context.Context, auth *entities.UserAuth) (int, error) {
	if m.LoginUserFunc != nil {
		return m.LoginUserFunc(ctx, auth)
	}
	return 0, errors.New("not implemented")
}

func (m *mockRepo) UploadOrder(ctx context.Context, userID int, order string) error {
	if m.UploadOrderFunc != nil {
		return m.UploadOrderFunc(ctx, userID, order)
	}
	return errors.New("not implemented")
}

func (m *mockRepo) GetOrders(ctx context.Context, userID int) ([]entities.Order, error) {
	if m.GetOrdersFunc != nil {
		return m.GetOrdersFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) UploadWithdraw(ctx context.Context, withdraw *entities.Withdraw, userID int) error {
	if m.UploadWithdrawFunc != nil {
		return m.UploadWithdrawFunc(ctx, withdraw, userID)
	}
	return errors.New("not implemented")
}

func (m *mockRepo) GetWithdraw(ctx context.Context, userID int) ([]entities.Withdraw, error) {
	if m.GetWithdrawFunc != nil {
		return m.GetWithdrawFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) UpdateOrder(ctx context.Context, order *entities.Accrual) error {
	if m.UpdateOrderFunc != nil {
		return m.UpdateOrderFunc(ctx, order)
	}
	return errors.New("not implemented")
}

func (m *mockRepo) GetAllUnProcessedOrders(ctx context.Context) ([]string, error) {
	if m.GetAllUnProcessedOrdersFunc != nil {
		return m.GetAllUnProcessedOrdersFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

// mockAccrualClient мок для accrual.Client
type mockAccrualClient struct {
	GetAccrualFunc func(baseURL, orderNumber string) (*entities.Accrual, error)
}

func (m *mockAccrualClient) GetAccrual(baseURL, orderNumber string) (*entities.Accrual, error) {
	if m.GetAccrualFunc != nil {
		return m.GetAccrualFunc(baseURL, orderNumber)
	}
	return nil, errors.New("not implemented")
}

func setupTestUsecase(mockRepo *mockRepo, mockAccrualClient *mockAccrualClient) *Usecase {
	ctx := context.Background()
	logger := zap.NewNop()
	cfg := &config.Model{
		HTTP: config.HTTPConfig{
			SecretToken: "test-secret-token",
		},
		AccrualSystem: config.HTTPConfig{
			Host: "http://localhost:8080",
		},
	}

	return &Usecase{
		ctx:           ctx,
		log:           logger.Named("usecase"),
		repo:          mockRepo,
		cfg:           cfg,
		accrualClient: mockAccrualClient,
	}
}

func TestNewUsecase(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	cfg := &config.Model{
		HTTP: config.HTTPConfig{
			SecretToken: "test-secret-token",
		},
	}

	uc, err := NewUsecase(ctx, cfg, logger, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, uc)
	assert.Equal(t, ctx, uc.ctx)
	assert.NotNil(t, uc.log)
	assert.Equal(t, cfg, uc.cfg)
}

func TestUsecase_Register_Success(t *testing.T) {
	expectedUserID := 123
	mockRepo := &mockRepo{
		RegisterUserFunc: func(ctx context.Context, auth *entities.UserAuth) (int, error) {
			assert.Equal(t, "testuser", auth.Login)
			assert.Equal(t, "testpass", auth.Password)
			return expectedUserID, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	creds := &entities.UserAuth{
		Login:    "testuser",
		Password: "testpass",
	}

	token, err := uc.Register(context.Background(), creds)

	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestUsecase_Register_RepoError(t *testing.T) {
	expectedErr := entities.ErrAlreadyInUse
	mockRepo := &mockRepo{
		RegisterUserFunc: func(ctx context.Context, auth *entities.UserAuth) (int, error) {
			return 0, expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	creds := &entities.UserAuth{
		Login:    "testuser",
		Password: "testpass",
	}

	token, err := uc.Register(context.Background(), creds)

	require.Error(t, err)
	assert.Empty(t, token)
	assert.True(t, errors.Is(err, expectedErr))
}

func TestUsecase_Login_Success(t *testing.T) {
	expectedUserID := 123
	mockRepo := &mockRepo{
		LoginUserFunc: func(ctx context.Context, auth *entities.UserAuth) (int, error) {
			assert.Equal(t, "testuser", auth.Login)
			assert.Equal(t, "testpass", auth.Password)
			return expectedUserID, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	creds := &entities.UserAuth{
		Login:    "testuser",
		Password: "testpass",
	}

	token, err := uc.Login(context.Background(), creds)

	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestUsecase_Login_RepoError(t *testing.T) {
	expectedErr := entities.ErrWrongCredentials
	mockRepo := &mockRepo{
		LoginUserFunc: func(ctx context.Context, auth *entities.UserAuth) (int, error) {
			return 0, expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	creds := &entities.UserAuth{
		Login:    "testuser",
		Password: "wrongpass",
	}

	token, err := uc.Login(context.Background(), creds)

	require.Error(t, err)
	assert.Empty(t, token)
	assert.True(t, errors.Is(err, expectedErr))
}

func TestUsecase_UploadOrder_Success(t *testing.T) {
	orderNumber := "9278923470"
	userID := 123
	mockRepo := &mockRepo{
		UploadOrderFunc: func(ctx context.Context, uid int, order string) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, orderNumber, order)
			return nil
		},
		UpdateOrderFunc: func(ctx context.Context, order *entities.Accrual) error {
			assert.Equal(t, orderNumber, order.Order)
			return nil
		},
	}
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "PROCESSED",
		Accrual: 500.0,
	}
	mockAccrualClient := &mockAccrualClient{
		GetAccrualFunc: func(baseURL, orderNum string) (*entities.Accrual, error) {
			assert.Equal(t, "http://localhost:8080", baseURL)
			assert.Equal(t, orderNumber, orderNum)
			return expectedAccrual, nil
		},
	}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadOrder(context.Background(), userID, orderNumber)

	require.NoError(t, err)
}

func TestUsecase_UploadOrder_AlreadyInUse(t *testing.T) {
	orderNumber := "9278923470"
	userID := 123
	mockRepo := &mockRepo{
		UploadOrderFunc: func(ctx context.Context, uid int, order string) error {
			return entities.ErrAlreadyInUse
		},
	}
	mockAccrualClient := &mockAccrualClient{
		GetAccrualFunc: func(baseURL, orderNum string) (*entities.Accrual, error) {
			return &entities.Accrual{
				Order:   orderNumber,
				Status:  "PROCESSED",
				Accrual: 500.0,
			}, nil
		},
	}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadOrder(context.Background(), userID, orderNumber)

	// Должен вернуть nil, даже если заказ уже существует (ErrAlreadyInUse игнорируется)
	assert.NoError(t, err)
}

func TestUsecase_UploadOrder_AccrualError(t *testing.T) {
	orderNumber := "9278923470"
	userID := 123
	mockRepo := &mockRepo{
		UploadOrderFunc: func(ctx context.Context, uid int, order string) error {
			return nil
		},
	}
	mockAccrualClient := &mockAccrualClient{
		GetAccrualFunc: func(baseURL, orderNum string) (*entities.Accrual, error) {
			return nil, errors.New("accrual service error")
		},
	}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadOrder(context.Background(), userID, orderNumber)

	// При ошибке accrual клиента должен вернуть nil
	assert.NoError(t, err)
}

func TestUsecase_UploadOrder_UpdateOrderError(t *testing.T) {
	orderNumber := "9278923470"
	userID := 123
	mockRepo := &mockRepo{
		UploadOrderFunc: func(ctx context.Context, uid int, order string) error {
			return nil
		},
		UpdateOrderFunc: func(ctx context.Context, order *entities.Accrual) error {
			return errors.New("update error")
		},
	}
	mockAccrualClient := &mockAccrualClient{
		GetAccrualFunc: func(baseURL, orderNum string) (*entities.Accrual, error) {
			return &entities.Accrual{
				Order:   orderNumber,
				Status:  "PROCESSED",
				Accrual: 500.0,
			}, nil
		},
	}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadOrder(context.Background(), userID, orderNumber)

	// При ошибке обновления заказа должен вернуть nil
	assert.NoError(t, err)
}

func TestUsecase_UploadOrder_RepoError(t *testing.T) {
	orderNumber := "9278923470"
	userID := 123
	expectedErr := entities.ErrPermissionDenied
	mockRepo := &mockRepo{
		UploadOrderFunc: func(ctx context.Context, uid int, order string) error {
			return expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadOrder(context.Background(), userID, orderNumber)

	require.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr))
}

func TestUsecase_GetOrders_Success(t *testing.T) {
	userID := 123
	expectedOrders := []entities.Order{
		{
			Number:     "9278923470",
			Status:     "PROCESSED",
			Accrual:    floatPtr(500.0),
			UploadedAt: time.Now(),
		},
		{
			Number:     "12345678903",
			Status:     "PROCESSING",
			UploadedAt: time.Now(),
		},
	}

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			assert.Equal(t, userID, uid)
			return expectedOrders, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	orders, err := uc.GetOrders(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, expectedOrders, orders)
}

func TestUsecase_GetOrders_Error(t *testing.T) {
	userID := 123
	expectedErr := errors.New("database error")

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			return nil, expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	orders, err := uc.GetOrders(context.Background(), userID)

	require.Error(t, err)
	assert.Nil(t, orders)
	assert.Equal(t, expectedErr, err)
}

func TestUsecase_UploadWithdraw_Success(t *testing.T) {
	userID := 123
	withdraw := &entities.Withdraw{
		OrderNumber: "9278923470",
		Sum:         100.5,
	}

	mockRepo := &mockRepo{
		UploadWithdrawFunc: func(ctx context.Context, w *entities.Withdraw, uid int) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, withdraw.OrderNumber, w.OrderNumber)
			assert.Equal(t, withdraw.Sum, w.Sum)
			return nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadWithdraw(context.Background(), withdraw, userID)

	require.NoError(t, err)
}

func TestUsecase_UploadWithdraw_Error(t *testing.T) {
	userID := 123
	withdraw := &entities.Withdraw{
		OrderNumber: "9278923470",
		Sum:         100.5,
	}
	expectedErr := errors.New("database error")

	mockRepo := &mockRepo{
		UploadWithdrawFunc: func(ctx context.Context, w *entities.Withdraw, uid int) error {
			return expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	err := uc.UploadWithdraw(context.Background(), withdraw, userID)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestUsecase_GetWithdraw_Success(t *testing.T) {
	userID := 123
	expectedWithdraws := []entities.Withdraw{
		{
			OrderNumber: "9278923470",
			Sum:         100.5,
			UploadedAt:  time.Now(),
		},
	}

	mockRepo := &mockRepo{
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			assert.Equal(t, userID, uid)
			return expectedWithdraws, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	withdraws, err := uc.GetWithdraw(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, expectedWithdraws, withdraws)
}

func TestUsecase_GetWithdraw_Error(t *testing.T) {
	userID := 123
	expectedErr := errors.New("database error")

	mockRepo := &mockRepo{
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			return nil, expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	withdraws, err := uc.GetWithdraw(context.Background(), userID)

	require.Error(t, err)
	assert.Nil(t, withdraws)
	assert.Equal(t, expectedErr, err)
}

func TestUsecase_GetBalance_Success(t *testing.T) {
	userID := 123
	orders := []entities.Order{
		{
			Number:     "9278923470",
			Status:     "PROCESSED",
			Accrual:    floatPtr(500.0),
			UploadedAt: time.Now(),
		},
		{
			Number:     "12345678903",
			Status:     "PROCESSED",
			Accrual:    floatPtr(200.0),
			UploadedAt: time.Now(),
		},
	}
	withdraws := []entities.Withdraw{
		{
			OrderNumber: "2377225624",
			Sum:         100.0,
			UploadedAt:  time.Now(),
		},
	}

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			assert.Equal(t, userID, uid)
			return orders, nil
		},
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			assert.Equal(t, userID, uid)
			return withdraws, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	balance, err := uc.GetBalance(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, balance)
	// Ожидаемый баланс: (500.0 + 200.0) - 100.0 = 600.0
	assert.Equal(t, 600.0, balance.Current)
	assert.Equal(t, 100.0, balance.Withdrawn)
}

func TestUsecase_GetBalance_NoAccrual(t *testing.T) {
	userID := 123
	orders := []entities.Order{
		{
			Number:     "9278923470",
			Status:     "PROCESSING",
			Accrual:    nil,
			UploadedAt: time.Now(),
		},
	}
	withdraws := []entities.Withdraw{}

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			return orders, nil
		},
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			return withdraws, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	balance, err := uc.GetBalance(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, balance)
	assert.Equal(t, 0.0, balance.Current)
	assert.Equal(t, 0.0, balance.Withdrawn)
}

func TestUsecase_GetBalance_GetOrdersError(t *testing.T) {
	userID := 123
	expectedErr := errors.New("database error")

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			return nil, expectedErr
		},
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			return []entities.Withdraw{}, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	balance, err := uc.GetBalance(context.Background(), userID)

	require.Error(t, err)
	assert.Nil(t, balance)
	assert.Equal(t, expectedErr, err)
}

func TestUsecase_GetBalance_GetWithdrawError(t *testing.T) {
	userID := 123
	expectedErr := errors.New("database error")

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			return []entities.Order{}, nil
		},
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			return nil, expectedErr
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	balance, err := uc.GetBalance(context.Background(), userID)

	require.Error(t, err)
	assert.Nil(t, balance)
	assert.Equal(t, expectedErr, err)
}

func TestUsecase_GetBalance_MultipleWithdraws(t *testing.T) {
	userID := 123
	orders := []entities.Order{
		{
			Number:     "9278923470",
			Status:     "PROCESSED",
			Accrual:    floatPtr(1000.0),
			UploadedAt: time.Now(),
		},
	}
	withdraws := []entities.Withdraw{
		{
			OrderNumber: "2377225624",
			Sum:         100.0,
			UploadedAt:  time.Now(),
		},
		{
			OrderNumber: "2377225625",
			Sum:         200.0,
			UploadedAt:  time.Now(),
		},
	}

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			return orders, nil
		},
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			return withdraws, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	balance, err := uc.GetBalance(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, balance)
	// Ожидаемый баланс: 1000.0 - (100.0 + 200.0) = 700.0
	assert.Equal(t, 700.0, balance.Current)
	assert.Equal(t, 300.0, balance.Withdrawn)
}

func TestUsecase_GetBalance_NegativeBalance(t *testing.T) {
	userID := 123
	orders := []entities.Order{
		{
			Number:     "9278923470",
			Status:     "PROCESSED",
			Accrual:    floatPtr(100.0),
			UploadedAt: time.Now(),
		},
	}
	withdraws := []entities.Withdraw{
		{
			OrderNumber: "2377225624",
			Sum:         500.0,
			UploadedAt:  time.Now(),
		},
	}

	mockRepo := &mockRepo{
		GetOrdersFunc: func(ctx context.Context, uid int) ([]entities.Order, error) {
			return orders, nil
		},
		GetWithdrawFunc: func(ctx context.Context, uid int) ([]entities.Withdraw, error) {
			return withdraws, nil
		},
	}
	mockAccrualClient := &mockAccrualClient{}

	uc := setupTestUsecase(mockRepo, mockAccrualClient)

	balance, err := uc.GetBalance(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, balance)
	// Ожидаемый баланс: 100.0 - 500.0 = -400.0
	assert.Equal(t, -400.0, balance.Current)
	assert.Equal(t, 500.0, balance.Withdrawn)
}

// Вспомогательная функция для создания указателя на float64
func floatPtr(f float64) *float64 {
	return &f
}
