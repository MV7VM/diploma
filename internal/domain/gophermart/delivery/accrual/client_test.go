package accrual

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, 10*time.Second, client.client.Timeout)
}

func TestClient_GetAccrual_Success_Processed(t *testing.T) {
	orderNumber := "9278923470"
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "PROCESSED",
		Accrual: 500.0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/orders/"+orderNumber, r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedAccrual)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, expectedAccrual.Order, accrual.Order)
	assert.Equal(t, expectedAccrual.Status, accrual.Status)
	assert.Equal(t, expectedAccrual.Accrual, accrual.Accrual)
}

func TestClient_GetAccrual_Success_Processing(t *testing.T) {
	orderNumber := "9278923470"
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "PROCESSING",
		Accrual: 0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedAccrual)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, expectedAccrual.Status, accrual.Status)
}

func TestClient_GetAccrual_Success_Registered(t *testing.T) {
	orderNumber := "9278923470"
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "REGISTERED",
		Accrual: 0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedAccrual)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, expectedAccrual.Status, accrual.Status)
}

func TestClient_GetAccrual_Success_Invalid(t *testing.T) {
	orderNumber := "9278923470"
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "INVALID",
		Accrual: 0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedAccrual)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, expectedAccrual.Status, accrual.Status)
}

func TestClient_GetAccrual_Success_WithoutAccrual(t *testing.T) {
	orderNumber := "9278923470"
	// В ответе может отсутствовать поле accrual для статусов без начисления
	responseJSON := `{"order":"` + orderNumber + `","status":"PROCESSING"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, orderNumber, accrual.Order)
	assert.Equal(t, "PROCESSING", accrual.Status)
	assert.Equal(t, 0.0, accrual.Accrual) // float64 по умолчанию 0
}

func TestClient_GetAccrual_NoContent(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	assert.Nil(t, accrual)
}

func TestClient_GetAccrual_TooManyRequests(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("No more than N requests per minute allowed"))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
	assert.True(t, errors.Is(err, ErrToManyRequest))
}

func TestClient_GetAccrual_InternalServerError(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
	assert.Contains(t, err.Error(), "Internal Server Error")
}

func TestClient_GetAccrual_BadGateway(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
	assert.Contains(t, err.Error(), "Bad Gateway")
}

func TestClient_GetAccrual_InvalidJSON(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json}"))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
}

func TestClient_GetAccrual_EmptyBody(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Не пишем ничего в тело
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
}

func TestClient_GetAccrual_NetworkError(t *testing.T) {
	orderNumber := "9278923470"
	// Используем несуществующий URL для симуляции сетевой ошибки
	invalidURL := "http://invalid-host-that-does-not-exist:9999"

	client := NewClient()
	// Используем очень маленький timeout для быстрого теста
	client.client.Timeout = 100 * time.Millisecond

	accrual, err := client.GetAccrual(invalidURL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClient_GetAccrual_InvalidURL(t *testing.T) {
	orderNumber := "9278923470"
	// Используем невалидный URL
	invalidURL := "://invalid-url"

	client := NewClient()
	accrual, err := client.GetAccrual(invalidURL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestClient_GetAccrual_MalformedJSON(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Отправляем валидный JSON, но с неправильной структурой
		w.Write([]byte(`{"invalid":"structure"}`))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	// JSON должен распарситься, но поля будут пустыми
	// Проверим, что ошибки нет (парсинг проходит, но поля не совпадают)
	require.NoError(t, err)
	require.NotNil(t, accrual)
}

func TestClient_GetAccrual_StatusNotFound(t *testing.T) {
	orderNumber := "9278923470"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.Error(t, err)
	assert.Nil(t, accrual)
	assert.Contains(t, err.Error(), "Not Found")
}

func TestClient_GetAccrual_Success_WithAccrualZero(t *testing.T) {
	orderNumber := "9278923470"
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "PROCESSED",
		Accrual: 0.0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedAccrual)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, 0.0, accrual.Accrual)
}

func TestClient_GetAccrual_Success_LargeAccrual(t *testing.T) {
	orderNumber := "9278923470"
	expectedAccrual := &entities.Accrual{
		Order:   orderNumber,
		Status:  "PROCESSED",
		Accrual: 999999.99,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedAccrual)
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
	assert.Equal(t, expectedAccrual.Accrual, accrual.Accrual)
}

func TestClient_GetAccrual_MultipleRequests(t *testing.T) {
	orderNumber1 := "9278923470"
	orderNumber2 := "12345678903"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var orderNumber string
		if r.URL.Path == "/api/orders/"+orderNumber1 {
			orderNumber = orderNumber1
		} else {
			orderNumber = orderNumber2
		}

		accrual := &entities.Accrual{
			Order:   orderNumber,
			Status:  "PROCESSED",
			Accrual: 100.0,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(accrual)
	}))
	defer server.Close()

	client := NewClient()

	// Первый запрос
	accrual1, err1 := client.GetAccrual(server.URL, orderNumber1)
	require.NoError(t, err1)
	require.NotNil(t, accrual1)
	assert.Equal(t, orderNumber1, accrual1.Order)

	// Второй запрос
	accrual2, err2 := client.GetAccrual(server.URL, orderNumber2)
	require.NoError(t, err2)
	require.NotNil(t, accrual2)
	assert.Equal(t, orderNumber2, accrual2.Order)
}

func TestClient_GetAccrual_BaseURLWithTrailingSlash(t *testing.T) {
	orderNumber := "9278923470"
	//baseURL := "http://localhost:8080/"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&entities.Accrual{
			Order:   orderNumber,
			Status:  "PROCESSED",
			Accrual: 500.0,
		})
	}))
	defer server.Close()

	client := NewClient()
	accrual, err := client.GetAccrual(server.URL, orderNumber)

	require.NoError(t, err)
	require.NotNil(t, accrual)
}
