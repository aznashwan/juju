// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package systemd

var RunCommand = &runCommand

func (s *Service) ServiceName() string {
	return s.serviceName()
}

func (s *Service) ServicePath() string {
	return s.servicePath()
}

func (s *Service) Validate() error {
	return s.validate()
}

func (s *Service) Render() ([]byte, error) {
	return s.render()
}

func (s *Service) ExistsAndSame() (bool, bool, error) {
	return s.existsAndSame()
}

func (s *Service) Enabled() bool {
	return s.enabled()
}
