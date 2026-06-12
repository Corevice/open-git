package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBType        string
	DBDSN         string
	Port          string
	JWTSecret     string
	RedisAddr     string
	MinioEndpoint string
}

func Load() Config {
	return Config{
		DBType:        getenv("DB_TYPE", "sqlite"),
		DBDSN:         os.Getenv("DB_DSN"),
		Port:          getenv("PORT", "8080"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		MinioEndpoint: os.Getenv("MINIO_ENDPOINT"),
	}
}

func (c Config) Validate() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return fmt.Errorf("port: invalid integer %q", c.Port)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port: out of range %d", port)
	}
	if c.DBType == "postgres" && c.DBDSN == "" {
		return fmt.Errorf("DB_DSN is required when DB_TYPE is postgres")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
