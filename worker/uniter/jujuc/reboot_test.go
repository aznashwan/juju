// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc_test

import (
	"github.com/juju/cmd"
	gc "gopkg.in/check.v1"
	"launchpad.net/gnuflag"
	"launchpad.net/gnuflags"

	"github.com/juju/juju/worker/uniter/jujuc"
)

type JujuRebootSuite struct {
	ContextSuite
}

var _ = gc.Suite(&JujuRebootSuite{})

func (s *JujuRebootSuite) TestNewJujuRebootCommand(c *gc.C) {
	cmd := jujuc.NewJujuRebootCommand(nil)
	c.Assert(cmd, gc.DeepEquals, &JujuRebootCommand{})

	ctx := s.GetHookContext(c, -1, "some remote")
	cmd := jujuc.NewJujuRebootCommand(ctx)
	c.Assert(cmd, gc.DeepEquals, &JujuRebootCommand{ctx: ctx})
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
	fs := &gnuflags.FlagSet{}

	rebootCmd.SetFlags(fs)

	flag := fs.LookUp("now")
	c.Assert(flag, gc.DeepEquals,
		&gnuflag.Flag{
			Name:  "now",
			Usage: "reboot immediately, killing the invoking process",
			Value: "false",
		})
}

type dummyContextStruct struct {
	Context
	priority jujuc.RebootPriority
}

func (ctx *dummyContextStruct) RequestReboot(prio jujuc.RebootPriority) {
	ctx.priority = prio
}

func (s *JujuRebootSuite) TestRunRebootNow(c *gc.C) {
	ctx := dummyContextStruct{Context: Context{}}
	rebootCmd := jujuc.NewJujuRebootCommand(ctx)
	rebootCmd.Now = true

	err := rebootCmd.Run(nil)
	c.Assert(err, gc.IsNil)

	c.Assert(ctx.priority, gc.Equals, jujuc.RebootNow)
}

func (s *JujuRebootSuite) TestRunRebootAfterHook(c *gc.C) {
	ctx := dummyContextStruct{Context: Context{}}
	rebootCmd := jujuc.NewJujuRebootCommand(ctx)
	rebootCmd.Now = false

	err := rebootCmd.Run(nil)
	c.Assert(err, gc.IsNil)

	c.Assert(ctx.priority, gc.Equals, jujuc.RebootAfterHook)
}
