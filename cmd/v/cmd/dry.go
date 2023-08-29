package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var switchCmd = &cobra.Command{
	Use: "switch",
}

var drySwitchCmd = &cobra.Command{
	Use: "dry",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := zap.NewDevelopment()
		if err != nil {
			return err
		}
		rc, err := newRootContext(log)
		if err != nil {
			return err
		}
		c := rc.RootContext()
		if c.IsDryRun() {
			fmt.Println("now you need to be careful 🛃 --dry-run=false")
		} else {
			fmt.Println("your are safe 🍃")
		}
		fmt.Println("dry", !c.IsDryRun())
		return c.DryMode(!c.IsDryRun())
	},
}
