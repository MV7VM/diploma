package usecase

import (
	"context"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/MV7VM/diploma/internal/domain/gophermart/repository/postgres"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------
// Use-case layer (business-logic façade)
// -----------------------------------------------------------------------------

const (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345678"
)

type Usecase struct {
	log  *zap.Logger
	repo repo
	cfg  *config.Model
}

type repo interface {
	RegisterUser(ctx context.Context, auth *entities.UserAuth) (int, error)
	LoginUser(ctx context.Context, auth *entities.UserAuth) (int, error)
	UploadOrder(ctx context.Context, userID int, order string) error
	GetOrders(ctx context.Context) ([]entities.Order, error)
}

func NewUsecase(cfg *config.Model, l *zap.Logger, repo *postgres.Repository) (*Usecase, error) {
	return &Usecase{cfg: cfg, log: l.Named("usecase"), repo: repo}, nil
}

func (u *Usecase) OnStart(ctx context.Context) error {
	return nil
}

func (u *Usecase) Register(ctx context.Context, creds *entities.UserAuth) (string, error) {
	userID, err := u.repo.RegisterUser(ctx, creds)
	if err != nil {
		u.log.Error("failed to register user", zap.Error(err))
		return "", err
	}

	token, err := u.createToken(userID)
	if err != nil {
		u.log.Error("failed to create token", zap.Error(err))
		return "", err
	}

	return token, nil
}

func (u *Usecase) Login(ctx context.Context, creds *entities.UserAuth) (string, error) {
	userID, err := u.repo.LoginUser(ctx, creds)
	if err != nil {
		u.log.Error("failed to login user", zap.Error(err))
		return "", err
	}

	token, err := u.createToken(userID)
	if err != nil {
		u.log.Error("failed to create token", zap.Error(err))
		return "", err
	}

	return token, nil
}

func (u *Usecase) UploadOrder(ctx context.Context, userID int, order string) error {
	err := u.repo.UploadOrder(ctx, userID, order)
	if err != nil {
		u.log.Error("failed to upload order", zap.Error(err))
		return err
	}

	return nil
}

func (u *Usecase) GetOrders(ctx context.Context) ([]entities.Order, error) {
	orders, err := u.repo.GetOrders(ctx)
	if err != nil {
		u.log.Error("failed to fetch orders", zap.Error(err))
		return nil, err
	}

	return orders, nil
}

func (u *Usecase) createToken(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
	})

	signedString, err := token.SignedString([]byte(u.cfg.HTTP.SecretToken))
	if err != nil {
		return "", err
	}

	return signedString, nil
}
