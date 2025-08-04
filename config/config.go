package config

import (
	"fmt"
	"time"
)

const (
	LocalEnv       = "local"
	DevelopmentEnv = "dev"
	ProductionEnv  = "prod"
)

type Config interface {
	App() AppConfig
	Server() ServerConfig
	Database() DatabaseConfig
	Redis() RedisConfig
	Cache() CacheConfig
	Logger() LoggerConfig
	OTP() OTPConfig
	Upload() UploadConfig
	External() ExternalConfig
	RPC() RPCConfig
}

type AppConfig interface {
	Name() string
	Version() string
	Environment() string
	IsProduction() bool
	AccessTokenExpiresIn() time.Duration
	AccessTokenSecret() string
	RefreshTokenExpiresIn() time.Duration
	RefreshTokenSecret() string
	TokenIssuer() string
	SessionLimitPerUser() int
	UserSessionLimitEnabled() bool
	APIKey() string
	SystemAdminDefaultPhone() string
	SystemAdminDefaultEmail() string
	SystemAdminDefaultPassword() string
}

type ServerConfig interface {
	Host() string
	Domain() string
	Port() int
	ReadTimeout() time.Duration
	WriteTimeout() time.Duration
	IdleTimeout() time.Duration
	MaxHeaderBytes() int
	AllowedOrigins() []string
}

type DatabaseConfig interface {
	Host() string
	Port() string
	User() string
	Password() string
	Name() string
	SSLMode() string
	MaxOpenConns() int
	MaxIdleConns() int
	ConnMaxLifetime() time.Duration
	LogLevel() string
	EnableLog() bool
}

type RedisConfig interface {
	Host() string
	Port() int
	Address() string
	Password() string
	DB() int
	Prefix() string
}

type CacheConfig interface {
	Provider() string
	DefaultTTL() time.Duration
}

type LoggerConfig interface {
	LogFilePath() string
	LogFileName() string
	TimestampFormat() string
	LogLevel() string
	FileExtension() string
	MaxFileSizeMB() int
	MaxFileAgeDays() int
	MaxBackupFiles() int
	IsCompressEnabled() bool
}

type OTPConfig interface {
	ExpiresIn() time.Duration
	RetryBaseWaitTime() time.Duration
	RetryMaxWaitTime() time.Duration
}

type UploadConfig interface {
	Provider() string
	LocalDir() string
	S3EndpointURL() string
	S3BucketName() string
	S3PathPrefix() string
	S3Region() string
	S3PresignURLTTL() time.Duration
	S3AccessKey() string
	S3SecretKey() string
}

type ExternalConfig interface {
	VietMapAPIKey() string
	FirebaseAccountKeyPath() string
}

type RPCConfig interface {
	Host() string
	Port() int
}

// config holds the actual configuration implementation
type config struct {
	AppCfg      appConfig      `yaml:"app"`
	ServerCfg   serverConfig   `yaml:"server"`
	DatabaseCfg databaseConfig `yaml:"database"`
	RedisCfg    redisConfig    `yaml:"redis"`
	CacheCfg    cacheConfig    `yaml:"cache"`
	LoggerCfg   loggerConfig   `yaml:"logger"`
	OTPCfg      otpConfig      `yaml:"otp"`
	UploadCfg   uploadConfig   `yaml:"upload"`
	ExternalCfg externalConfig `yaml:"external"`
	RPCCfg      rpcConfig      `yaml:"rpc"`
}

func (c *config) App() AppConfig {
	return &c.AppCfg
}

func (c *config) Server() ServerConfig {
	return &c.ServerCfg
}

func (c *config) Database() DatabaseConfig {
	return &c.DatabaseCfg
}

func (c *config) Redis() RedisConfig {
	return &c.RedisCfg
}

func (c *config) Cache() CacheConfig {
	return &c.CacheCfg
}

func (c *config) Logger() LoggerConfig {
	return &c.LoggerCfg
}

func (c *config) OTP() OTPConfig {
	return &c.OTPCfg
}

func (c *config) Upload() UploadConfig {
	return &c.UploadCfg
}

