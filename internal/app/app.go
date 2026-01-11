package app

import (
	"context"

	"github.com/MV7VM/diploma/internal/config"
	"github.com/MV7VM/diploma/internal/domain/gophermart/delivery/http"
	"github.com/MV7VM/diploma/internal/domain/gophermart/repository"
	"github.com/MV7VM/diploma/internal/domain/gophermart/usecase"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func New() *fx.App {
	return fx.New(
		fx.Options(
			repository.New(), //
			usecase.New(),
			http.New(),
		),
		fx.Provide(
			config.NewConfig,
			context.Background,
			zap.NewDevelopment,
		),
		fx.WithLogger(
			func(log *zap.Logger) fxevent.Logger {
				return &fxevent.ZapLogger{Logger: log}
			},
		),
	)
}
