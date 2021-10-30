package cmd

import (
	"context"

	"github.com/adamgoose/tsk/config"
	"github.com/adamgoose/tsk/stack"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Brings the thing up",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg := config.New()
		s, err := stack.GetStack(ctx, cfg)
		if err != nil {
			return err
		}

		o, err := s.Up(ctx)
		if err != nil {
			return err
		}

		if !o.Outputs["dnsConfigured"].Value.(bool) {
			cmd.Println("Your tailscale node has been deployed, but DNS isn't set up yet. Try running 'tsk up' again.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
