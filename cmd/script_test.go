package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	commands := map[string]func() int{
		"gori": Main,
		"git": func() int {
			cmd := exec.Command("/usr/bin/git", os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					return exitError.ExitCode()
				}
				return 1
			}
			return 0
		},
	}
	os.Exit(testscript.RunMain(m, commands))
}

func TestGori(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "test",
	})
}
