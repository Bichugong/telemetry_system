package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	InfluxDB InfluxDBConfig `mapstructure:"influxdb"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type ServerConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	HealthEndpoint string `mapstructure:"health_endpoint"`
}

type MQTTConfig struct {
	Broker         string   `mapstructure:"broker"`
	ClientID       string   `mapstructure:"client_id"`
	Username       string   `mapstructure:"username"`
	Password       string   `mapstructure:"password"`
	Topics         []string `mapstructure:"topics"`
	QoS            int      `mapstructure:"qos"`
	ReconnectRetry int      `mapstructure:"reconnect_retry"`
}

type InfluxDBConfig struct {
	URL           string `mapstructure:"url"`
	Token         string `mapstructure:"token"`
	Org           string `mapstructure:"org"`
	Bucket        string `mapstructure:"bucket"`
	RetentionDays int    `mapstructure:"retention_days"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}