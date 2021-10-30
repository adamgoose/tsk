package cmd

import (
	"context"

	"github.com/adamgoose/tsk/config"
	"github.com/adamgoose/tsk/stack"
	"github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Displays the stack state",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg := config.New()
		s, err := stack.GetStack(ctx, cfg)
		if err != nil {
			return err
		}

		summary, err := s.Export(ctx)
		if err != nil {
			return err
		}

		cmd.Printf("%s", summary.Deployment)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stackCmd)
}
