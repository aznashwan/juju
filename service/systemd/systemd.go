// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"text/template"

	"github.com/juju/errors"
	"github.com/juju/utils"

	"github.com/juju/juju/service/common"
)

// This regexp will match the common output of a running service's
// `systemctl status unit.service` output.
var runningRegexp = regexp.MustCompile(`.*Active: active \(running\).*`)

// The systemd service file directory for the current user.
var InitDir = path.Join(utils.Home(), ".config/systemd/user")

// Service is the structure which provides a handle on a systemd service.
type Service struct {
	Name string
	Conf common.Conf
}

// New service returns a new *systemd.Service with the associated name and
// initial Conf. If no InitDir is provided, it defaults to /etc/systemd/system.
func NewService(name string, conf common.Conf) *Service {
	if conf.InitDir == "" {
		conf.InitDir = InitDir
	}
	return &Service{Name: name, Conf: conf}
}

// UpdateConfig allows for the resetting of the Conf associated to s.
func (s *Service) UpdateConfig(conf common.Conf) {
	s.Conf = conf
}

// serviceName simply returns the fully qualified name of the service.
func (s *Service) serviceName() string {
	return s.Name + ".service"
}

// servicePath returns the full path to the service file associated with s.
func (s *Service) servicePath() string {
	return path.Join(s.Conf.InitDir, s.serviceName())
}

// validate returns an error if the Service is not properly defined.
func (s *Service) validate() error {
	if s.Name == "" {
		return errors.New("missing Name")
	}
	if s.Conf.InitDir == "" {
		return errors.New("missing InitDir")
	}
	if s.Conf.Desc == "" {
		return errors.New("missing Desc")
	}
	if s.Conf.Cmd == "" {
		return errors.New("missing Cmd")
	}
	return nil
}

// render returns the systemd service file in slice of bytes form.
func (s *Service) render() ([]byte, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	if err := serviceTemplate.Execute(&buff, s.Conf); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// runCommand is simply a variable for utils.RunCommand which was aliased for
// testing purposes.
var runCommand = utils.RunCommand

// existsAndSame is a helper function which determines whether a service file
// with the same name exists and whether it has the same contents.
func (s *Service) existsAndSame() (exists, same bool, err error) {
	var expected, found []byte

	if expected, err = s.render(); err != nil {
		return false, false, errors.Trace(err)
	}

	if found, err = ioutil.ReadFile(s.servicePath()); err != nil {
		var reterr error
		if os.IsNotExist(err) {
			reterr = nil
		} else {
			reterr = errors.Trace(err)
		}

		return false, false, reterr
	}

	return true, bytes.Equal(found, expected), nil
}

// enabled returns true if the service has been enabled by the user in systemd.
func (s *Service) enabled() bool {
	enabledRegexp := regexp.MustCompile(fmt.Sprintf(".*%s; enabled.*", s.servicePath()))

	out, err := runCommand("systemctl", "--user", "status", s.serviceName())
	if err != nil {
		return false
	}

	return enabledRegexp.Match([]byte(out))
}

// Install properly places the service file of s in the InitDir and starts the
// service through systemd.
// NOTE: a service will be enabled by default for it to count as installed.
func (s *Service) Install() error {
	// check if the service is already installed
	if exists, same, err := s.existsAndSame(); err != nil {
		return errors.Trace(err)
	} else if same && s.enabled() {
		return nil
	} else if exists {
		if err := s.StopAndRemove(); err != nil {
			return errors.Trace(err)
		}
	}

	// write the service file to the InitDir
	if contents, err := s.render(); err != nil {
		return errors.Trace(err)
	} else if err := ioutil.WriteFile(s.servicePath(), contents, 0644); err != nil {
		return errors.Trace(err)
	}

	// run the enabling command to complete the install
	if _, err := runCommand("systemctl", "--user", "enable", s.serviceName()); err != nil {
		return err
	}

	return s.Start()
}

// Installed returns true if a service file was correctly set in the InitDir
// and was properly enabled.
func (s *Service) Installed() bool {
	if _, err := os.Stat(s.servicePath()); err != nil {
		return false
	} else {
		return s.enabled()
	}
}

// Exists returns a boolean representing whether or not the exact service file
// which s would render to already exists and if it has been enabled.
func (s *Service) Exists() bool {
	_, same, _ := s.existsAndSame()
	return same && s.enabled()
}

// Running returns a boolean of whether or not the service is actively running.
func (s *Service) Running() bool {
	out, err := runCommand("systemctl", "--user", "status", s.serviceName())
	if err != nil {
		return false
	}

	return runningRegexp.Match([]byte(out))
}

// Start issues the command to systemd to immediately start the service.
func (s *Service) Start() error {
	if s.Running() {
		return nil
	}

	_, err := runCommand("systemctl", "--user", "start", s.serviceName())
	return err
}

// Stop issues the command to systemd to immediately stop the service.
func (s *Service) Stop() error {
	if !s.Running() {
		return nil
	}

	_, err := runCommand("systemctl", "--user", "stop", s.serviceName())
	return err
}

// Remove disables the service and deletes the existing service file associated to s.
func (s *Service) Remove() error {
	if !s.Installed() {
		return nil
	}

	// we do not care about the returned error because `systemctl --user disable`
	// simply classifies the disabling operation as succesfull even if the
	// service is already disabled or does not exist entirely.
	runCommand("systemctl", "--user", "disable", s.serviceName())

	return os.Remove(s.servicePath())
}

// StopAndRemove stops the service and removes the service file from InitDir.
func (s *Service) StopAndRemove() error {
	if err := s.Stop(); err != nil {
		return err
	}

	return s.Remove()
}

// InstallCommands returns the shell commands to install, enable and run the service.
func (s *Service) InstallCommands() ([]string, error) {
	contents, err := s.render()
	if err != nil {
		return nil, err
	}

	return []string{
		fmt.Sprintf("cat >> %s << 'EOF'\n%sEOF\n", s.servicePath(), contents),
		"systemctl --user enable " + s.Name,
		"systemctl --user start " + s.Name,
	}, nil
}

var serviceTemplate = template.Must(template.New("").Parse(`
[Unit]
Description={{.Desc}}
After=syslog.target
After=network.target

[Service]
Type=forking
ExecStart={{.Cmd}}
Restart=always
TimeoutSec=300

[Install]
WantedBy=multi-user.target
`[1:]))
