// Package http implements the public REST API facade over the business use-case
// layer.  All endpoints are grouped under the legacy prefix "/app" for mobile
// backward-compatibility.
package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/MV7VM/diploma/internal/domain/gophermart/usecase"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	logger *zap.Logger
	serv   *gin.Engine
	cfg    *config.Model
	uc     uc
}

type uc interface {
	Register(ctx context.Context, creds *entities.UserAuth) (string, error)
	Login(ctx context.Context, creds *entities.UserAuth) (string, error)
	UploadOrder(ctx context.Context, userID int, order string) error
	GetOrders(ctx context.Context, userID int) ([]entities.Order, error)
}

// NewServer wires up Gin, logging and use-case dependencies.
func NewServer(logger *zap.Logger, cfg *config.Model, uc *usecase.Usecase) (*Server, error) {
	return &Server{
		logger: logger,
		serv:   gin.Default(),
		uc:     uc,
		cfg:    cfg,
	}, nil
}

// OnStart registers routes and launches an HTTP listener in a goroutine.
func (s *Server) OnStart(_ context.Context) error {
	go func() {
		s.createController()

		s.logger.Info("HTTP server started", zap.String("addr", s.cfg.HTTP.Host))
		if err := s.serv.Run(s.cfg.HTTP.Host); err != nil {
			s.logger.Error("HTTP server exited", zap.Error(err))
		}
	}()

	return nil
}

// OnStop is a no-op here (Gin has no explicit shutdown hook).
func (s *Server) OnStop(_ context.Context) error {
	s.logger.Info("HTTP server stopped")
	return nil
}

func (s *Server) Register(c *gin.Context) {
	usrCred := &entities.UserAuth{}
	err := c.ShouldBindJSON(usrCred)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := s.uc.Register(c, usrCred)
	if err != nil {
		if errors.Is(err, entities.ErrAlreadyInUse) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("auth", token, 3600, "", "", false, true)
	c.Header("Authorization", token)
	c.Status(http.StatusOK)
}

func (s *Server) Login(c *gin.Context) {
	usrCred := &entities.UserAuth{}
	err := c.ShouldBindJSON(usrCred)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := s.uc.Login(c, usrCred)
	if err != nil {
		if errors.Is(err, entities.ErrWrongCredentials) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("auth", token, 36000, "", "", false, true)
	c.Header("Authorization", token)
}

func (s *Server) UploadOrder(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to read request body: " + err.Error(),
		})
		return
	}

	num := strings.TrimSpace(string(body))
	if !validateOrderNumber(num) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "invalid order number",
		})
		return
	}

	uID := c.GetFloat64("userID")
	fmt.Println(int(uID))

	err = s.uc.UploadOrder(c, int(c.GetFloat64("userID")), num)
	switch {
	case errors.Is(err, entities.ErrAlreadyInUse):
		c.AbortWithStatus(http.StatusOK)
		return
	case errors.Is(err, entities.ErrPermissionDenied):
		c.AbortWithStatus(http.StatusConflict)
		return
	//case errors.Is(err, entities.ErrAlreadyInUse):
	//	c.AbortWithStatus(http.StatusOK)
	//	return
	case err == nil:
		c.AbortWithStatus(http.StatusAccepted)
		return
	default:
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (s *Server) GetOrders(c *gin.Context) {
	orders, err := s.uc.GetOrders(c, int(c.GetFloat64("userID")))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(orders) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, orders)
}

func validateOrderNumber(number string) bool {
	// Удаляем все пробелы и нецифровые символы
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, number)

	// Проверяем минимальную длину (обычно от 2 цифр)
	if len(cleaned) < 2 {
		return false
	}

	sum := 0
	isSecond := false

	// Идем по цифрам справа налево
	for i := len(cleaned) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(cleaned[i]))
		if err != nil {
			return false // Если есть нецифровые символы
		}

		if isSecond {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	// Если сумма делится на 10 без остатка - номер валиден
	return sum%10 == 0
}

