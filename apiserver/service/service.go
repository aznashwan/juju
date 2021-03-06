// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package service contains api calls for functionality
// related to deploying and managing services and their
// related charms.
package service

import (
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names"
	"gopkg.in/juju/charm.v5"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/params"
	jjj "github.com/juju/juju/juju"
	"github.com/juju/juju/state"
	statestorage "github.com/juju/juju/state/storage"
	"github.com/juju/juju/storage"
	"github.com/juju/juju/storage/provider"
)

var (
	logger = loggo.GetLogger("juju.apiserver.service")

	newStateStorage = statestorage.NewStorage
)

func init() {
	common.RegisterStandardFacade("Service", 1, NewAPI)
}

// Service defines the methods on the service API end point.
type Service interface {
	SetMetricCredentials(args params.ServiceMetricCredentials) (params.ErrorResults, error)
}

// API implements the service interface and is the concrete
// implementation of the api end point.
type API struct {
	check      *common.BlockChecker
	state      *state.State
	authorizer common.Authorizer
}

// NewAPI returns a new service API facade.
func NewAPI(
	st *state.State,
	resources *common.Resources,
	authorizer common.Authorizer,
) (*API, error) {
	if !authorizer.AuthClient() {
		return nil, common.ErrPerm
	}

	return &API{
		state:      st,
		authorizer: authorizer,
		check:      common.NewBlockChecker(st),
	}, nil
}

// SetMetricCredentials sets credentials on the service.
func (api *API) SetMetricCredentials(args params.ServiceMetricCredentials) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Creds)),
	}
	if len(args.Creds) == 0 {
		return result, nil
	}
	for i, a := range args.Creds {
		service, err := api.state.Service(a.ServiceName)
		if err != nil {
			result.Results[i].Error = common.ServerError(err)
			continue
		}
		err = service.SetMetricCredentials(a.MetricCredentials)
		if err != nil {
			result.Results[i].Error = common.ServerError(err)
		}
	}
	return result, nil
}

// ServicesDeploy fetches the charms from the charm store and deploys them.
func (api *API) ServicesDeploy(args params.ServicesDeploy) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Services)),
	}
	if err := api.check.ChangeAllowed(); err != nil {
		return result, errors.Trace(err)
	}
	owner := api.authorizer.GetAuthTag().String()
	for i, arg := range args.Services {
		err := DeployService(api.state, owner, arg)
		result.Results[i].Error = common.ServerError(err)
	}
	return result, nil
}

