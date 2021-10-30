package cmd

import (
	"context"

	"github.com/adamgoose/tsk/config"
	"github.com/adamgoose/tsk/stack"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Brings the thing up (but not really)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg := config.New()
		s, err := stack.GetStack(ctx, cfg)
		if err != nil {
			return err
		}

		opts := []optpreview.Option{}
		if planVerbose {
			opts = append(opts, optpreview.Diff())
		}

		p, err := s.Preview(ctx, opts...)
		if err != nil {
			return err
		}

		cmd.Println(p.StdOut)

		return nil
	},
}

var planVerbose bool = false

func init() {
	rootCmd.AddCommand(planCmd)

	planCmd.Flags().BoolVarP(&planVerbose, "verbose", "v", false, "Display verbose diff")
}
