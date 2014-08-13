// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package debug_test

import (
	"testing"
	"path/filepath"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/worker/uniter/debug"
)

type DebugHooksCommonSuite struct{}

var _ = gc.Suite(&DebugHooksCommonSuite{})

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

// TestHooksContext tests the behaviour of HooksContext.
func (*DebugHooksCommonSuite) TestHooksContext(c *gc.C) {
	ctx := debug.NewHooksContext("foo/8")
	c.Assert(ctx.Unit, gc.Equals, "foo/8")
	c.Assert(ctx.FlockDir, gc.Equals, "/tmp")
	ctx.FlockDir = "/var/lib/juju"
	// Despite the paths being Linux specific from the get-go, the basic 
	// functionality is still tested well on Windows too.
	c.Assert(ctx.ClientFileLock(), gc.Equals, filepath.FromSlash("/var/lib/juju/juju-unit-foo-8-debug-hooks"))
	c.Assert(ctx.ClientExitFileLock(), gc.Equals, filepath.FromSlash("/var/lib/juju/juju-unit-foo-8-debug-hooks-exit"))
}
