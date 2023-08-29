package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var commitBasicAllowEmptyCmd = &cobra.Command{
	Use: "c",
	RunE: func(_cmd *cobra.Command, args []string) (err error) {

		var log *zap.Logger
		log, err = zap.NewDevelopment()
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				log.Error("commitBasicAllowEmpty", zap.Error(err))
			}
		}()

		gitargs := []string{"commit", "-v", "--allow-empty"}
		if flagCommitBasicUpdate || flagCommitBasicAmend { // --update or --amend flag
			gitargs = append(gitargs, "--amend")
		}
		cmd := exec.Command("git", gitargs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Start()
		if err != nil {
			return err
		}

		err = cmd.Wait()
		if err != nil {
			return err
		}

		return nil

	},
}

var (
	flagCommitBasicUpdate, flagCommitBasicAmend bool
)
