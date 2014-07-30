// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc_test

import (
	"fmt"
	"runtime"

	"github.com/juju/cmd"
	"github.com/juju/loggo"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/testing"
	"github.com/juju/juju/worker/uniter/jujuc"
)

type JujuLogSuite struct {
	ContextSuite
}

var _ = gc.Suite(&JujuLogSuite{})

//Added assertLogs to suite to access s.cmdSuffix
func (s *JujuLogSuite) assertLogs(c *gc.C, ctx jujuc.Context, writer *loggo.TestWriter, unitname, badge string) {
	msg1 := "the chickens"
	msg2 := "are 110% AWESOME"
	com, err := jujuc.NewCommand(ctx, "juju-log"+s.cmdSuffix)
	c.Assert(err, gc.IsNil)
	for _, t := range []struct {
		args  []string
		level loggo.Level
	}{
		{
			level: loggo.INFO,
		}, {
			args:  []string{"--debug"},
			level: loggo.DEBUG,
		}, {
			args:  []string{"--log-level", "TRACE"},
			level: loggo.TRACE,
		}, {
			args:  []string{"--log-level", "info"},
			level: loggo.INFO,
		}, {
			args:  []string{"--log-level", "WaRnInG"},
			level: loggo.WARNING,
		}, {
			args:  []string{"--log-level", "error"},
			level: loggo.ERROR,
		},
	} {
		writer.Clear()
		c.Assert(err, gc.IsNil)

		args := append(t.args, msg1, msg2)
		code := cmd.Main(com, &cmd.Context{}, args)
		c.Assert(code, gc.Equals, 0)
		log := writer.Log()
		c.Assert(log, gc.HasLen, 1)
		c.Assert(log[0].Level, gc.Equals, t.level)
		c.Assert(log[0].Module, gc.Equals, fmt.Sprintf("unit.%s.juju-log", unitname))
		c.Assert(log[0].Message, gc.Equals, fmt.Sprintf("%s%s %s", badge, msg1, msg2))
	}
}

func (s *JujuLogSuite) TestBadges(c *gc.C) {
	tw := &loggo.TestWriter{}
	_, err := loggo.ReplaceDefaultWriter(tw)
	loggo.GetLogger("unit").SetLogLevel(loggo.TRACE)
	c.Assert(err, gc.IsNil)
	hctx := s.GetHookContext(c, -1, "")
	s.assertLogs(c, hctx, tw, "u/0", "")
	hctx = s.GetHookContext(c, 1, "u/1")
	s.assertLogs(c, hctx, tw, "u/0", "peer1:1: ")
}

func newJujuLogCommand(c *gc.C) cmd.Command {
	ctx := &Context{}
	//Cannot access s.cmdSuffix here
	if runtime.GOOS == "windows" {
		com, err := jujuc.NewCommand(ctx, "juju-log.exe")
		c.Assert(err, gc.IsNil)
		return com
	} else {
		com, err := jujuc.NewCommand(ctx, "juju-log")
		c.Assert(err, gc.IsNil)
		return com
	}
}

func (s *JujuLogSuite) TestRequiresMessage(c *gc.C) {
	com := newJujuLogCommand(c)
	testing.TestInit(c, com, nil, "no message specified")
}

func (s *JujuLogSuite) TestLogInitMissingLevel(c *gc.C) {
	com := newJujuLogCommand(c)
	testing.TestInit(c, com, []string{"-l"}, "flag needs an argument.*")

	com = newJujuLogCommand(c)
	testing.TestInit(c, com, []string{"--log-level"}, "flag needs an argument.*")
}

func (s *JujuLogSuite) TestLogInitMissingMessage(c *gc.C) {
	com := newJujuLogCommand(c)
	testing.TestInit(c, com, []string{"-l", "FATAL"}, "no message specified")

	com = newJujuLogCommand(c)
	testing.TestInit(c, com, []string{"--log-level", "FATAL"}, "no message specified")
}

func (s *JujuLogSuite) TestLogDeprecation(c *gc.C) {
	com := newJujuLogCommand(c)
	ctx, err := testing.RunCommand(c, com, "--format", "foo", "msg")
	c.Assert(err, gc.IsNil)
	c.Assert(testing.Stderr(ctx), gc.Equals, "--format flag deprecated for command \"juju-log\"")
}
