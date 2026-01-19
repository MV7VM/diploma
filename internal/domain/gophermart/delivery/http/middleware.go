package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

func (s *Server) withLogger(c *gin.Context) {

	startTime := time.Now()

	c.Next()

	s.logger.Info("",
		zap.String("uri", c.Request.RequestURI),
		zap.String("method", c.Request.Method),
		zap.Any("duration", time.Since(startTime)),
	)

}

func (s *Server) auth(c *gin.Context) {
	_, err := c.Cookie("auth")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = s.parseToken(c)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Next()
}

func (s *Server) parseToken(c *gin.Context) error {
	cookie, err := c.Cookie("auth")
	if err != nil {
		return err
	}

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.HTTP.SecretToken), nil
	})
	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		for key, value := range claims {
			c.Set(key, value)
		}
	}

	return nil
}
