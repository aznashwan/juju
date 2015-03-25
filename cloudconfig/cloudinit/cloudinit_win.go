// Copyright 2015 Canonical Ltd.
// Copyright 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

// The cloudinit package implements a way of creating
// a cloud-init configuration file.
// See https://help.ubuntu.com/community/CloudInit.
package cloudinit

import (
	"github.com/juju/utils/packaging"
	"github.com/juju/utils/proxy"
)

// WindowsCloudConfig is the cloudconfig type specific to Ubuntu machines
// It simply contains a cloudConfig with the added package management-related
// methods for the Ubuntu version of cloudinit.
// It satisfies the cloudinit.CloudConfig interface.
type WindowsCloudConfig struct {
	*cloudConfig
}

// SetPackageProxy implements PackageProxyConfig.
func (cfg *WindowsCloudConfig) SetPackageProxy(url string) {
	return
}

// UnsetPackageProxy implements PackageProxyConfig.
func (cfg *WindowsCloudConfig) UnsetPackageProxy() {
	return
}

// PackageProxy implements PackageProxyConfig.
func (cfg *WindowsCloudConfig) PackageProxy() string {
	return ""
}

// SetPackageMirror implements PackageMirrorConfig.
func (cfg *WindowsCloudConfig) SetPackageMirror(url string) {
	return
}

// UnsetPackageMirror implements PackageMirrorConfig.
func (cfg *WindowsCloudConfig) UnsetPackageMirror() {
	return
}

// PackageMirror implements PackageMirrorConfig.
func (cfg *WindowsCloudConfig) PackageMirror() string {
	return ""
}

// AddPackageSource implements PackageSourcesConfig.
func (cfg *WindowsCloudConfig) AddPackageSource(src packaging.PackageSource) {
	return
}

// PackageSources implements PackageSourcesConfig.
func (cfg *WindowsCloudConfig) PackageSources() []packaging.PackageSource {
	// NOTE: this should not ever get called, so it is safe to return nil here:
	return nil
}

// AddPackagePreferences implements PackageSourcesConfig.
func (cfg *WindowsCloudConfig) AddPackagePreferences(prefs packaging.PackagePreferences) {
	return
}

// PackagePreferences implements PackageSourcesConfig.
func (cfg *WindowsCloudConfig) PackagePreferences() []packaging.PackagePreferences {
	// NOTE: this should not ever get called, so it is safe to return nil here:
	return nil
}

// RenderYAML implements RenderConfig.
func (cfg *WindowsCloudConfig) RenderYAML() ([]byte, error) {
	return cfg.renderWindows()
}

// RenderScript implements RenderConfig.
func (cfg *WindowsCloudConfig) RenderScript() (string, error) {
	// NOTE: This shouldn't really be called on windows as it's used only for
	// initialization via ssh or on local providers.
	script, err := cfg.renderWindows()
	if err != nil {
		return "", err
	}

	return string(script), err
}

// getCommandsForAddingPackages implements RenderConfig..
func (cfg *WindowsCloudConfig) getCommandsForAddingPackages() ([]string, error) {
	return nil, nil
}

// renderWindows is a helper function which renders the runCmds of the Windows
// CloudConfig to a PowerShell script.
func (cfg *WindowsCloudConfig) renderWindows() ([]byte, error) {
	winCmds := cfg.RunCmds()
	var script []byte
	newline := "\r\n"
	header := "#ps1_sysnative\r\n"
	script = append(script, header...)
	for _, cmd := range winCmds {
		script = append(script, newline...)
		script = append(script, cmd...)

	}
	return script, nil
}

// AddPackageCommands implements AdvancedPackagingConfig.
func (cfg *WindowsCloudConfig) AddPackageCommands(
	aptProxySettings proxy.Settings,
	aptMirror string,
	addUpdateScripts bool,
	addUpgradeScripts bool,
) {
	// Who knows; one day chocolaty might be here...
	return
}

// AddCloudArchiveCloudTools implements AdvancedPackagingConfig.
func (cfg *WindowsCloudConfig) AddCloudArchiveCloudTools() {
}

// updatePackages implements AdvancedPackagingConfig.
func (cfg *WindowsCloudConfig) updatePackages() {
	return
}

// updateProxySettings implements AdvancedPackagingConfig.
func (cfg *WindowsCloudConfig) updateProxySettings(proxy.Settings) {
	return
}
