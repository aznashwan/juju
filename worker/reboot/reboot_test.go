// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot_test

import (
	stdtesting "testing"
	"time"

	"github.com/juju/names"
	// jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/api"
	apireboot "github.com/juju/juju/api/reboot"
	jujutesting "github.com/juju/juju/juju/testing"
	coretesting "github.com/juju/juju/testing"
	//  "github.com/juju/juju/utils/rebootstate"
	"github.com/juju/juju/state"
	"github.com/juju/juju/worker"
	"github.com/juju/juju/worker/reboot"
)

const worstCase = 5 * time.Second

func TestPackage(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}

type RebootSuite struct {
	jujutesting.JujuConnSuite

	stateMachine	*state.Machine
	apiState		*api.State
	rebootState		*apireboot.State
}

var _ = gc.Suite(&RebootSuite{})

func (s *RebootSuite) SetUpSuite(c *gc.C) {
	s.JujuConnSuite.SetUpSuite(c)
}

func (s *RebootSuite) SetUpTest(c *gc.C) {
	s.PatchValue(&reboot.LockDir, c.MkDir())
	s.PatchValue(&agent.DefaultDataDir, c.MkDir())

	s.JujuConnSuite.SetUpTest(c)
	s.apiState, s.stateMachine = s.OpenAPIAsNewMachine(c)

	var err error
	s.rebootState, err = s.apiState.Reboot()
	c.Assert(err, gc.IsNil)
}

func (s *RebootSuite) TearDownTest(c *gc.C) {
	s.JujuConnSuite.TearDownTest(c)
}

func (s *RebootSuite) TearDownSuite(c *gc.C) {
	s.JujuConnSuite.TearDownSuite(c)
}

type mockAgentConfig struct {
	agent.Config
	tag names.Tag
}

func (cfg *mockAgentConfig) Tag() names.Tag {
	return cfg.tag
}

func agentConfig(tag names.Tag) agent.Config {
	return &mockAgentConfig{tag: tag}
}

func (s *RebootSuite) newRebootWorker(c *gc.C) worker.Worker {
	rebooter, err := reboot.NewReboot(s.rebootState,
		agentConfig(s.stateMachine.Tag().(names.MachineTag)))
	c.Assert(err, gc.IsNil)

	return rebooter
}

func (s *RebootSuite) TestStop(c *gc.C) {
	rebooter := s.newRebootWorker()

	c.Assert(worker.Stop(rebooter), gc.IsNil)
}
