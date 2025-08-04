package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Validate validates the configuration
func Validate(cfg Config) error {
	if err := validateApp(cfg.App()); err != nil {
		return fmt.Errorf("app config validation failed: %w", err)
	}

	if err := validateServer(cfg.Server()); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	if err := validateDatabase(cfg.Database()); err != nil {
		return fmt.Errorf("database config validation failed: %w", err)
	}

	if err := validateRedis(cfg.Redis()); err != nil {
		return fmt.Errorf("redis config validation failed: %w", err)
	}

	if err := validateCache(cfg.Cache()); err != nil {
		return fmt.Errorf("cache config validation failed: %w", err)
	}

	if err := validateLogger(cfg.Logger()); err != nil {
		return fmt.Errorf("logger config validation failed: %w", err)
	}

	if err := validateOTP(cfg.OTP()); err != nil {
		return fmt.Errorf("otp config validation failed: %w", err)
	}

	if err := validateUpload(cfg.Upload()); err != nil {
		return fmt.Errorf("upload config validation failed: %w", err)
	}

	if err := validateExternal(cfg.External()); err != nil {
		return fmt.Errorf("external config validation failed: %w", err)
	}
	if err := validateRPC(cfg.RPC()); err != nil {
		return fmt.Errorf("rpc config validation failed: %w", err)
	}
	return nil
}

func validateApp(cfg AppConfig) error {
	if cfg.Environment() == "" {
		return fmt.Errorf("environment variable is required, please set ENV env variable")
	}

	switch cfg.Environment() {
	case LocalEnv, DevelopmentEnv, ProductionEnv:
	default:
		return fmt.Errorf("ENV=%s is invalid, only accept `%s`, `%s`, `%s`", cfg.Environment(), LocalEnv, DevelopmentEnv, ProductionEnv)
	}

	if cfg.TokenIssuer() == "" {
		return fmt.Errorf("token_issuer is required")
	}

	if cfg.AccessTokenExpiresIn() <= 0 {
		return fmt.Errorf("access_token_expires_in must be positive")
	}

	if cfg.RefreshTokenExpiresIn() <= 0 {
		return fmt.Errorf("refresh_token_expires_in must be positive")
	}

	if cfg.SessionLimitPerUser() <= 0 {
		return fmt.Errorf("session_limit_per_user must be positive")
	}

	// Validate JWT access token is shorter than refresh token
	if cfg.AccessTokenExpiresIn() >= cfg.RefreshTokenExpiresIn() {
		return fmt.Errorf("access_token_expires_in must be less than refresh_token_expires_in")
	}

	if cfg.AccessTokenSecret() == "" {
		return fmt.Errorf("access token secret is required, please set ACCESS_TOKEN_SECRET env variable")
	}

	if cfg.RefreshTokenSecret() == "" {
		return fmt.Errorf("refresh token secret is required, please set REFRESH_TOKEN_SECRET env variable")
	}

	if cfg.APIKey() == "" {
		return fmt.Errorf("api key is required, please set API_KEY env variable")
	}

	if cfg.SystemAdminDefaultPhone() == "" {
		return fmt.Errorf("system admin default phone is required, please set SYSTEM_ADMIN_DEFAULT_PHONE env variable")
	}

	if cfg.SystemAdminDefaultEmail() == "" {
		return fmt.Errorf("system admin default email is required, please set SYSTEM_ADMIN_DEFAULT_EMAIL env variable")
	}

	if cfg.SystemAdminDefaultPassword() == "" {
		return fmt.Errorf("system admin default password is required, please set SYSTEM_ADMIN_DEFAULT_PASSWORD env variable")
	}

	return nil
}

func validateServer(cfg ServerConfig) error {
	if cfg.Host() == "" {
		return fmt.Errorf("host is required")
	}

	// Validate host format
	if cfg.Host() != "0.0.0.0" && cfg.Host() != "localhost" {
		if net.ParseIP(cfg.Host()) == nil {
			return fmt.Errorf("host must be a valid IP address or 'localhost'")
		}
	}

	if cfg.Port() <= 0 || cfg.Port() > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if cfg.ReadTimeout() <= 0 {
		return fmt.Errorf("read_timeout must be positive")
	}

	if cfg.WriteTimeout() <= 0 {
		return fmt.Errorf("write_timeout must be positive")
	}

	// Validate domain format
	if cfg.Domain() != "" && !strings.HasPrefix(cfg.Domain(), "http") {
		return fmt.Errorf("domain must start with http:// or https://")
	}

	return nil
}

