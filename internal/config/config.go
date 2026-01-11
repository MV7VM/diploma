package config

import (
	"flag"
	"os"

	"github.com/gofrs/uuid"
)

func NewConfig() (*Model, error) {
	var cfg Model

	flag.StringVar(&cfg.HTTP.Host, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.Repo.PsqlConnString, "d", "postgresql://postgres:password@localhost:5432/postgres", "file for recovery storage")
	flag.StringVar(&cfg.AccrualSystem.Host, "r", "localhost:8082", "file for recovery storage")

	flag.Parse()

	if filePath := os.Getenv("RUN_ADDRESS"); filePath != "" {
		cfg.HTTP.Host = filePath
	}

	if dbConn := os.Getenv("DATABASE_URI"); dbConn != "" {
		cfg.Repo.PsqlConnString = dbConn
	}

	if client := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); client != "" {
		cfg.AccrualSystem.Host = client
	}

	secretKey, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	cfg.HTTP.SecretToken = secretKey.String()

	passKey, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	cfg.PasswordCryptoKey = passKey.String()

	return &cfg, nil
}
