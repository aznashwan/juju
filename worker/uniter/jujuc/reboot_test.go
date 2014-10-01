// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc_test

import (
	"github.com/juju/cmd"
	gc "gopkg.in/check.v1"
	"launchpad.net/gnuflag"

	"github.com/juju/juju/testing"
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

func (s *JujuRebootSuite) TestJujuRebootCommand(c *gc.C) {
	var jujuRebootTests = []struct {
		summary 	string
		hctx    	*Context
		args    	[]string
		code    	int
		prio		jujuc.RebootPriority
	}{{
		summary:	"test reboot priority defaulting to RebootAfterHook",
		hctx:		&Context{giveError: false, rebootPrio: jujuc.RebootSkip},
		args:		[]string{},
		code:		0,
		prio:		jujuc.RebootAfterHook,
	}, {
		summary: "test reboot priority being set to RebootNow",
		hctx:    	&Context{giveError: false, rebootPrio: jujuc.RebootSkip},
		args:    	[]string{"now"},
		code:    	0,
		prio:		jujuc.RebootNow,
	}, {
		summary: "test a failed running of juju-reboot",
		hctx:    	&Context{giveError: true, rebootPrio: jujuc.RebootSkip},
		args:    	[]string{},
		code:    	1,
		prio:		jujuc.RebootAfterHook,
	}, {
		summary: "test a failed running with parameter provided",
		hctx:    	&Context{giveError: true, rebootPrio: jujuc.RebootSkip},
		args:    	[]string{"now"},
		code:    	1,
		prio:		jujuc.RebootNow,
	}, {
		summary: "test invalid args provided",
		hctx:    	&Context{giveError: false, rebootPrio: jujuc.RebootSkip},
		args:    	[]string{"way", " too", " many", " args"},
		code:    	0,
		prio:		jujuc.RebootSkip,
	}}

	for i, t := range jujuRebootTests {
		c.Logf("Test %d: %s", i, t.summary)

		com, err := jujuc.NewCommand(t.hctx, cmdString("juju-reboot"))
		c.Assert(err, gc.IsNil)
		ctx := testing.Context(c)
		code := cmd.Main(com, ctx, []string{t.args})
		c.Check(code, gc.Equals, t.code)
		c.Check(t.hctx.rebootPrio, gc.Equals, t.prio)
	}
}