func validateDatabase(cfg DatabaseConfig) error {
	if cfg.Host() == "" {
		return fmt.Errorf("database host is required")
	}

	if cfg.Port() == "" {
		return fmt.Errorf("database port is required")
	}

	// Validate port is numeric
	if port, err := strconv.Atoi(cfg.Port()); err != nil {
		return fmt.Errorf("database port must be numeric: %w", err)
	} else if port <= 0 || port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}

	if cfg.User() == "" {
		return fmt.Errorf("database user is required")
	}

	if cfg.Password() == "" {
		return fmt.Errorf("database password is required")
	}

	if cfg.Name() == "" {
		return fmt.Errorf("database name is required")
	}

	if cfg.MaxOpenConns() <= 0 {
		return fmt.Errorf("max_open_conns must be positive")
	}

	if cfg.MaxIdleConns() <= 0 {
		return fmt.Errorf("max_idle_conns must be positive")
	}

	if cfg.MaxIdleConns() > cfg.MaxOpenConns() {
		return fmt.Errorf("max_idle_conns cannot be greater than max_open_conns")
	}

	if cfg.ConnMaxLifetime() <= 0 {
		return fmt.Errorf("conn_max_lifetime must be positive")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	isValidSSL := false
	for _, mode := range validSSLModes {
		if cfg.SSLMode() == mode {
			isValidSSL = true
			break
		}
	}
	if !isValidSSL {
		return fmt.Errorf("ssl_mode must be one of: %s", strings.Join(validSSLModes, ", "))
	}

	// Validate log level
	if cfg.EnableLog() {
		validLogLevels := []string{"silent", "error", "warn", "info"}
		isValidLogLevel := false
		for _, level := range validLogLevels {
			if cfg.LogLevel() == level {
				isValidLogLevel = true
				break
			}
		}
		if !isValidLogLevel {
			return fmt.Errorf("database log_level must be one of: %s", strings.Join(validLogLevels, ", "))
		}
	}

	return nil
}

func validateRedis(cfg RedisConfig) error {
	if cfg.Host() == "" {
		return fmt.Errorf("redis host is required")
	}

	if cfg.Port() <= 0 || cfg.Port() > 65535 {
		return fmt.Errorf("redis port must be between 1 and 65535")
	}

	if cfg.DB() < 0 || cfg.DB() > 15 {
		return fmt.Errorf("redis db must be between 0 and 15")
	}

	return nil
}

