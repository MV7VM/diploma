package accrual

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
)

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

const (
	getAccrualPath = "/api/orders/"
)

var (
	ErrUnknownOrder  entities.MyError = "Unknown Order"
	ErrToManyRequest entities.MyError = "To Many Request"
)

// GetAccrual получает информацию о расчёте начислений баллов лояльности для заказа.
// Возвращает:
// - *Accrual и nil при успешном ответе (200 OK)
// - nil и nil при ответе 204 (заказ не зарегистрирован)
// - nil и ErrToManyRequest при ответе 429 (превышен лимит запросов)
// - nil и error при других ошибках (500, сетевые ошибки и т.д.)
func (c *Client) GetAccrual(baseURL, orderNumber string) (*entities.Accrual, error) {
	url := baseURL + getAccrualPath + orderNumber

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	log.Println(url, response, err)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// Читаем тело ответа
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		// Парсим JSON
		var result entities.Accrual
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		return &result, nil
	case http.StatusNoContent:
		return nil, nil
	case http.StatusTooManyRequests:
		return nil, ErrToManyRequest
	default:
		return nil, errors.New(response.Status)
	}
}
