// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ssh_test

import (
	"testing"
	"runtime"

	gc "launchpad.net/gocheck"
)

func TestPackage(t *testing.T) {
	// skipped all these tests on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping all ssh-related tests on Windows")
	}
	gc.TestingT(t)
}
