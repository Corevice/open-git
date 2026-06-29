package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	MinioAccessKey    string
	MinioSecretKey    string
	MinioUseTLS       bool
	MinioBucket       string
	GitDataRoot       string
	SSHPort           string
	SSHEnabled        bool
	SSHListenAddr     string
	SSHHostKeyPath    string
	APIBaseURL        string
	WebBaseURL        string
	DocsBaseURL        string
	WebhookSecretKey   string
	MetricsEnabled      bool
	MetricsPath         string
	MetricsAuthToken    string
	Domain              string
	ACMEEmail           string
	TLSMode             string // acme | custom | selfsigned
	TLSCertFile         string
	TLSKeyFile          string
	TrustedProxyCIDRs   string
}

func Load() Config {
	connMaxLifetime, err := time.ParseDuration(getenv("DB_CONN_MAX_LIFETIME", "1h"))
	if err != nil {
		connMaxLifetime = time.Hour
	}

	sshPort := getenv("SSH_PORT", "2222")
	sshListenAddr := os.Getenv("SSH_LISTEN_ADDR")
	if sshListenAddr == "" {
		if strings.HasPrefix(sshPort, ":") {
			sshListenAddr = sshPort
		} else {
			sshListenAddr = ":" + sshPort
		}
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
		MinioAccessKey:    getenv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:    getenv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseTLS:       getenvBool("MINIO_USE_TLS", false),
		MinioBucket:       getenv("MINIO_BUCKET", "artifacts"),
		GitDataRoot:       getenv("GIT_DATA_ROOT", "./data/git"),
		SSHPort:           sshPort,
		SSHEnabled:        getenvBool("SSH_ENABLED", true),
		SSHListenAddr:     sshListenAddr,
		SSHHostKeyPath:    getenv("SSH_HOST_KEY_PATH", "./data/ssh_host_rsa_key"),
		APIBaseURL:        getenv("API_BASE_URL", "http://localhost:8080/api/v3"),
		WebBaseURL:        getenv("WEB_BASE_URL", "http://localhost:8080"),
		DocsBaseURL:       getenv("DOCS_BASE_URL", "https://docs.github.com/rest"),
		WebhookSecretKey:  os.Getenv("WEBHOOK_SECRET_KEY"),
		MetricsEnabled:      getenvBool("METRICS_ENABLED", true),
		MetricsPath:         getenv("METRICS_PATH", "/metrics"),
		MetricsAuthToken:    os.Getenv("METRICS_AUTH_TOKEN"),
		Domain:              os.Getenv("DOMAIN"),
		ACMEEmail:           os.Getenv("ACME_EMAIL"),
		TLSMode:             getenv("TLS_MODE", "acme"),
		TLSCertFile:         os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:          os.Getenv("TLS_KEY_FILE"),
		TrustedProxyCIDRs:   os.Getenv("TRUSTED_PROXY_CIDRS"),
	}
}

func (c *Config) Validate() error {
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
	if c.MetricsEnabled && !strings.HasPrefix(c.MetricsPath, "/") {
		log.Printf("METRICS_PATH %q invalid, falling back to /metrics", c.MetricsPath)
		c.MetricsPath = "/metrics"
	}
	if c.DBType == "postgres" && c.DBDSN == "" {
		return fmt.Errorf("DB_DSN is required when DB_TYPE is postgres")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Domain == "" {
		return fmt.Errorf("DOMAIN is required")
	}
	switch c.TLSMode {
	case "acme", "custom", "selfsigned":
	default:
		return fmt.Errorf("TLS_MODE must be \"acme\", \"custom\", or \"selfsigned\", got %q", c.TLSMode)
	}
	if c.TLSMode == "acme" {
		acmeEmailPattern := regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
		if c.ACMEEmail == "" {
			return fmt.Errorf("ACME_EMAIL is required when TLS_MODE is acme")
		}
		if !acmeEmailPattern.MatchString(c.ACMEEmail) {
			return fmt.Errorf("ACME_EMAIL must be a valid email address")
		}
	}
	if c.TLSMode == "custom" {
		if c.TLSCertFile == "" {
			return fmt.Errorf("TLS_CERT_FILE is required when TLS_MODE is custom")
		}
		if c.TLSKeyFile == "" {
			return fmt.Errorf("TLS_KEY_FILE is required when TLS_MODE is custom")
		}
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
