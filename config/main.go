package config

import (
	"github.com/spf13/viper"
)

func New() (config Config) {
	viper.Unmarshal(&config)
	return
}

type Config struct {
	StorageDir string           `mapstructure:"storage_dir"`
	Tailscale  TailscaleConfig  `mapstructure:"tailscale"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
}

type TailscaleConfig struct {
	EphemeralKey string `mapstructure:"ephemeral_key"`
	APIKey       string `mapstructure:"api_key"`
	Tailnet      string `mapstructure:"tailnet"`
	Hostname     string `mapstructure:"hostname"`
}

type KubernetesConfig struct {
	Username    string `mapstructure:"username"`
	Namespace   string `mapstructure:"namespace"`
	ServiceCIDR string `mapstructure:"service_cidr"`
}
