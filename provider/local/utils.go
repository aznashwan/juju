// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package local

import (
	"github.com/juju/juju/version"
	"github.com/juju/utils/packaging/manager"
	"github.com/juju/utils/proxy"
)

// isPackageInstalled is a helper function which instantiates a new
// PackageManager for the current system and checks whether the given package is
// installed.
var isPackageInstalled = func(pack string) bool {
	pacman, _ := manager.NewPackageManager(version.Current.Series)
	return pacman.IsInstalled(pack)
}

// detectPackageProxies is a helper function which instantiates a new
// PackageManager for the current system and checks for package manager proxy
// settings.
var detectPackageProxies = func() (proxy.Settings, error) {
	pacman, _ := manager.NewPackageManager(version.Current.Series)
	return pacman.GetProxySettings()
}
