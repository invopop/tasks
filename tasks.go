// Package tasks is used by Magefile scripts to tag and version applications.
package tasks

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Info provides an information object defining the current build
// environment, either using the current repo (preferred) or
// Google Cloud build environment variables.
type Info struct {
	Repo        bool
	Uncommitted bool
	Tag         string
	Branch      string
	Commit      string
	ShortCommit string
	Time        string
}

const (
	mainBranch      = "main"
	timeStampFormat = "v20060102T1504"
)

// Now generates an information object from the current environment.
func Now() Info {
	tn := time.Now().UTC().Format(time.RFC3339)
	i := Info{
		Time: tn,
		Repo: inRepo(),
	}
	if i.Repo {
		// Extract Git Details
		i.Uncommitted = uncommittedChanges()
		i.Tag, _ = getCurrentTag()
		i.Branch, _ = getCurrentBranch()
		i.Commit, _ = getCommit()
		i.ShortCommit, _ = getShortCommit()
	}
	return i
}

// Release generates a version number based on the current timestamps,
// tags the repo, and pushes.
func Release() error {
	if uncommittedChanges() {
		return mg.Fatal(1, "Uncommitted changes!")
	}
	// do we have an active tag?
	if _, err := getCurrentTag(); err != nil {
		v := generateVersion()
		err = sh.Run("git", "tag", v)
		if err != nil {
			return err
		}
	}
	return sh.Run("git", "push", "--tags")
}

// Version takes the info object and attempts to determine what a version
// string might look like.
func (i Info) Version() string {
	if i.Repo {
		// we have a repo, try to use it to extract version
		var str string
		if i.Tag != "" {
			str = i.Tag
		} else {
			str = fmt.Sprintf("%s-%s", i.Branch, i.ShortCommit)
			if i.Uncommitted {
				str = str + "-WIP"
			}
		}
		return str
	}
	// No repo, assume in a CI environment where we already have a VERSION
	return os.Getenv("VERSION")
}

// LDFlags generates the flags to send to the linker to add build meta data
func (i Info) LDFlags() string {
	return fmt.Sprintf(`-X 'main.BuildVersion=%v' -X 'main.BuildTime=%v'`, i.Version(), i.Time)
}

func inRepo() bool {
	err := sh.Run("git", "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func uncommittedChanges() bool {
	err := sh.Run("git", "diff-index", "--quiet", "HEAD", "--")
	return err != nil
}

func getCommit() (string, error) {
	txt, err := sh.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(txt), nil
}

func getShortCommit() (string, error) {
	txt, err := sh.Output("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(txt), nil
}

func getCurrentTag() (string, error) {
	txt, err := sh.Output("git", "describe", "--exact-match", "--tags")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(txt), nil
}

func getCurrentBranch() (string, error) {
	txt, err := sh.Output("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(txt), nil
}

func generateVersion() string {
	branch, err := getCurrentBranch()
	if err != nil {
		branch = mainBranch
	}
	tn := time.Now().UTC()
	v := tn.Format(timeStampFormat)
	if branch == mainBranch {
		return v
	}
	return branch + "-" + v
}
