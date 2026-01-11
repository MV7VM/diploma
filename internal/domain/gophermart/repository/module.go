package repository

import (
	"github.com/MV7VM/diploma/internal/domain/gophermart/repository/postgres"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Module("repository",
		fx.Provide(
			postgres.NewRepository,
		),
		fx.Invoke(
			func(lc fx.Lifecycle, s *postgres.Repository) {
				lc.Append(fx.Hook{
					OnStart: s.OnStart,
					OnStop:  s.OnStop,
				})
			},
		),
	)
}
