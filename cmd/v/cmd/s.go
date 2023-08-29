package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use: "s",
	RunE: func(cmd *cobra.Command, args []string) error {

		sc := exec.Command("git", "status", "-u")
		sc.Stdout = os.Stdout
		return sc.Run()

	},
}
