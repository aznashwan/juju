// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc_test

import (
	"github.com/juju/cmd"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"launchpad.net/gnuflag"

	"github.com/juju/juju/worker/uniter/jujuc"
)

type JujuRebootSuite struct{}

var _ = gc.Suite(&JujuRebootSuite{})

func (s *JujuRebootSuite) TestNewJujuRebootCommand(c *gc.C) {
	cmd := jujuc.NewJujuRebootCommand(nil)
	c.Assert(cmd, gc.DeepEquals, &jujuc.JujuRebootCommand{})
}

func (s *JujuRebootSuite) TestInfo(c *gc.C) {
	rebootCmd := jujuc.NewJujuRebootCommand(nil)
	expectedCmdInfo := &cmd.Info{
		Name:    "juju-reboot",
		Args:    "",
		Purpose: "Reboot the machine we are running on",
	}

	cmdInfo := rebootCmd.Info()

	c.Assert(cmdInfo, gc.DeepEquals, expectedCmdInfo)
}

func (s *JujuRebootSuite) TestSetFlags(c *gc.C) {
	rebootCmd := jujuc.JujuRebootCommand{Now: true}
	fs := &gnuflag.FlagSet{}

	rebootCmd.SetFlags(fs)

	flag := fs.Lookup("now")
	c.Assert(flag, gc.NotNil)
}

func (s *JujuRebootSuite) TestRunRebootNow(c *gc.C) {
	ctx := Context{}
	rebootCmd := jujuc.NewJujuRebootCommand(&ctx)
	jujuRebootCmd, ok := rebootCmd.(*jujuc.JujuRebootCommand)
	c.Assert(ok, jc.IsTrue)
	jujuRebootCmd.Now = true

	err := jujuRebootCmd.Run(nil)
	c.Assert(err, gc.IsNil)

	c.Assert(ctx.priority, gc.Equals, jujuc.RebootNow)
}

func (s *JujuRebootSuite) TestRunRebootAfterHook(c *gc.C) {
	ctx := Context{}
	rebootCmd := jujuc.NewJujuRebootCommand(&ctx)
	jujuRebootCmd, ok := rebootCmd.(*jujuc.JujuRebootCommand)
	c.Assert(ok, jc.IsTrue)
	jujuRebootCmd.Now = false

	err := jujuRebootCmd.Run(nil)
	c.Assert(err, gc.IsNil)

	c.Assert(ctx.priority, gc.Equals, jujuc.RebootAfterHook)
}
