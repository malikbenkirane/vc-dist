package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var tagCmd = &cobra.Command{
	Use: "t",
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInc(false, false) // ! alias
	},
}

var tagAsIncCmd = &cobra.Command{
	Use: "inc",
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInc(true, true) // alias inc
	},
}

var tagAsNewCmd = &cobra.Command{
	Use: "new",
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInc(true, false) // alias new
	},
}

type tag interface {
	Inc(major, minor, patch, revision bool)
	ReleaseContent(name string)
	MajorVersion() int
	MinorVersion() int
	PatchNumber() int
	PreReleaseCount() int
	Revision() int
	PreReleaseName() string
	WriteToViperConfig() error
	fmt.Stringer
}

func (t *_tag) ReleaseContent(name string) {
	t.version = newVersion(t.major, t.minor, t.patch, t.rc, name)
}

func (t *_tag) Inc(major, minor, patch, revision bool) {
	if major {
		t.major++
	}
	if minor {
		t.minor++
	}
	if patch {
		t.patch++
	}
	if revision {
		t.rc++
	}
	t.version = newVersion(t.major, t.minor, t.patch, t.rc, t.rcName)
}

var (
	flagTagMajor    int    // --major
	flagTagMinor    int    // --minor
	flagTagPatch    int    // --patch
	flagTagPreCount int    // --rc
	flagTagPreName  string // --rc-name
	flagTagInc      bool   // --inc
	flagTagNew      bool   // --new

	flagTagIncMajor    bool // --imajor=false
	flagTagIncMinor    bool // --iminor=false
	flagTagIncPatch    bool // --ipatch=false
	flagTagIncRevision bool // --irc=true

	flagTagSemanticVersion string // --v=v0.1.0-alpha.1
)

const (
	viperTagFlagReleaseContentRevision = "current.tag.revision"
	viperTagFlagMajorVersion           = "current.tag.version.major"
	viperTagFlagMinorVersion           = "current.tag.version.minor"
	viperTagFlagPatchVersion           = "current.tag.version.patch"
	viperTagFlagReleaseContentName     = "current.tag.release.name"
)

func viperTag() tag {
	return newTag(
		viper.GetInt(viperTagFlagReleaseContentRevision),
		viper.GetInt(viperTagFlagMajorVersion),
		viper.GetInt(viperTagFlagMinorVersion),
		viper.GetInt(viperTagFlagPatchVersion),
		viper.GetString(viperTagFlagReleaseContentName),
	)
}

func newVersion(major, minor, patch, rc int, name string) string {
	return fmt.Sprintf("v%d.%d.%d-%s.%d", major, minor, patch, name, rc)
}

func newTag(rc, major, minor, patch int, name string) tag {
	name = strings.ReplaceAll(slug.Make(name), "-", "_")
	return &_tag{
		version: newVersion(major, minor, patch, rc, name),
		rc:      rc,
		major:   major,
		minor:   minor,
		patch:   patch,
		rcName:  name,
	}
}

type _tag struct {
	version, rcName         string
	rc, major, minor, patch int
}

func (t *_tag) WriteToViperConfig() error {
	viper.Set(viperTagFlagReleaseContentRevision, t.rc)
	viper.Set(viperTagFlagMajorVersion, t.major)
	viper.Set(viperTagFlagMinorVersion, t.minor)
	viper.Set(viperTagFlagPatchVersion, t.patch)
	viper.Set(viperTagFlagReleaseContentName, t.rcName)
	return viper.WriteConfigAs(contextPath) //FIXME(malikbenkirane)
}

func (t *_tag) MajorVersion() int      { return t.major }
func (t *_tag) MinorVersion() int      { return t.minor }
func (t *_tag) PatchNumber() int       { return t.patch }
func (t *_tag) PreReleaseCount() int   { return t.rc }
func (t *_tag) Revision() int          { return t.rc }
func (t *_tag) PreReleaseName() string { return t.rcName }
func (t *_tag) String() string         { return t.version }

func execInc(alias, inc bool) error {

	var c cli

	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}

	rc, err := newRootContext(log)
	if err != nil {
		return err
	}

	c = rc.RootContext()

	if !alias {

		if !flagTagNew || flagTagInc {
			inc = true
		}
		if flagTagNew {
			inc = false
		}

	}

	if inc {
		t := viperTag()
		t.Inc(flagTagIncMajor, flagTagIncMinor, flagTagIncPatch, flagTagIncRevision)
		fmt.Println("inc -->", t.String())
		head, branches, err := c.Branch()
		if err != nil {
			return err
		}
		fmt.Println("head:", head)
		fmt.Println("branches:")
		for i, b := range branches {
			fmt.Println(i, b)
		}
		return apply(c, t)
	}

	var done bool
	if err = func() error {
		var errSemver = errors.New("semver")
		err = func() error {
			v, err := semver.Make(flagTagSemanticVersion)
			if err != nil {
				log.Debug("semver: make", zap.Error(err))
				return errSemver
			}
			var t tag
			{
				maj, min, fix, pre := int(v.Major), int(v.Minor), int(v.Patch), v.Pre
				log.Debug("semver",
					zap.Int("major", maj),
					zap.Int("minor", min),
					zap.Int("patch", fix),
					zap.Any("pre/prVersions", pre),
				)
				var (
					rc   int
					name string
				)
				{
					var preName string
					var preNumber int
					if len(pre) == 2 {
						preName = pre[0].VersionStr
						preNumber = int(pre[1].VersionNum)
					}
					if len(preName) > 0 {
						name = preName
						rc = preNumber
					}
				}
				t = newTag(rc, maj, min, fix, name)
			}
			return apply(c, t)
		}()
		if errors.Is(err, errSemver) {
			return nil
		}
		if err != nil {
			return err
		}
		done = true
		return nil
	}(); err != nil || done {
		return err
	}

	t := newTag(flagTagPreCount, flagTagMajor, flagTagMinor, flagTagPatch, flagTagPreName)
	fmt.Println("new", t)
	return apply(c, t)

}

func apply(c cli, t tag) error {
	if c.IsDryRun() {
		fmt.Print(`
dry mode
-

change dry behavior with subcommand

    vc switch dry

-
`)
	}
	{
		tag := exec.Command("git", "tag", t.String())
		if c.IsDryRun() {
			fmt.Println(tag)
		} else {
			tag.Stdout = os.Stdout
			tag.Stdin = os.Stdin
			tag.Stderr = os.Stderr
			if err := tag.Run(); err != nil {
				return fmt.Errorf("run tag: %w", err)
			}
		}
	}
	{
		push := exec.Command("git", "push", c.Remote(), t.String())
		if c.IsDryRun() {
			fmt.Println(push)
		} else {
			push.Stdout = os.Stdout
			push.Stdin = os.Stdin
			push.Stderr = os.Stderr
			if err := push.Run(); err != nil {
				return fmt.Errorf("run push: %w", err)
			}
		}
	}

	return t.WriteToViperConfig()
}
