package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"go.uber.org/zap"
)

type instance interface {
	skimPipe(from *exec.Cmd)
	run() (io.Reader, error)
}

func errw(err, kind error) error { return fmt.Errorf("%w: %w", kind, err) }

func (i *_instance) run() (io.Reader, error) {
	if i._skimPipe == nil {
		return nil, errSkimPipeToNull
	}
	from := i._skimPipe
	if err := func() (err error) {

		defer func() {
			if err != nil {
				i.log.Error("run skim pipe", zap.Error(err))
			}
		}()

		i.log.Debug("require sk in path")
		sk := exec.Command("sk")

		var inp io.WriteCloser
		inp, err = sk.StdinPipe()
		if err != nil {
			return errw(err, errSkimPipeSkStdinPipe)
		}

		sk.Stdout = i.out

		if err = sk.Start(); err != nil {
			return errw(err, errSkimPipeSkStart)
		}

		from.Stdout = inp

		if err = from.Run(); err != nil {
			return errw(err, errSkimPipeFromRun)
		}

		if err = inp.Close(); err != nil {
			return errw(err, errSkimPipeCloseInp)
		}

		if err = sk.Wait(); err != nil {
			return errw(err, errSkimPipeFromWait)
		}

		return nil
	}(); err != nil {
		return nil, err
	}
	return i.out, nil
}

func (i *_instance) skimPipe(from *exec.Cmd) {
	i._skimPipe = from
}

func newInstance(log *zap.Logger, from *exec.Cmd) instance {
	return &_instance{
		log:       log,
		out:       &bytes.Buffer{},
		_skimPipe: from,
	}
}

type _instance struct {
	log       *zap.Logger
	_skimPipe *exec.Cmd
	out       *bytes.Buffer
}

//go:generate stringer -type=errSkimPipeCode -output=errors_skim_pipe.gen.go
type errSkimPipeCode int

const (
	errSkimPipeToNullC = errSkimPipeCode(iota)
	errSkimPipeSkStdinPipeC
	errSkimPipeSkStartC
	errSkimPipeFromRunC
	errSkimPipeCloseInpC
	errSkimPipeFromWaitC
)

var (
	errSkimPipeToNull      = errors.New(errSkimPipeToNullC.String())
	errSkimPipeSkStdinPipe = errors.New(errSkimPipeSkStdinPipeC.String())
	errSkimPipeSkStart     = errors.New(errSkimPipeSkStartC.String())
	errSkimPipeFromRun     = errors.New(errSkimPipeFromRunC.String())
	errSkimPipeCloseInp    = errors.New(errSkimPipeCloseInpC.String())
	errSkimPipeFromWait    = errors.New(errSkimPipeFromWaitC.String())
)

var flagAddAuto bool

func init() {
	cmdAdd.Flags().BoolVarP(&flagAddAuto, "auto", "a", false, "stage all tracked files")
}

var cmdAdd = &cobra.Command{
	Use: "a",
	RunE: func(cmd *cobra.Command, args []string) error {

		log, err := zap.NewDevelopment()
		if err != nil {
			return err
		}

		if flagAddAuto {
			cmd := exec.Command("git", "commit", "-av")
			cmd.Stdout = os.Stdout
			cmd.Stdin = os.Stdin
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}

		statcmd := exec.Command("git", "status", "-s", "-u")

		out, err := newInstance(log, statcmd).run()
		if err != nil {
			return err
		}

		var state, filename string
		{
			s := bufio.NewScanner(out)
			s.Split(bufio.ScanWords)
			for s.Scan() {
				if state == "" {
					state = s.Text() // state is first arg
				}
				filename = s.Text() // keep last arg
			}
			log.Debug("", zap.String("state", state), zap.String("filename", filename))
		}

		_, err = exec.Command("git", "add", filename).Output()
		if err != nil {
			return err
		}

		return nil

	},
}
