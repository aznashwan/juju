// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/service/common"
	"github.com/juju/juju/service/systemd"
	coretesting "github.com/juju/juju/testing"
)

func Test(t *testing.T) {
	gc.TestingT(t)
}

type SystemdSuite struct {
	coretesting.BaseSuite
	service *systemd.Service
	initDir string
}

var _ = gc.Suite(&SystemdSuite{})

func (s *SystemdSuite) SetUpTest(c *gc.C) {
	s.initDir = c.MkDir()
	s.PatchValue(&systemd.InitDir, s.initDir)
	s.service = systemd.NewService(
		"dummy-service",
		common.Conf{
			Desc: "dummy service for testing",
			Cmd:  "some-command --execute",
		},
	)
}

// tests wether the InitDir of a new service is properly set to its default.
func (s *SystemdSuite) TestInitDirDefaulting(c *gc.C) {
	service := systemd.NewService("service", common.Conf{})
	c.Assert(service.Conf.InitDir, gc.Equals, s.initDir)
}

// test that servicePath() properly returns the full path of the service file.
func (s *SystemdSuite) TestServicePath(c *gc.C) {
	c.Assert(s.service.ServicePath(), gc.Equals, path.Join(s.initDir, "dummy-service.service"))
}

// buildDummyRunCommand returns a function with an identical signature to
// systemd.RunCommand that returns the strings given by args and a nil error.
// the returned function remembers the commands it recieved in *issuedcmds.
func (s *SystemdSuite) buildDummyRunCommand(issuedcmds *[]string, args ...string) func(string, ...string) (string, error) {
	*issuedcmds = []string{}
	return func(cmd string, cmdargs ...string) (string, error) {
		*issuedcmds = append(*issuedcmds, fmt.Sprintf("%s %s", cmd, strings.Join(cmdargs, " ")))
		return strings.Join(args, "\n"), nil
	}
}

// writeServiceFile writes a proper service file to the InitDir if good is set
// to true, and an improper one if good is set to false.
func (s *SystemdSuite) writeServiceFile(c *gc.C, good bool) {
	if good {
		contents, err := s.service.Render()
		c.Assert(err, jc.ErrorIsNil)
		err = ioutil.WriteFile(s.service.ServicePath(), contents, 0644)
		c.Assert(err, jc.ErrorIsNil)
	} else {
		err := ioutil.WriteFile(s.service.ServicePath(), []byte("nothing relevant"), 0644)
		c.Assert(err, jc.ErrorIsNil)
	}
}

// enabledServiceMessage returns the string corresponding to the output of
// `systemctl status` on the s.service if it were enabled.
func (s *SystemdSuite) enabledServiceMessage() string {
	return fmt.Sprintf("%s; enabled", s.service.ServicePath())
}

// serviceStatusCommand returns a slice of strings with a single element
// representing the systemctl command of statting the service represented by s.
func (s *SystemdSuite) serviceStatusCommand() []string {
	return []string{"systemctl --user status " + s.service.ServiceName()}
}

// test existsAndSame under various circumstances.
func (s *SystemdSuite) TestExistsAndSame(c *gc.C) {
	// non-existent service file
	exists, same, err := s.service.ExistsAndSame()
	c.Assert(exists, jc.IsFalse)
	c.Assert(same, jc.IsFalse)
	c.Assert(err, jc.ErrorIsNil)

	// properly installed service file with mismatching content
	s.writeServiceFile(c, false)
	exists, same, err = s.service.ExistsAndSame()
	c.Assert(exists, jc.IsTrue)
	c.Assert(same, jc.IsFalse)
	c.Assert(err, jc.ErrorIsNil)

	// properly installed service file with suitable content
	s.writeServiceFile(c, true)
	exists, same, err = s.service.ExistsAndSame()
	c.Assert(exists, jc.IsTrue)
	c.Assert(same, jc.IsTrue)
	c.Assert(err, jc.ErrorIsNil)
}

// test wether enabled properly recognises the status of the service.
func (s *SystemdSuite) TestEnabled(c *gc.C) {
	var issuedcmds []string

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, "totally not enabled"))
	c.Assert(s.service.Enabled(), jc.IsFalse)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))
	c.Assert(s.service.Enabled(), jc.IsTrue)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())
}

// test a completely fresh installation of the service using Install.
func (s *SystemdSuite) TestInstallFromScratch(c *gc.C) {
	var issuedcmds []string
	var expectedSystemctlCmds = []string{
		// not expected because of the lazy evaluation of same && s.enabled():
		// "systemctl --user status " + s.service.ServiceName(),
		// called in Install:
		"systemctl --user enable " + s.service.ServiceName(),
		// from the call to Start:
		"systemctl --user status " + s.service.ServiceName(),
		"systemctl --user start " + s.service.ServiceName(),
	}

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))

	c.Assert(s.service.Install(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, expectedSystemctlCmds)

	found, err := ioutil.ReadFile(s.service.ServicePath())
	c.Assert(err, jc.ErrorIsNil)
	expected, err := s.service.Render()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found, gc.DeepEquals, expected)
}

// test wether Install promptly returns if the service is already installed.
func (s *SystemdSuite) TestInstallDefaultsIfAlreadyInstalled(c *gc.C) {
	var issuedcmds []string

	s.writeServiceFile(c, true)
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))

	c.Assert(s.service.Install(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())
}

