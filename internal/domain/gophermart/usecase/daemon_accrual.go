package usecase

import (
	"errors"
	"time"

	"github.com/MV7VM/diploma/internal/domain/gophermart/delivery/accrual"
	"github.com/MV7VM/diploma/internal/domain/gophermart/entities"
	"go.uber.org/zap"
)

func (u *Usecase) accrualDaemon() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		orderNumbers, err := u.repo.GetAllUnProcessedOrders(u.ctx)
		if err != nil {
			u.log.Error("failed to get unprocessed orders", zap.Error(err))
			continue
		}

		for i := range orderNumbers {
			res, err := u.accrualClient.GetAccrual(u.cfg.AccrualSystem.Host, orderNumbers[i])
			u.log.Info("getting accrual order", zap.String("order", orderNumbers[i]), zap.Any("res", res), zap.Error(err))
			if err != nil && errors.Is(err, accrual.ErrToManyRequest) {
				time.Sleep(1 * time.Minute)
			} else if err != nil && !errors.Is(err, accrual.ErrToManyRequest) {
				res = &entities.Accrual{Order: orderNumbers[i], Status: entities.OrderStatusInvalid}
			}

			err = u.repo.UpdateOrder(u.ctx, res)
			if err != nil {
				u.log.Error("failed to update order", zap.String("order", orderNumbers[i]), zap.Error(err))
				continue
			}
		}
	}
}