func (c *config) External() ExternalConfig {
	return &c.ExternalCfg
}

func (c *config) RPC() RPCConfig {
	return &c.RPCCfg
}

type appConfig struct {
	NameStr        string `yaml:"name"`
	VersionStr     string `yaml:"version"`
	EnvironmentStr string `env:"ENV" env-default:"local"`

	TokenIssuerStr string `yaml:"token_issuer"`

	AccessTokenExpiresInDur time.Duration `yaml:"access_token_expires_in"`
	AccessTokenSecretStr    string        `env:"ACCESS_TOKEN_SECRET"`

	RefreshTokenExpiresInDur time.Duration `yaml:"refresh_token_expires_in"`
	RefreshTokenSecretStr    string        `env:"REFRESH_TOKEN_SECRET"`

	SessionLimitPerUserInt      int  `yaml:"session_limit_per_user"`
	UserSessionLimitEnabledBool bool `yaml:"user_session_limit_enabled"`

	APIKeyStr string `env:"API_KEY" env-default:""`

	SysAdminDefaultPhoneStr    string `env:"SYSTEM_ADMIN_DEFAULT_PHONE" env-default:""`
	SysAdminDefaultEmailStr    string `env:"SYSTEM_ADMIN_DEFAULT_EMAIL" env-default:""`
	SysAdminDefaultPasswordStr string `env:"SYSTEM_ADMIN_DEFAULT_PASSWORD" env-default:""`
}

func (c *appConfig) Name() string {
	return c.NameStr
}

func (c *appConfig) Version() string {
	return c.VersionStr
}

func (c *appConfig) Environment() string {
	return c.EnvironmentStr
}

func (c *appConfig) IsProduction() bool {
	return c.EnvironmentStr == ProductionEnv
}

func (c *appConfig) AccessTokenExpiresIn() time.Duration {
	return c.AccessTokenExpiresInDur
}

func (c *appConfig) AccessTokenSecret() string {
	return c.AccessTokenSecretStr
}

func (c *appConfig) RefreshTokenExpiresIn() time.Duration {
	return c.RefreshTokenExpiresInDur
}

func (c *appConfig) RefreshTokenSecret() string {
	return c.RefreshTokenSecretStr
}

func (c *appConfig) TokenIssuer() string {
	return c.TokenIssuerStr
}

func (c *appConfig) SessionLimitPerUser() int {
	return c.SessionLimitPerUserInt
}

func (c *appConfig) UserSessionLimitEnabled() bool {
	return c.UserSessionLimitEnabledBool
}

func (c *appConfig) APIKey() string {
	return c.APIKeyStr
}

func (c *appConfig) SystemAdminDefaultPhone() string {
	return c.SysAdminDefaultPhoneStr
}

func (c *appConfig) SystemAdminDefaultEmail() string {
	return c.SysAdminDefaultEmailStr
}

func (c *appConfig) SystemAdminDefaultPassword() string {
	return c.SysAdminDefaultPasswordStr
}

type serverConfig struct {
	HostStr           string   `yaml:"host"`
	DomainStr         string   `yaml:"domain"`
	PortInt           int      `yaml:"port"`
	ReadTimeoutStr    string   `yaml:"read_timeout"`
	WriteTimeoutStr   string   `yaml:"write_timeout"`
	IdleTimeoutStr    string   `yaml:"idle_timeout" env-default:"120s"`
	MaxHeaderBytesInt int      `yaml:"max_header_bytes" env-default:"1048576"` // 1MB
	AllowedOriginsArr []string `yaml:"allowed_origins"`
}

func (s *serverConfig) Host() string {
	return s.HostStr
}

func (s *serverConfig) Domain() string {
	return s.DomainStr
}

func (s *serverConfig) Port() int {
	return s.PortInt
}

func (s *serverConfig) ReadTimeout() time.Duration {
	duration, _ := time.ParseDuration(s.ReadTimeoutStr)
	return duration
}

func (s *serverConfig) WriteTimeout() time.Duration {
	duration, _ := time.ParseDuration(s.WriteTimeoutStr)
	return duration
}

