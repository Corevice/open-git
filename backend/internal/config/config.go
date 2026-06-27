package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBType            string
	DBDSN             string
	DBAutoMigrate     bool
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	Port              string
	JWTSecret         string
	RedisAddr         string
	MinioEndpoint     string
	GitDataRoot       string
	SSHListenAddr     string
	SSHHostKeyPath    string
}

func Load() Config {
	connMaxLifetime, err := time.ParseDuration(getenv("DB_CONN_MAX_LIFETIME", "1h"))
	if err != nil {
		connMaxLifetime = time.Hour
	}

	return Config{
		DBType:            getenv("DB_TYPE", "sqlite"),
		DBDSN:             os.Getenv("DB_DSN"),
		DBAutoMigrate:     getenvBool("DB_AUTO_MIGRATE", false),
		DBMaxOpenConns:    getenvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getenvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: connMaxLifetime,
		Port:              getenv("PORT", "8080"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		RedisAddr:         os.Getenv("REDIS_ADDR"),
		MinioEndpoint:     os.Getenv("MINIO_ENDPOINT"),
		GitDataRoot:       getenv("GIT_DATA_ROOT", "./data/git"),
		SSHListenAddr:     getenv("SSH_LISTEN_ADDR", ":2222"),
		SSHHostKeyPath:    getenv("SSH_HOST_KEY_PATH", "./data/ssh_host_rsa_key"),
	}
}

func (c Config) Validate() error {
	if c.DBType != "postgres" && c.DBType != "sqlite" {
		return fmt.Errorf("DB_TYPE must be \"postgres\" or \"sqlite\", got %q", c.DBType)
	}

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

func getenvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func getenvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
