// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package kvm_test

import (
	"runtime"
	"testing"

	gc "launchpad.net/gocheck"
)

func Test(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping kvm on windows")
	}
	gc.TestingT(t)
}
