// Copyright 2015 Canonical Ltd.
// Copyright 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

// The cloudinit package implements a way of creating
// a cloud-init configuration file which is CentOS compatible.
// See https://help.ubuntu.com/community/CloudInit.
package cloudinit

import (
	"fmt"

	"github.com/juju/utils/packaging"
	"github.com/juju/utils/packaging/configuration"
	"github.com/juju/utils/proxy"
	"gopkg.in/yaml.v1"
)

// CentOSCloudConfig is the cloudconfig type specific to CentOS machines.
// It simply contains a cloudConfig and adds the package management related
// methods for CentOS, which are mostly modeled as runcmds.
// It implements the cloudinit.Config interface.
type CentOSCloudConfig struct {
	*cloudConfig
}

// SetPackageProxy is defined on the PackageProxyConfig interface.
func (cfg *CentOSCloudConfig) SetPackageProxy(url string) {
	cfg.SetAttr("package_proxy", url)
}

// addPackageProxyCmd is a helper function which returns the corresponding runcmd
// to apply the package proxy settings on a CentOS machine.
func addPackageProxyCmd(cfg CloudConfig, url string) string {
	return fmt.Sprintf("/bin/echo 'proxy=%s' >> /etc/yum.conf", url)
}

// UnsetPackageProxy is defined on the PackageProxyConfig interface.
func (cfg *CentOSCloudConfig) UnsetPackageProxy() {
	cfg.UnsetAttr("package_proxy")
}

// PackageProxy is defined on the PackageProxyConfig interface.
func (cfg *CentOSCloudConfig) PackageProxy() string {
	proxy, _ := cfg.attrs["package_proxy"].(string)
	return proxy
}

// SetPackageMirror is defined on the PackageMirrorConfig interface.
func (cfg *CentOSCloudConfig) SetPackageMirror(url string) {
	cfg.SetAttr("package_mirror", url)
}

// addPackageMirrorCmd is a helper function that returns the corresponding runcmds
// to apply the package mirror settings on a CentOS machine.
func addPackageMirrorCmd(cfg CloudConfig, url string) string {
	return fmt.Sprintf(configuration.ReplaceCentOSMirror, url)
}

// UnsetPackageMirror is defined on the PackageMirrorConfig interface.
func (cfg *CentOSCloudConfig) UnsetPackageMirror() {
	cfg.UnsetAttr("package_mirror")
}

// PackageMirror is defined on the PackageMirrorConfig interface.
func (cfg *CentOSCloudConfig) PackageMirror() string {
	mirror, _ := cfg.attrs["package_mirror"].(string)
	return mirror
}

// AddPackageSource is defined on the PackageSourcesConfig interface.
func (cfg *CentOSCloudConfig) AddPackageSource(src packaging.PackageSource) {
	cfg.attrs["package_sources"] = append(cfg.PackageSources(), src)
}

// PackageSources is defined on the PackageSourcesConfig interface.
func (cfg *CentOSCloudConfig) PackageSources() []packaging.PackageSource {
	sources, _ := cfg.attrs["package_sources"].([]packaging.PackageSource)
	return sources
}

// AddPackagePreferences is defined on the PackageSourcesConfig interface.
func (cfg *CentOSCloudConfig) AddPackagePreferences(prefs packaging.PackagePreferences) {
	// TODO (aznashwan): research a way of using yum-priorities in the
	// context of a single package and implement the appropriate runcmds.
}

// PackagePreferences is defined on the PackageSourcesConfig interface.
func (cfg *CentOSCloudConfig) PackagePreferences() []packaging.PackagePreferences {
	// TODO (aznashwan): add this when priorities in yum make sense.
	return []packaging.PackagePreferences{}
}

// Render is defined on the the Renderer interface.
func (cfg *CentOSCloudConfig) RenderYAML() ([]byte, error) {
	// check for package proxy setting and add commands:
	var proxy string
	if proxy = cfg.PackageProxy(); proxy != "" {
		cfg.AddRunCmd(addPackageProxyCmd(cfg, proxy))
		cfg.UnsetPackageProxy()
	}

	// check for package mirror settings and add commands:
	var mirror string
	if mirror = cfg.PackageMirror(); mirror != "" {
		cfg.AddRunCmd(addPackageMirrorCmd(cfg, mirror))
		cfg.UnsetPackageMirror()
	}

	// add appropriate commands for package sources configuration:
	srcs := cfg.PackageSources()
	for _, src := range srcs {
		cfg.AddScripts(addPackageSourceCmds(cfg, src)...)
	}
	cfg.UnsetAttr("package_sources")

	data, err := yaml.Marshal(cfg.attrs)
	if err != nil {
		return nil, err
	}

	//restore
	//TODO(bogdanteleaga, aznashwan): check that this actually works
	// We have the same thing in ubuntu as well
	cfg.SetPackageProxy(proxy)
	cfg.SetPackageMirror(mirror)
	cfg.SetAttr("package_sources", srcs)

	return append([]byte("#cloud-config\n"), data...), nil
}