func (s *serverConfig) IdleTimeout() time.Duration {
	duration, _ := time.ParseDuration(s.IdleTimeoutStr)
	return duration
}

func (s *serverConfig) AllowedOrigins() []string {
	return s.AllowedOriginsArr
}

func (s *serverConfig) MaxHeaderBytes() int {
	return s.MaxHeaderBytesInt
}

type databaseConfig struct {
	HostStr            string `env:"POSTGRES_HOST" env-default:"localhost"`
	PortStr            string `env:"POSTGRES_PORT" env-default:"5432"`
	UserStr            string `env:"POSTGRES_USER" env-default:"postgres"`
	PasswordStr        string `env:"POSTGRES_PASSWORD" env-default:"postgres"`
	NameStr            string `env:"POSTGRES_DBNAME" env-default:"postgres"`
	SSLModeStr         string `env:"POSTGRES_SSL_MODE" env-default:"disable"`
	MaxOpenConnsInt    int    `yaml:"max_open_conns" env-default:"25"`
	MaxIdleConnsInt    int    `yaml:"max_idle_conns" env-default:"10"`
	ConnMaxLifetimeStr string `yaml:"conn_max_lifetime" env-default:"5m"`
	EnableLoggingBool  bool   `yaml:"enable_logging" env-default:"false"`
	LogLevelStr        string `yaml:"log_level" env-default:"warn"`
}

func (d *databaseConfig) Host() string {
	return d.HostStr
}

func (d *databaseConfig) Port() string {
	return d.PortStr
}

func (d *databaseConfig) User() string {
	return d.UserStr
}

func (d *databaseConfig) Password() string {
	return d.PasswordStr
}

func (d *databaseConfig) Name() string {
	return d.NameStr
}

func (d *databaseConfig) SSLMode() string {
	return d.SSLModeStr
}

func (d *databaseConfig) MaxOpenConns() int {
	return d.MaxOpenConnsInt
}

func (d *databaseConfig) MaxIdleConns() int {
	return d.MaxIdleConnsInt
}

func (d *databaseConfig) ConnMaxLifetime() time.Duration {
	duration, _ := time.ParseDuration(d.ConnMaxLifetimeStr)
	return duration
}

func (d *databaseConfig) EnableLog() bool {
	return d.EnableLoggingBool
}

func (d *databaseConfig) LogLevel() string {
	return d.LogLevelStr
}

type cacheConfig struct {
	ProviderStr   string `yaml:"provider"`
	DefaultTTLStr string `yaml:"default_ttl"`
}

func (c *cacheConfig) Provider() string {
	return c.ProviderStr
}

func (c *cacheConfig) DefaultTTL() time.Duration {
	duration, _ := time.ParseDuration(c.DefaultTTLStr)
	return duration
}

type externalConfig struct {
	VietMapAPIKeyStr          string `env:"VIETMAP_API_KEY" env-default:""`
	FirebaseAccountKeyPathStr string `yaml:"firebase_account_key_path"`
}

func (e *externalConfig) FirebaseAccountKeyPath() string {
	return e.FirebaseAccountKeyPathStr
}

func (e *externalConfig) VietMapAPIKey() string {
	return e.VietMapAPIKeyStr
}

type loggerConfig struct {
	LogFilePathStr     string `yaml:"log_file_path"`
	LogFileNameStr     string `yaml:"log_file_name"`
	TimestampFormatStr string `yaml:"timestamp_format"`
	LogLevelStr        string `yaml:"log_level"`
	FileExtensionStr   string `yaml:"file_extension"`
	MaxFileSizeMBInt   int    `yaml:"max_file_size_mb"`
	MaxFileAgeDaysInt  int    `yaml:"max_file_age_days"`
	MaxBackupFilesInt  int    `yaml:"max_backup_files"`
	EnableCompressed   bool   `yaml:"enable_compressed"`
}

func (l *loggerConfig) LogFilePath() string {
	return l.LogFilePathStr
}

func (l *loggerConfig) LogFileName() string {
	return l.LogFileNameStr
}

func (l *loggerConfig) TimestampFormat() string {
	return l.TimestampFormatStr
}