// DeployService fetches the charm from the charm store and deploys it.
// The logic has been factored out into a common function which is called by
// both the legacy API on the client facade, as well as the new service facade.
func DeployService(st *state.State, owner string, args params.ServiceDeploy) error {
	curl, err := charm.ParseURL(args.CharmUrl)
	if err != nil {
		return errors.Trace(err)
	}
	if curl.Revision < 0 {
		return errors.Errorf("charm url must include revision")
	}

	if args.ToMachineSpec != "" && names.IsValidMachine(args.ToMachineSpec) {
		_, err = st.Machine(args.ToMachineSpec)
		if err != nil {
			return errors.Annotatef(err, `cannot deploy "%v" to machine %v`, args.ServiceName, args.ToMachineSpec)
		}
	}

	// Try to find the charm URL in state first.
	ch, err := st.Charm(curl)
	if errors.IsNotFound(err) {
		// Clients written to expect 1.16 compatibility require this next block.
		if curl.Schema != "cs" {
			return errors.Errorf(`charm url has unsupported schema %q`, curl.Schema)
		}
		if err = AddCharmWithAuthorization(st, params.AddCharmWithAuthorization{
			URL: args.CharmUrl,
		}); err == nil {
			ch, err = st.Charm(curl)
		}
	}
	if err != nil {
		return errors.Trace(err)
	}

	storageConstraints := args.Storage
	if storageConstraints == nil {
		storageConstraints = make(map[string]storage.Constraints)
	}
	// Validate the storage parameters against the charm metadata,
	// and ensure there are no conflicting parameters.
	if err := validateCharmStorage(args, ch); err != nil {
		return err
	}
	// Handle stores with no corresponding constraints.
	for store, charmStorage := range ch.Meta().Storage {
		if _, ok := args.Storage[store]; ok {
			// TODO(axw) if pool is not specified, we should set it to
			// the environment's default pool.
			continue
		}
		if charmStorage.Shared {
			// TODO(axw) get the environment's default shared storage
			// pool, and create constraints here.
			return errors.Errorf(
				"no constraints specified for shared charm storage %q",
				store,
			)
		}
		if charmStorage.CountMin <= 0 {
			continue
		}
		if charmStorage.Type != charm.StorageFilesystem {
			// TODO(axw) clarify what the rules are for "block" kind when
			// no constraints are specified. For "filesystem" we use rootfs.
			return errors.Errorf(
				"no constraints specified for %v charm storage %q",
				charmStorage.Type,
				store,
			)
		}
		storageConstraints[store] = storage.Constraints{
			// The pool is the provider type since rootfs provider has no configuration.
			Pool:  string(provider.RootfsProviderType),
			Count: uint64(charmStorage.CountMin),
		}
	}

	var settings charm.Settings
	if len(args.ConfigYAML) > 0 {
		settings, err = ch.Config().ParseSettingsYAML([]byte(args.ConfigYAML), args.ServiceName)
	} else if len(args.Config) > 0 {
		// Parse config in a compatible way (see function comment).
		settings, err = parseSettingsCompatible(ch, args.Config)
	}
	if err != nil {
		return errors.Trace(err)
	}
	// Convert network tags to names for any given networks.
	requestedNetworks, err := networkTagsToNames(args.Networks)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = jjj.DeployService(st,
		jjj.DeployServiceParams{
			ServiceName: args.ServiceName,
			// TODO(dfc) ServiceOwner should be a tag
			ServiceOwner:   owner,
			Charm:          ch,
			NumUnits:       args.NumUnits,
			ConfigSettings: settings,
			Constraints:    args.Constraints,
			ToMachineSpec:  args.ToMachineSpec,
			Networks:       requestedNetworks,
			Storage:        storageConstraints,
		})
	return err
}

// ServiceSetSettingsStrings updates the settings for the given service,
// taking the configuration from a map of strings.
func ServiceSetSettingsStrings(service *state.Service, settings map[string]string) error {
	ch, _, err := service.Charm()
	if err != nil {
		return err
	}
	// Parse config in a compatible way (see function comment).
	changes, err := parseSettingsCompatible(ch, settings)
	if err != nil {
		return err
	}
	return service.UpdateConfigSettings(changes)
}

func validateCharmStorage(args params.ServiceDeploy, ch *state.Charm) error {
	if len(args.Storage) == 0 {
		return nil
	}
	if len(args.ToMachineSpec) != 0 {
		// TODO(axw) when we support dynamic disk provisioning, we can
		// relax this. We will need to consult the storage provider to
		// decide whether or not this is allowable.
		return errors.New("cannot specify storage and machine placement")
	}
	// Remaining validation is done in state.AddService.
	return nil
}

func networkTagsToNames(tags []string) ([]string, error) {
	netNames := make([]string, len(tags))
	for i, tag := range tags {
		t, err := names.ParseNetworkTag(tag)
		if err != nil {
			return nil, err
		}
		netNames[i] = t.Id()
	}
	return netNames, nil
}

// parseSettingsCompatible parses setting strings in a way that is
// compatible with the behavior before this CL based on the issue
// http://pad.lv/1194945. Until then setting an option to an empty
// string caused it to reset to the default value. We now allow
// empty strings as actual values, but we want to preserve the API
// behavior.
func parseSettingsCompatible(ch *state.Charm, settings map[string]string) (charm.Settings, error) {
	setSettings := map[string]string{}
	unsetSettings := charm.Settings{}
	// Split settings into those which set and those which unset a value.
	for name, value := range settings {
		if value == "" {
			unsetSettings[name] = nil
			continue
		}
		setSettings[name] = value
	}
	// Validate the settings.
	changes, err := ch.Config().ParseSettingsStrings(setSettings)
	if err != nil {
		return nil, err
	}
	// Validate the unsettings and merge them into the changes.
	unsetSettings, err = ch.Config().ValidateSettings(unsetSettings)
	if err != nil {
		return nil, err
	}
	for name := range unsetSettings {
		changes[name] = nil
	}
	return changes, nil
}