func (cfg *CentOSCloudConfig) RenderScript() (string, error) {
	return renderScriptCommon(cfg)
}

// AddCloudArchiveCloudTools is defined on the AdvancedPackagingConfig.
func (cfg *CentOSCloudConfig) AddCloudArchiveCloudTools() {
	src, pref := configuration.GetCloudArchiveSource(cfg.series)
	cfg.AddPackageSource(src)
	cfg.AddPackagePreferences(pref)
}

func (cfg *CentOSCloudConfig) getCommandsForAddingPackages() ([]string, error) {
	var cmds []string

	if newMirror := cfg.PackageMirror(); newMirror != "" {
		cmds = append(cmds, LogProgressCmd("Changing package mirror does not yet work on CentOS"))
		// TODO(bogdanteleaga, aznashwan): This should work after a further PR
		// where we add more mirrror options values to environs.Config
		cmds = append(cmds, addPackageMirrorCmd(cfg, newMirror))
	}

	for _, src := range cfg.PackageSources() {
		// TODO(bogdanteleaga. aznashwan): Keys are usually offered by repositories, and you need to
		// accept them. Check how this can be done non interactively.
		cmds = append(cmds, LogProgressCmd("Adding yum repository: %s", src.Url))
		cmds = append(cmds, cfg.paccmder.AddRepositoryCmd(src.Url))
	}

	// TODO(bogdanteleaga. aznashwan): Research what else needs to be done here

	// Define the "package_get_loop" function
	cmds = append(cmds, configuration.PackageManagerLoopFunction)

	if cfg.SystemUpdate() {
		cmds = append(cmds, LogProgressCmd("Running yum update"))
		cmds = append(cmds, "package_manager_loop "+cfg.paccmder.UpdateCmd())
	}
	if cfg.SystemUpgrade() {
		cmds = append(cmds, LogProgressCmd("Running yum upgrade"))
		cmds = append(cmds, "package_manager_loop "+cfg.paccmder.UpgradeCmd())
	}

	pkgs := cfg.Packages()
	for _, pkg := range pkgs {
		cmds = append(cmds, LogProgressCmd("Installing package: %s", pkg))
		cmds = append(cmds, "package_manager_loop "+cfg.paccmder.InstallCmd(pkg))
	}
	return cmds, nil
}

// AddPackageCommands is defined on the AdvancedPackagingConfig interface.
func (cfg *CentOSCloudConfig) AddPackageCommands(
	packageProxySettings proxy.Settings,
	packageMirror string,
	addUpdateScripts bool,
	addUpgradeScripts bool,
) {
	addPackageCommandsCommon(
		cfg,
		packageProxySettings,
		packageMirror,
		addUpdateScripts,
		addUpgradeScripts,
		cfg.series,
	)
}

// updatePackages is defined on the AdvancedPackagingConfig interface.
func (cfg *CentOSCloudConfig) updatePackages() {
	packages := []string{
		"curl",
		"bridge-utils",
		"rsyslog-gnutls",
		"cloud-utils",
	}

	// The required packages need to come from the correct repo.
	// For precise, that might require an explicit repo targeting.
	// We cannot just pass packages below, because
	// this will generate install commands which older
	// versions of cloud-init (e.g. 0.6.3 in precise) will
	// interpret incorrectly (see bug http://pad.lv/1424777).
	for _, pack := range packages {
		if cfg.pacconfer.IsCloudArchivePackage(pack) {
			// On precise, we need to pass a --target-release entry in
			// pieces for it to work:
			for _, p := range cfg.pacconfer.ApplyCloudArchiveTarget(pack) {
				cfg.AddPackage(p)
			}
		} else {
			cfg.AddPackage(pack)
		}
	}
}

//TODO(bogdanteleaga, aznashwan): On ubuntu when we render the conf as yaml we
//have apt_proxy and when we render it as bash we use the equivalent of this.
//However on centOS even when rendering the YAML we use a helper function
//addPackageProxyCmds. Research if calling the same is fine.
func (cfg *CentOSCloudConfig) updateProxySettings(proxySettings proxy.Settings) {
}
