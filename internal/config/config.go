package config

import (
	"flag"
	"os"
)

func NewConfig() (*Model, error) {
	var cfg Model

	flag.StringVar(&cfg.HTTP.Host, "a", "localhost:8082", "address and port to run server")
	flag.StringVar(&cfg.Repo.PsqlConnString, "d", "postgresql://postgres:password@localhost:5432/postgres", "db connection string")
	flag.StringVar(&cfg.AccrualSystem.Host, "r", "http://localhost:8080", "accrual system host")
	flag.StringVar(&cfg.HTTP.SecretToken, "s", "019bd840-49f0-7d1c-9279-7792e0bf1f4e", "private key for jwt")
	flag.StringVar(&cfg.PasswordCryptoKey, "p", "019bd840-8098-7b46-835b-16c3321d112d", "password crypto key")

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

	if jwtKey := os.Getenv("JWT_PRIVATE_KEY"); jwtKey != "" {
		cfg.HTTP.SecretToken = jwtKey
	}

	if passKey := os.Getenv("PASSWORD_CRYPTO_KEY"); passKey != "" {
		cfg.AccrualSystem.Host = passKey
	}

	//secretKey, err := uuid.NewV7()
	//if err != nil {
	//	return nil, err
	//}
	//
	//cfg.HTTP.SecretToken = secretKey.String()
	//
	//passKey, err := uuid.NewV7()
	//if err != nil {
	//	return nil, err
	//}
	//
	//cfg.PasswordCryptoKey = passKey.String()

	return &cfg, nil
}
