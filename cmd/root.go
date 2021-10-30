package cmd

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/adamgoose/tsk/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "tsk",
	Short: "Runs an Ephemeral Tailscale node in Kubernetes",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.New()

		// Prepare Storage Directory
		storageDir, err := filepath.Abs(cfg.StorageDir)
		if err != nil {
			return fmt.Errorf("Unable to resolve storageDir: %v", err)
		}
		info, err := os.Stat(storageDir)
		if err != nil && os.IsNotExist(err) {
			err := os.MkdirAll(storageDir, 0766)
			if err != nil {
				return fmt.Errorf("Unable to create storageDir: %v", err)
			}
		} else if err != nil {
			return fmt.Errorf("Unable to find storageDir: %v", err)
		} else if !info.IsDir() {
			return fmt.Errorf("Configured storageDir is not a directory: %v", err)
		}
		viper.Set("storage_dir", fmt.Sprintf("file://%s/", storageDir))

		// Validate Tailscale Configuration
		if cfg.Tailscale.EphemeralKey == "" {
			return fmt.Errorf("Tailscale Ephemeral Key is required, consider setting --ephemeral-key|TSK_EPHEMERAL_KEY")
		}
		if cfg.Tailscale.APIKey == "" {
			return fmt.Errorf("Tailscale API Key is required, consider setting --api-key|TSK_API_KEY")
		}
		if cfg.Tailscale.Tailnet == "" {
			return fmt.Errorf("Tailscale Tailnet is required, consider setting -N|--tailnet|TSK_TAILNET")
		}
		if cfg.Tailscale.Hostname == "" {
			// TODO: use the kube context?
			return fmt.Errorf("Tailscale Hostname is required, consider setting -H|--hostname|TSK_HOSTNAME")
		}

		// Validate Kubernetes Configuration
		if cfg.Kubernetes.Namespace == "" {
			return fmt.Errorf("Kubernetes Namespace is required, consider setting -n|--namespace|TSK_NAMESPACE")
		}
		if cfg.Kubernetes.Username == "" {
			u, err := user.Current()
			if err != nil {
				return fmt.Errorf("Unable to determine username, consider setting -u|--username|TSK_USERNAME: %v", err)
			}

			// TODO: make sure its domain-safe
			viper.Set("kubernetes.username", u.Username)
		} else {
			// TODO: make sure its domain-safe
		}

		if _, _, err := net.ParseCIDR(cfg.Kubernetes.ServiceCIDR); err != nil {
			return fmt.Errorf("Unable to parse kubernetes service cidr: %v", err)
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	viper.SetEnvPrefix("tsk")
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().String("storage-dir", "~/.tsk", "Storage Dir")
	viper.BindPFlag("storage_dir", rootCmd.PersistentFlags().Lookup("storage-dir"))
	viper.SetDefault("storage_dir", filepath.Join(os.Getenv("HOME"), ".tsk"))

	rootCmd.PersistentFlags().String("ephemeral-key", "", "Ephemeral Key from Tailscale")
	viper.BindPFlag("tailscale.ephemeral_key", rootCmd.PersistentFlags().Lookup("ephemeral-key"))
	viper.RegisterAlias("tailscale.ephemeral_key", "ephemeral_key")

	rootCmd.PersistentFlags().String("api-key", "", "API Key from Tailscale")
	viper.BindPFlag("tailscale.api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.RegisterAlias("tailscale.api_key", "api_key")

	rootCmd.PersistentFlags().StringP("tailnet", "N", "", "Tailnet from Tailscale")
	viper.BindPFlag("tailscale.tailnet", rootCmd.PersistentFlags().Lookup("tailnet"))
	viper.RegisterAlias("tailscale.tailnet", "tailnet")

	rootCmd.PersistentFlags().StringP("hostname", "H", "", "Hostname to register with Tailscale")
	viper.BindPFlag("tailscale.hostname", rootCmd.PersistentFlags().Lookup("hostname"))
	viper.RegisterAlias("tailscale.hostname", "hostname")

	rootCmd.PersistentFlags().StringP("username", "u", "", "Username to label the deployment with")
	viper.BindPFlag("kubernetes.username", rootCmd.PersistentFlags().Lookup("username"))
	viper.RegisterAlias("kubernetes.username", "username")

	rootCmd.PersistentFlags().StringP("namespace", "n", "default", "Kubernetes Namespace to target")
	viper.BindPFlag("kubernetes.namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	viper.RegisterAlias("kubernetes.namespace", "namespace")
	viper.SetDefault("kubernetes.namespace", "default")

	rootCmd.PersistentFlags().String("cidr", "", "Kubernetes Serivce CIDR to expose")
	viper.BindPFlag("kubernetes.service_cidr", rootCmd.PersistentFlags().Lookup("cidr"))
	viper.RegisterAlias("kubernetes.service_cidr", "cidr")

}