func (l *loggerConfig) LogLevel() string {
	return l.LogLevelStr
}

func (l *loggerConfig) FileExtension() string {
	return l.FileExtensionStr
}

func (l *loggerConfig) MaxFileSizeMB() int {
	return l.MaxFileSizeMBInt
}

func (l *loggerConfig) MaxFileAgeDays() int {
	return l.MaxFileAgeDaysInt
}

func (l *loggerConfig) MaxBackupFiles() int {
	return l.MaxBackupFilesInt
}

func (l *loggerConfig) IsCompressEnabled() bool {
	return l.EnableCompressed
}

type otpConfig struct {
	ExpiresInStr         string `yaml:"expires_in"`
	RetryBaseWaitTimeStr string `yaml:"retry_base_wait_time"`
	RetryMaxWaitTimeStr  string `yaml:"retry_max_wait_time"`
}

func (o *otpConfig) ExpiresIn() time.Duration {
	duration, _ := time.ParseDuration(o.ExpiresInStr)
	return duration
}

func (o *otpConfig) RetryBaseWaitTime() time.Duration {
	duration, _ := time.ParseDuration(o.RetryBaseWaitTimeStr)
	return duration
}

func (o *otpConfig) RetryMaxWaitTime() time.Duration {
	duration, _ := time.ParseDuration(o.RetryMaxWaitTimeStr)
	return duration
}

type uploadConfig struct {
	ProviderStr        string `yaml:"provider"`
	LocalDirStr        string `yaml:"local_dir"`
	S3EndpointURLStr   string `yaml:"s3_endpoint_url"`
	S3BucketNameStr    string `yaml:"s3_bucket_name"`
	S3PathPrefixStr    string `yaml:"s3_path_prefix"`
	S3RegionStr        string `yaml:"s3_region"`
	S3PresignURLTTLStr string `yaml:"s3_presign_url_ttl"`
	S3AccessKeyStr     string `env:"UPLOAD_S3_ACCESS_KEY" env-default:""`
	S3SecretKeyStr     string `env:"UPLOAD_S3_SECRET_KEY" env-default:""`
}

func (c *uploadConfig) Provider() string {
	return c.ProviderStr
}

func (c *uploadConfig) LocalDir() string {
	return c.LocalDirStr
}

func (c *uploadConfig) S3EndpointURL() string {
	return c.S3EndpointURLStr
}

func (c *uploadConfig) S3BucketName() string {
	return c.S3BucketNameStr
}

func (c *uploadConfig) S3PathPrefix() string {
	return c.S3PathPrefixStr
}

func (c *uploadConfig) S3Region() string {
	return c.S3RegionStr
}

func (c *uploadConfig) S3PresignURLTTL() time.Duration {
	duration, _ := time.ParseDuration(c.S3PresignURLTTLStr)
	return duration
}

func (c *uploadConfig) S3AccessKey() string {
	return c.S3AccessKeyStr
}

func (c *uploadConfig) S3SecretKey() string {
	return c.S3SecretKeyStr
}

type redisConfig struct {
	HostStr     string `env:"REDIS_HOST" env-default:"localhost"`
	PortInt     int    `env:"REDIS_PORT" env-default:"6379"`
	PasswordStr string `env:"REDIS_PASSWORD"`
	DBInt       int    `env:"REDIS_DB" env-default:"0"`
	PrefixStr   string `yaml:"prefix"`
}

func (r *redisConfig) Host() string {
	return r.HostStr
}

func (r *redisConfig) Port() int {
	return r.PortInt
}

func (r *redisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host(), r.Port())
}

func (r *redisConfig) Password() string {
	return r.PasswordStr
}

func (r *redisConfig) DB() int {
	return r.DBInt
}

func (r *redisConfig) Prefix() string {
	return r.PrefixStr
}

type rpcConfig struct {
	HostStr string `yaml:"host"`
	PortInt int    `yaml:"port"`
}

func (r *rpcConfig) Host() string {
	return r.HostStr
}

func (r *rpcConfig) Port() int {
	return r.PortInt
}
