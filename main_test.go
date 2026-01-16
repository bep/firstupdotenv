package main

import (
	"strings"
	"testing"
)

func TestFindEnvInCurrentDir(t *testing.T) {
	t.Parallel()

	env, err := createEnvSourceFromCurrentDir()
	if err != nil {
		t.Fatal(err)
	}

	check := func(s ...string) {
		for _, want := range s {
			if !strings.Contains(env, want) {
				t.Errorf("env %q does not contain %q", env, want)
			}
		}
	}

	check("export FIRSTUPDOTENV_CURRENT_SET_ENV=FOO,BAR")
	check(`export FOO="value1"`)
	check(`export BAR="value2"`)
}