func validateCache(cfg CacheConfig) error {
	provider := cfg.Provider()
	validProviders := []string{"redis", "memory"}
	isValid := false
	for _, p := range validProviders {
		if provider == p {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("cache provider must be one of: %s", strings.Join(validProviders, ", "))
	}

	if cfg.DefaultTTL() <= 0 {
		return fmt.Errorf("default_ttl must be positive")
	}

	return nil
}

func validateLogger(cfg LoggerConfig) error {
	if cfg.LogFilePath() == "" {
		return fmt.Errorf("log_file_path is required")
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(cfg.LogFilePath(), 0755); err != nil {
		return fmt.Errorf("cannot create log directory: %w", err)
	}

	if cfg.LogFileName() == "" {
		return fmt.Errorf("log_file_name is required")
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	isValidLevel := false
	for _, level := range validLevels {
		if cfg.LogLevel() == level {
			isValidLevel = true
			break
		}
	}
	if !isValidLevel {
		return fmt.Errorf("log_level must be one of: %s", strings.Join(validLevels, ", "))
	}

	if cfg.MaxFileSizeMB() <= 0 {
		return fmt.Errorf("max_file_size_mb must be positive")
	}

	if cfg.MaxFileAgeDays() <= 0 {
		return fmt.Errorf("max_file_age_days must be positive")
	}

	if cfg.MaxBackupFiles() <= 0 {
		return fmt.Errorf("max_backup_files must be positive")
	}

	// Validate timestamp format by trying to format current time
	if cfg.TimestampFormat() != "" {
		testTime := time.Now()
		if testTime.Format(cfg.TimestampFormat()) == cfg.TimestampFormat() {
			return fmt.Errorf("invalid timestamp_format: %s", cfg.TimestampFormat())
		}
	}

	return nil
}

func validateOTP(cfg OTPConfig) error {
	if cfg.ExpiresIn() <= 0 {
		return fmt.Errorf("otp expires_in must be positive")
	}

	if cfg.RetryBaseWaitTime() <= 0 {
		return fmt.Errorf("otp retry_base_wait_time must be positive")
	}

	if cfg.RetryMaxWaitTime() <= 0 {
		return fmt.Errorf("otp retry_max_wait_time must be positive")
	}

	if cfg.RetryBaseWaitTime() >= cfg.RetryMaxWaitTime() {
		return fmt.Errorf("retry_base_wait_time must be less than retry_max_wait_time")
	}

	// OTP should not expire too quickly (minimum 1 minute)
	if cfg.ExpiresIn() < time.Minute {
		return fmt.Errorf("otp expires_in must be at least 1 minute for security")
	}

	// OTP should not expire too slowly (maximum 30 minutes)
	if cfg.ExpiresIn() > 30*time.Minute {
		return fmt.Errorf("otp expires_in should not exceed 30 minutes for security")
	}

	return nil
}

func validateUpload(cfg UploadConfig) error {
	provider := cfg.Provider()
	if provider != "s3" && provider != "local" {
		return fmt.Errorf("upload provider must be 's3' or 'local'")
	}

	if provider == "local" {
		if cfg.LocalDir() == "" {
			return fmt.Errorf("local_dir is required when provider is 'local'")
		}

		// Check if directory exists or can be created
		if err := os.MkdirAll(cfg.LocalDir(), 0755); err != nil {
			return fmt.Errorf("cannot create local upload directory: %w", err)
		}
	}

	if provider == "s3" {
		if cfg.S3BucketName() == "" {
			return fmt.Errorf("s3_bucket_name is required when provider is 's3'")
		}
		if cfg.S3Region() == "" {
			return fmt.Errorf("s3_region is required when provider is 's3'")
		}
		if cfg.S3AccessKey() == "" {
			return fmt.Errorf("s3 access key id is required when provider is 's3'")
		}
		if cfg.S3SecretKey() == "" {
			return fmt.Errorf("s3 secret access key is required when provider is 's3'")
		}
		if cfg.S3PresignURLTTL() <= 0 {
			return fmt.Errorf("s3 presign_url_ttl must be positive")
		}
		if cfg.S3EndpointURL() != "" && !strings.HasPrefix(cfg.S3EndpointURL(), "http") {
			return fmt.Errorf("s3 endpoint_url must start with http:// or https://")
		}
	}

	return nil
}

func validateExternal(cfg ExternalConfig) error {
	if cfg.VietMapAPIKey() == "" {
		return fmt.Errorf("vietmap api key is required, please set VIETMAP_API_KEY env variable")
	}

	// TODO: Uncomment when Firebase is implemented
	// if cfg.FirebaseAccountKeyPath() == "" {
	// 	return fmt.Errorf("firebase__account_key_path is required")
	// }

	// // Check if service account file exists
	// if _, err := os.Stat(cfg.FirebaseAccountKeyPath()); err != nil {
	// 	if os.IsNotExist(err) {
	// 		return fmt.Errorf("firebase service account key file does not exist: %s", cfg.FirebaseAccountKeyPath())
	// 	}
	// 	return fmt.Errorf("cannot access firebase service account key file: %w", err)
	// }

	return nil
}

func validateRPC(cfg RPCConfig) error {
	if cfg.Host() == "" {
		return fmt.Errorf("rpc host is required")
	}
	if cfg.Port() <= 0 || cfg.Port() > 65535 {
		return fmt.Errorf("rpc port must be between 1 and 65535")
	}
	// Validate host format
	if cfg.Host() != "0.0.0.0" && cfg.Host() != "localhost" {
		if net.ParseIP(cfg.Host()) == nil {
			return fmt.Errorf("rpc host must be a valid IP address or 'localhost'")
		}
	}
	return nil
}
