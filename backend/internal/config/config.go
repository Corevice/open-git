package config

import "os"

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

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
