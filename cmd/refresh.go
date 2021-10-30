package cmd

import (
	"context"

	"github.com/adamgoose/tsk/config"
	"github.com/adamgoose/tsk/stack"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the state",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg := config.New()
		s, err := stack.GetStack(ctx, cfg)
		if err != nil {
			return err
		}

		if _, err := s.Refresh(ctx); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}