// test Installed's behavior under any possible set of parameters.
func (s *SystemdSuite) TestInstalled(c *gc.C) {
	var issuedcmds []string

	// non-existent service file
	c.Assert(s.service.Installed(), jc.IsFalse)

	// existing service file but un-enabled service
	s.writeServiceFile(c, true)
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, "totally not enabled"))

	c.Assert(s.service.Installed(), jc.IsFalse)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())

	// properly installed and enabled service file
	s.writeServiceFile(c, true)
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))

	c.Assert(s.service.Installed(), jc.IsTrue)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())
}

// test Exists'behavior under any possible set of parameters.
func (s *SystemdSuite) TestExists(c *gc.C) {
	var issuedcmds []string
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, "totally not enabled"))

	// non-existent service file, service disabled
	c.Assert(s.service.Exists(), jc.IsFalse)

	// existing service file with improper contents, service still disabled
	s.writeServiceFile(c, false)
	c.Assert(s.service.Exists(), jc.IsFalse)

	// existing service file with improper contents, service now enabled
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))
	c.Assert(s.service.Exists(), jc.IsFalse)

	// proper service file but un-enabled service
	s.writeServiceFile(c, true)
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, "totally not enabled"))
	c.Assert(s.service.Exists(), jc.IsFalse)

	// proper service file, service enabled
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))
	c.Assert(s.service.Exists(), jc.IsTrue)
}

// test Running under both possible scenarios.
func (s *SystemdSuite) TestRunning(c *gc.C) {
	var issuedcmds []string

	// `systemctl status` saying service is not running
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, "absolutely whatever"))

	c.Assert(s.service.Running(), jc.IsFalse)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())

	// `systemctl status` saying service is running
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, " Active: active (running) "))

	c.Assert(s.service.Running(), jc.IsTrue)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())
}

// test Start properly issues the ommands to start the service.
func (s *SystemdSuite) TestStartService(c *gc.C) {
	var issuedcmds []string
	expectedSystemctlCmds := []string{
		"systemctl --user status " + s.service.ServiceName(),
		"systemctl --user start " + s.service.ServiceName(),
	}

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, "neither active, nor running"))

	c.Assert(s.service.Start(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, expectedSystemctlCmds)
}

// test that Start does nothing is the process is already running.
func (s *SystemdSuite) TestStartDefaultsIfAlreadyRunning(c *gc.C) {
	var issuedcmds []string

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, " Active: active (running) "))

	c.Assert(s.service.Start(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())
}

// test Stop properly issues the command to stop the service.
func (s *SystemdSuite) TestStopService(c *gc.C) {
	var issuedcmds []string
	expectedSystemctlCmds := []string{
		"systemctl --user status " + s.service.ServiceName(),
		"systemctl --user stop " + s.service.ServiceName(),
	}

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, " Active: active (running) "))

	c.Assert(s.service.Stop(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, expectedSystemctlCmds)
}

// test if Stop does nothing if the service is already not running.
func (s *SystemdSuite) TestStopDefaultsIfNotRunning(c *gc.C) {
	var issuedcmds []string

	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, " not running, dammit! "))

	c.Assert(s.service.Stop(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, s.serviceStatusCommand())
}

// test if Remove properly cleans up an installed service.
func (s *SystemdSuite) TestRemoveService(c *gc.C) {
	var issuedcmds []string
	expectedSystemctlCmds := []string{
		"systemctl --user status " + s.service.ServiceName(),
		"systemctl --user disable " + s.service.ServiceName(),
	}

	s.writeServiceFile(c, true)
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, s.enabledServiceMessage()))

	c.Assert(s.service.Remove(), jc.ErrorIsNil)
	_, err := os.Stat(s.service.ServicePath())
	c.Assert(err, jc.Satisfies, os.IsNotExist)
	c.Assert(issuedcmds, gc.DeepEquals, expectedSystemctlCmds)
}

// test if Remove defaults if service does not show up as installed.
func (s *SystemdSuite) TestRemoveDefaultsIfNotInstalled(c *gc.C) {
	var issuedcmds []string
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds, " doesn't matter "))

	c.Assert(s.service.Remove(), jc.ErrorIsNil)
	c.Assert(issuedcmds, gc.DeepEquals, []string{})
}

// test StopAndRemove on a properly installed service.
func (s *SystemdSuite) TestStopAndRemove(c *gc.C) {
	var issuedcmds []string
	expectedSystemctlCmds := []string{
		"systemctl --user status " + s.service.ServiceName(),
		"systemctl --user stop " + s.service.ServiceName(),
		"systemctl --user status " + s.service.ServiceName(),
		"systemctl --user disable " + s.service.ServiceName(),
	}

	s.writeServiceFile(c, true)
	s.PatchValue(systemd.RunCommand, s.buildDummyRunCommand(&issuedcmds,
		" Active: active (running) ",
		s.enabledServiceMessage(),
	))

	c.Assert(s.service.StopAndRemove(), jc.ErrorIsNil)
	_, err := os.Stat(s.service.ServicePath())
	c.Assert(err, jc.Satisfies, os.IsNotExist)
	c.Assert(issuedcmds, gc.DeepEquals, expectedSystemctlCmds)
}