//
//func (s *Server) CreateShortURL(c *gin.Context) {
//	// Получаем raw body
//	body, err := c.GetRawData()
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "failed to read request body: " + err.Error(),
//		})
//		return
//	}
//
//	url := strings.TrimSpace(string(body))
//	if !validateURL(url) {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "invalid url",
//		})
//		return
//	}
//
//	shortURL, conflict, err := s.uc.CreateShortURL(c.Request.Context(), url)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": err.Error(),
//		})
//		return
//	}
//
//	if conflict {
//		c.String(http.StatusConflict, s.cfg.HTTP.ReturningURL+shortURL)
//		return
//	}
//
//	c.String(http.StatusCreated, s.cfg.HTTP.ReturningURL+shortURL)
//}
//
//type CreateShortURLByBodyReq struct {
//	URL string `json:"url"`
//}
//
//type CreateShortURLByBodyResp struct {
//	ShortURL string `json:"result"`
//}
//
//func (s *Server) CreateShortURLByBody(c *gin.Context) {
//	// Получаем raw body
//	body, err := io.ReadAll(c.Request.Body)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "Failed to read request body",
//		})
//		return
//	}
//
//	// Декодируем JSON в структуру
//	var reqBody CreateShortURLByBodyReq
//
//	err = json.Unmarshal(body, &reqBody)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "Invalid JSON format",
//		})
//		return
//	}
//
//	url := strings.TrimSpace(reqBody.URL)
//	if !validateURL(url) {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "invalid url",
//		})
//		return
//	}
//
//	shortURL, conflict, err := s.uc.CreateShortURL(c.Request.Context(), url)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": err.Error(),
//		})
//		return
//	}
//
//	if conflict {
//		c.JSON(http.StatusConflict, CreateShortURLByBodyResp{
//			ShortURL: s.cfg.HTTP.ReturningURL + shortURL,
//		})
//		return
//	}
//
//	c.JSON(http.StatusCreated, CreateShortURLByBodyResp{
//		ShortURL: s.cfg.HTTP.ReturningURL + shortURL,
//	})
//}
//
//func (s *Server) GetByID(c *gin.Context) {
//	id := c.Param("id")
//
//	url, err := s.uc.GetByID(c, id)
//	if err != nil {
//		s.logger.Error("failed to get url", zap.String("url", id), zap.Error(err))
//		c.AbortWithStatus(http.StatusBadRequest)
//		return
//	}
//
//	c.Header("Location", url)
//
//	c.Status(http.StatusTemporaryRedirect)
//}
//
//func (s *Server) Ping(c *gin.Context) {
//	err := s.uc.Ping(c)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{
//			"error": err.Error(),
//		})
//		return
//	}
//
//	c.Status(http.StatusOK)
//}
//
//func (s *Server) BatchURL(c *gin.Context) {
//	var batchedReq []entities.BatchItem
//	if err := c.ShouldBindJSON(&batchedReq); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "failed to read request body",
//		})
//		return
//	}
//
//	if len(batchedReq) == 0 {
//		c.JSON(http.StatusBadRequest, gin.H{
//			"error": "batch payload is empty",
//		})
//		return
//	}
//
//	err := s.uc.BatchURLs(c, batchedReq)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{
//			"error": err.Error(),
//		})
//		return
//	}
//
//	for i := range batchedReq {
//		batchedReq[i].ShortURL = s.cfg.HTTP.ReturningURL + batchedReq[i].ShortURL
//	}
//
//	c.JSON(http.StatusCreated, batchedReq)
//}
//
//func validateURL(urlStr string) bool {
//	urlStr = strings.TrimSpace(urlStr)
//	if urlStr == "" {
//		return false
//	}
//
//	// Пытаемся распарсить URL
//	u, err := url.Parse(urlStr)
//	if err != nil {
//		return false
//	}
//
//	// Если нет схемы, добавляем http:// и пытаемся снова
//	if u.Scheme == "" {
//		u, err = url.Parse("http://" + urlStr)
//		if err != nil {
//			return false
//		}
//	}
//
//	// Проверяем, что есть host
//	if u.Host == "" {
//		return false
//	}
//
//	// Проверяем, что схема поддерживается
//	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
//		return false
//	}
//
//	return true
//}
