package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	flagMore    bool
	flagDryMode string
)

func init() {

	rootCmd.Flags().BoolVar(&flagMore, "more", false, "display more log")

	// --mode=(dry|run|"")
	rootCmd.PersistentFlags().StringVar(&flagDryMode, "mode", "", "--mode=dry or --mode=run overwrite dry-run state")

	rootCmd.AddCommand(cmdAdd)
	rootCmd.AddCommand(statusCmd)

	rootCmd.AddCommand(commitBasicAllowEmptyCmd)
	commitBasicAllowEmptyCmd.Flags().BoolVar(&flagCommitBasicAmend, "amend", false, "stick --amend to git commit")
	commitBasicAllowEmptyCmd.Flags().BoolVar(&flagCommitBasicUpdate, "update", false, "--amend alias")

	//XXX (engage.go.xxx) rootCmd.AddCommand(engageCmd)

	rootCmd.AddCommand(tagCmd)
	defaultMajor, defaultMinor, defaultPatch, defaultRc := 0, 1, 0, 1
	defaultName := "alpha"
	tagCmd.Flags().IntVar(&flagTagMajor, "major", defaultMajor, "set major revision (semantic version)")
	tagCmd.Flags().IntVar(&flagTagMinor, "minor", defaultMinor, "set minor revision (semantic version)")
	tagCmd.Flags().IntVar(&flagTagPatch, "patch", defaultPatch, "set patch revision (semantic version)")
	tagCmd.Flags().IntVar(&flagTagPreCount, "rc", defaultRc, "set patch revision (semantic version)")
	tagCmd.Flags().StringVar(&flagTagPreName, "rc-name", defaultName, "set patch revision (semantic version)")
	tagCmd.Flags().BoolVar(&flagTagNew, "new", false, "overwrite tagger counter") // antonym to flagTagIinc
	tagCmd.Flags().BoolVar(&flagTagInc, "inc", true, "increment tagger counter")

	tagCmd.Flags().BoolVar(&flagTagIncMajor, "imajor", false, "inc major")
	tagCmd.Flags().BoolVar(&flagTagIncMinor, "iminor", false, "inc minor")
	tagCmd.Flags().BoolVar(&flagTagIncPatch, "ipatch", false, "inc patch")
	tagCmd.Flags().BoolVar(&flagTagIncRevision, "irc", true, "inc revision (pre-release count)")

	rootCmd.AddCommand(switchCmd)
	switchCmd.AddCommand(drySwitchCmd)

	rootCmd.AddCommand(pushCmd)

	rootCmd.AddCommand(tagAsIncCmd)

	tagAsIncCmd.Flags().BoolVar(&flagTagIncMajor, "maj", false, "increment major semantic version")
	tagAsIncCmd.Flags().BoolVar(&flagTagIncMinor, "min", false, "increment minor semantic version")
	tagAsIncCmd.Flags().BoolVar(&flagTagIncPatch, "p", false, "increment patch semantic version")
	tagAsIncCmd.Flags().BoolVar(&flagTagIncRevision, "c", false, "increment revision semantic version")

	rootCmd.AddCommand(tagAsNewCmd)

	tagAsNewCmd.Flags().IntVar(&flagTagMajor, "maj", defaultMajor, "set new major in semantic version")
	tagAsNewCmd.Flags().IntVar(&flagTagMinor, "min", defaultMinor, "set new minor in semantic version")
	tagAsNewCmd.Flags().IntVar(&flagTagPatch, "p", defaultPatch, "set new patch in semantic version")
	tagAsNewCmd.Flags().IntVar(&flagTagPreCount, "c", defaultRc, "set new rc.inc in semantic version")
	tagAsNewCmd.Flags().StringVar(&flagTagPreName, "rc", defaultName, "set new rc.name in semantic version")
	tagAsNewCmd.Flags().StringVarP(&flagTagSemanticVersion, "version", "v", "", "set with specified semver")

}

type cli interface {
	// Branch returns HEAD brach if any, all branches and maybe an error.
	// Branch is supposed to eval `git branch` and process the output.
	Branch() (string, []string, error)
	Remote() string
	Draft() bool
	IsDryRun() bool
	DryMode(mode bool) error
}

func newCli() (cli, error) {
	return &_cli{}, nil
}

func (_ *_cli) Draft() bool {
	return viper.GetBool(string(viperContextVarDraft))
}

func (_ *_cli) Remote() string {
	return viper.GetString(string(viperContextVarRemote))
}

func (_ *_cli) IsDryRun() bool {
	switch flagDryMode {
	case "dry":
		return true
	case "run":
		return false
	default:
		return viper.GetBool(string(viperContextVarDryMode))
	}
}

const contextPath = ".vc.yaml"

func (_ *_cli) DryMode(mode bool) error {
	viper.Set(string(viperContextVarDryMode), mode)
	return viper.WriteConfigAs(contextPath)
}

type _cli struct{}

func (c *_cli) Branch() (head string, all []string, err error) {
	err = func() error {

		cmd := exec.Command("git", "branch")
		r, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}

		if err = cmd.Start(); err != nil {
			return err
		}

		s := bufio.NewScanner(r)
		for s.Scan() {
			if _head, found := strings.CutPrefix(s.Text(), "* "); found {
				head = _head
			}
			all = append(all, s.Text())
		}

		if err = cmd.Wait(); err != nil {
			return err
		}

		return nil

	}()
	return
}

type viperContextVar string

const (
	viperContextVarRemote  = viperContextVar("remote")
	viperContextVarDraft   = viperContextVar("draft")
	viperContextVarDryMode = viperContextVar("dry_mode")

	// see also viperTags (t.go)
)

func viperInit() (cli, error) {
	viper.SetConfigName(".vc")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetDefault(string(viperContextVarDryMode), true)
	viper.SetDefault(string(viperContextVarRemote), "origin")
	viper.SetDefault(string(viperContextVarDraft), true)
	viper.SetDefault(viperTagFlagMajorVersion, 0)
	viper.SetDefault(viperTagFlagMinorVersion, 0)
	viper.SetDefault(viperTagFlagPatchVersion, 0)
	viper.SetDefault(viperTagFlagReleaseContentName, "alpha")
	viper.SetDefault(viperTagFlagReleaseContentRevision, 1)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	return newCli()
}

func clear() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func newRootContext(log *zap.Logger) (rc, error) {
	cli, err := viperInit()
	if err != nil {
		return nil, err
	}
	return &_rc{
		cli: cli,
		log: log,
	}, nil
}

type rc interface {
	RootContext() cli
}

type _rc struct {
	cli cli
	log *zap.Logger
}

func (c _rc) RootContext() cli { return c.cli }

var rootCmd = &cobra.Command{
	Use: "vc",
	RunE: func(cmd *cobra.Command, args []string) error {

		clear()

		fmt.Println()

		{
			logArgs := []string{"log"}
			if !flagMore {
				logArgs = append(logArgs, "-1")
			}
			logCmd := exec.Command("git", logArgs...)
			logCmd.Stdout = os.Stdout
			if err := logCmd.Run(); err != nil {
				return err
			}
		}

		fmt.Println()

		{
			cmd := exec.Command("git", "status", "-s")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		}

		return nil

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
