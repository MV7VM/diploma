package accrual

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
)

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
	}
}

const (
	getAccuralPath                    = "/api/orders/"
	ErrUnknownOrder  entities.MyError = "Unknown Order"
	ErrToManyRequest entities.MyError = "To Many Request"
)

func (c *Client) GetAccrual(url, number string) (*entities.Accrual, error) {
	response, err := c.client.Get(url + getAccuralPath + number)
	if err != nil {
		return nil, err
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

		// Проверяем, что статус PROCESSED и возвращаем значения
		if result.Status == "PROCESSED" {
			return nil, nil
		}

		// Если статус не PROCESSED, возвращаем статус и 0
		return &result, nil
	case http.StatusNoContent:
		return nil, nil
	case http.StatusTooManyRequests:
		return nil, ErrToManyRequest
	default:
		return nil, errors.New(response.Status)
	}
}
