// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ssh_test

import (
	"testing"

	"github.com/juju/juju/version"

	gc "launchpad.net/gocheck"
)

func TestPackage(t *testing.T) {
	os, _ := version.GetOSFromSeries(version.Current.Series)
	if os == version.Windows {
		t.Skip()
	}
	gc.TestingT(t)
}
