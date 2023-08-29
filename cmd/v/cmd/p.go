package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var pushCmd = &cobra.Command{
	Use: "p",
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
		head, _, err := c.Branch()
		if err != nil {
			return err
		}
		{
			push := exec.Command("git", "push", "-o", "ci.skip", c.Remote(), head)
			if c.IsDryRun() {
				fmt.Println("dry mode:", push)
				return nil
			}
			push.Stdin = os.Stdin
			push.Stdout = os.Stdout
			push.Stderr = os.Stderr

			if err := push.Run(); err != nil {
				return err
			}
		}
		return nil
	},
}
