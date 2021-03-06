// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package service provides access to the service api facade.
// This facade contains api calls that are specific to services.
// As a rule of thumb, if the argument for an api requries a service name
// and affects only that service then the call belongs here.
package service

import (
	"github.com/juju/errors"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/constraints"
	"github.com/juju/juju/storage"
)

// Client allows access to the service API end point.
type Client struct {
	base.ClientFacade
	st     *api.State
	facade base.FacadeCaller
}

// NewClient creates a new client for accessing the service api.
func NewClient(st *api.State) *Client {
	frontend, backend := base.NewClientFacade(st, "Service")
	return &Client{ClientFacade: frontend, st: st, facade: backend}
}

// SetMetricCredentials sets the metric credentials for the service specified.
func (c *Client) SetMetricCredentials(service string, credentials []byte) error {
	creds := []params.ServiceMetricCredential{
		{service, credentials},
	}
	p := params.ServiceMetricCredentials{creds}
	results := new(params.ErrorResults)
	err := c.facade.FacadeCall("SetMetricCredentials", p, results)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(results.OneError())
}

// ServiceDeploy obtains the charm, either locally or from
// the charm store, and deploys it. It allows the specification of
// requested networks that must be present on the machines where the
// service is deployed. Another way to specify networks to include/exclude
// is using constraints.
func (c *Client) ServiceDeploy(
	charmURL string,
	serviceName string,
	numUnits int,
	configYAML string,
	cons constraints.Value,
	toMachineSpec string,
	networks []string,
	storage map[string]storage.Constraints,
) error {
	args := params.ServicesDeploy{
		Services: []params.ServiceDeploy{{
			ServiceName:   serviceName,
			CharmUrl:      charmURL,
			NumUnits:      numUnits,
			ConfigYAML:    configYAML,
			Constraints:   cons,
			ToMachineSpec: toMachineSpec,
			Networks:      networks,
			Storage:       storage,
		}},
	}
	var results params.ErrorResults
	err := c.facade.FacadeCall("ServicesDeploy", args, &results)
	if err != nil {
		return err
	}
	return results.OneError()
}
