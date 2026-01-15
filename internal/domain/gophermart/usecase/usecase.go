package usecase

import (
	"context"
	"errors"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/delivery/accrual"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"github.com/MV7VM/diploma/internal/domain/gophermart/repository/postgres"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// -----------------------------------------------------------------------------
// Use-case layer (business-logic façade)
// -----------------------------------------------------------------------------

const (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345678"
)

type Usecase struct {
	ctx           context.Context
	log           *zap.Logger
	repo          repo
	cfg           *config.Model
	accrualClient *accrual.Client
}

type repo interface {
	RegisterUser(ctx context.Context, auth *entities.UserAuth) (int, error)
	LoginUser(ctx context.Context, auth *entities.UserAuth) (int, error)
	UploadOrder(ctx context.Context, userID int, order string) error
	GetOrders(ctx context.Context, userID int) ([]entities.Order, error)
	UploadWithdraw(ctx context.Context, withdraw *entities.Withdraw, userID int) error
	GetWithdraw(ctx context.Context, userID int) ([]entities.Withdraw, error)
	UpdateOrder(ctx context.Context, order *entities.Accrual) error
	GetAllUnProcessedOrders(ctx context.Context) ([]string, error)
}

func NewUsecase(ctx context.Context, cfg *config.Model, l *zap.Logger, repo *postgres.Repository, client *accrual.Client) (*Usecase, error) {
	return &Usecase{
		ctx:           ctx,
		cfg:           cfg,
		log:           l.Named("usecase"),
		repo:          repo,
		accrualClient: client,
	}, nil
}

func (u *Usecase) OnStart(_ context.Context) error {
	go u.accrualDaemon()
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
	repoErr := u.repo.UploadOrder(ctx, userID, order)
	if repoErr != nil && !errors.Is(repoErr, entities.ErrAlreadyInUse) {
		u.log.Error("failed to upload order", zap.Error(repoErr))
		return repoErr
	}

	orderAccrual, err := u.accrualClient.GetAccrual(u.cfg.AccrualSystem.Host, order)
	if err != nil {
		u.log.Error("failed to get order accrual", zap.Error(err))
		return nil
	}

	err = u.repo.UpdateOrder(ctx, orderAccrual)
	if err != nil {
		u.log.Error("failed to update order accrual", zap.Error(err))
		return nil
	}

	return repoErr
}

func (u *Usecase) GetOrders(ctx context.Context, userID int) ([]entities.Order, error) {
	orders, err := u.repo.GetOrders(ctx, userID)
	if err != nil {
		u.log.Error("failed to fetch orders", zap.Error(err))
		return nil, err
	}

	return orders, nil
}

func (u *Usecase) UploadWithdraw(ctx context.Context, withdraw *entities.Withdraw, userID int) error {
	//balance, err := u.GetBalance(ctx, userID)
	//if err != nil {
	//	u.log.Error("failed to fetch balance", zap.Error(err))
	//	return err
	//}
	//
	//if balance.Current < withdraw.Sum {
	//	return entities.ErrEmptyBalance
	//}

	err := u.repo.UploadWithdraw(ctx, withdraw, userID)
	if err != nil {
		u.log.Error("failed to upload withdraw", zap.Error(err))
		return err
	}

	return nil
}

func (u *Usecase) GetWithdraw(ctx context.Context, userID int) ([]entities.Withdraw, error) {
	withdraws, err := u.repo.GetWithdraw(ctx, userID)
	if err != nil {
		u.log.Error("failed to fetch withdraws", zap.Error(err))
		return nil, err
	}

	return withdraws, nil
}

func (u *Usecase) GetBalance(ctx context.Context, userID int) (res *entities.Balance, err error) {
	var (
		withdraws []entities.Withdraw
		orders    []entities.Order
	)
	res = &entities.Balance{}

	eg, ctxErr := errgroup.WithContext(ctx)
	eg.Go(func() error {
		withdraws, err = u.repo.GetWithdraw(ctxErr, userID)
		if err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		orders, err = u.repo.GetOrders(ctxErr, userID)
		if err != nil {
			return err
		}

		return nil
	})

	err = eg.Wait()
	if err != nil {
		u.log.Error("failed to fetch balance", zap.Error(err))
		return nil, err
	}

	for i := range withdraws {
		res.Withdrawn += withdraws[i].Sum
	}

	for i := range orders {
		if orders[i].Accrual != nil {
			res.Current += float64(*orders[i].Accrual)
		}
	}

	res.Current = res.Current - res.Withdrawn

	return res, nil
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
