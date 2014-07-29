// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing_test

import (
	"os"
	"runtime"

	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/juju/osenv"
	"github.com/juju/juju/testing"
)

type TestingBaseSuite struct {
	testing.BaseSuite
}

var _ = gc.Suite(&TestingBaseSuite{})

// added appropriate wWindows-specific parameters
const (
	winHome       = `C:\\home`
	winJujuHome   = `C:\\home\\juju`
	linuxHome     = "/home/eric"
	linuxJujuHome = "/home/eric/juju"
)

func (s *TestingBaseSuite) SetUpTest(c *gc.C) {
	// made the setup OS-specific
	if runtime.GOOS == "windows" {
		utils.SetHome(winHome)
		os.Setenv("JUJU_HOME", winJujuHome)
		osenv.SetJujuHome(winJujuHome)
	} else {
		utils.SetHome(linuxHome)
		os.Setenv("JUJU_HOME", linuxJujuHome)
		osenv.SetJujuHome(linuxJujuHome)
	}

	s.BaseSuite.SetUpTest(c)
}

func (s *TestingBaseSuite) TearDownTest(c *gc.C) {
	s.BaseSuite.TearDownTest(c)

	// Test that the environment is restored.
	// made the tearsown OS-specific
	if runtime.GOOS == "windows" {
		c.Assert(utils.Home(), gc.Equals, winHome)
		c.Assert(os.Getenv("JUJU_HOME"), gc.Equals, winJujuHome)
	} else {
		c.Assert(utils.Home(), gc.Equals, linuxHome)
		c.Assert(os.Getenv("JUJU_HOME"), gc.Equals, linuxJujuHome)
	}
}

func (s *TestingBaseSuite) TestFakeHomeReplacesEnvironment(c *gc.C) {
	// made the testing OS-specific
	if runtime.GOOS == "windows" {
		c.Assert(utils.Home(), gc.Not(gc.Equals), winHome)
		c.Assert(os.Getenv("JUJU_HOME"), gc.Equals, "")
	} else {
		c.Assert(utils.Home(), gc.Not(gc.Equals), linuxHome)
		c.Assert(os.Getenv("JUJU_HOME"), gc.Equals, "")
	}
}
