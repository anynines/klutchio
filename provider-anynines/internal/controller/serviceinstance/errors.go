/*
Copyright 2024 Klutch Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serviceinstance

import utilerr "github.com/anynines/klutch/provider-anynines/pkg/utilerr"

const (
	errNotServiceInstance = utilerr.PlainUserErr("managed resource is not a ServiceInstance custom resource")

	errGetServiceInstance    = "cannot get ServiceInstance"
	errUpdateServiceInstance = "cannot update ServiceInstance"
	errDeleteServiceInstance = "cannot delete ServiceInstance"

	errGetCreds     = "cannot get credentials"
	errGetPC        = "cannot get ProviderConfig"
	errNewClient    = "cannot create new Client"
	errTrackPCUsage = "cannot track ProviderConfig usage"

	errInstanceIDStatusUnset = "InstanceID status field is unset"
	errPlanIDUnset           = "instance's Status.AtProvider.PlanID field must be set but is unset"
	errServiceIDUnset        = "instance's Status.AtProvider.ServiceID field must be set but is unset"

	errGetOperation    = "failed to get operation status"
	errOperationFailed = "operation failed"
)

var (
	errInstanceIDNotUnique = utilerr.FromStr("Could not ensure uniqueness of generated UID for Service Instance")
)

type errServiceNameNotFound struct {
	name string
}

func (e errServiceNameNotFound) Error() string {
	return `no entry for the service name "` + e.name + `"`
}
func (e errServiceNameNotFound) UserDisplay() error {
	return e
}

type errPlanNameNotFound struct {
	name string
}

func (e errPlanNameNotFound) Error() string {
	return `no entry for the plan name "` + e.name + `"`
}
func (e errPlanNameNotFound) UserDisplay() error {
	return &e
}
