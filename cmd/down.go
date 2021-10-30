package cmd

import (
	"context"

	"github.com/adamgoose/tsk/config"
	"github.com/adamgoose/tsk/stack"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Brings the thing down",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg := config.New()
		s, err := stack.GetStack(ctx, cfg)
		if err != nil {
			return err
		}

		if _, err := s.Destroy(ctx); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
