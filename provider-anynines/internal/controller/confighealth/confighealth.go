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

package confighealth

import (
	"context"
	"fmt"
	"time"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"

	v1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	credhelp "github.com/anynines/klutchio/provider-anynines/internal/controller/utils"
	client "github.com/anynines/klutchio/provider-anynines/pkg/client/osb"
)

const (
	// healthCheckTimeout is the maximum time that a health check reconciliation may take
	healthCheckTimeout = 10 * time.Second
	// successCheckInterval defines the time to wait between successful health checks
	successCheckInterval = 3 * time.Minute
	// failureCheckInterval defines the time to wait before retrying a failed health check
	failureCheckInterval = 30 * time.Second
)

// Event reasons
const (
	ReasonCheckSuccess string = "CheckSuccess"
	ReasonCheckFailure string = "CheckFailure"
)

// Setup adds a controller that reconciles ProviderConfigs by periodically checking if the
// backend they describe is reachable.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := fmt.Sprintf("health/%s", v1.ProviderConfigGroupKind)

	r := reconciler{
		kube:         mgr.GetClient(),
		log:          o.Logger.WithValues("controller", name),
		nowFn:        time.Now,
		newServiceFn: client.NewOsbService,
		recorder:     mgr.GetEventRecorderFor(name),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1.ProviderConfig{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type reconciler struct {
	kube         k8sclient.Client
	log          logging.Logger
	nowFn        func() time.Time
	newServiceFn func(username, password []byte, url string) (osbclient.Client, error)
	recorder     record.EventRecorder
}

func (r reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	timeoutContext, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	var pc v1.ProviderConfig

	if err := r.kube.Get(ctx, req.NamespacedName, &pc); err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
		return ctrl.Result{}, err
	}

	now := r.nowFn()
	requeueAfter := successCheckInterval

	// this is necessary for multiple controllers and if multiple events are inserted into the control loop
	if isCheckNeeded(&pc, now) {
		log := r.log.WithValues("request", req)
		log.Debug("Performing Check")
		// by passing a dedicated context, we put a limit on how long the check is allowed to take
		// FIXME the a9s client should handle contexts correctly, it currently ignores them
		updated := r.getUpdatedProviderConfig(timeoutContext, &pc, now)
		status := updated.Status.Health.LastStatus
		log.Debug("Check complete", "status", status)

		// use parent context in case timeoutContext has exceeded its deadline
		if err := r.kube.Status().Patch(ctx, updated, k8sclient.MergeFrom(&pc)); err != nil {
			return ctrl.Result{}, err
		}

		// When the check is failing, recheck after a different interval
		if !status {
			requeueAfter = failureCheckInterval
		}

		// When the status changes, record an event
		if pc.Status.Health.LastCheckTime == nil || status != pc.Status.Health.LastStatus {
			if status {
				r.recorder.Event(&pc, corev1.EventTypeNormal, ReasonCheckSuccess,
					"ProviderConfig is now healthy")
			} else {
				r.recorder.Eventf(&pc, corev1.EventTypeWarning, ReasonCheckFailure,
					"Health check failed: %v", updated.Status.Health.LastMessage)
			}
		}
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: requeueAfter,
	}, nil
}

func isCheckNeeded(pc *v1.ProviderConfig, now time.Time) bool {
	if pc.Status.Health.LastCheckTime == nil {
		// Health was never checked before
		return true
	}

	sinceLastCheck := now.Sub(pc.Status.Health.LastCheckTime.Time)
	lastHealthy := pc.Status.Health.LastStatus
	checkInterval := successCheckInterval

	if !lastHealthy {
		checkInterval = failureCheckInterval
	}

	return sinceLastCheck > checkInterval
}

func (r reconciler) getUpdatedProviderConfig(ctx context.Context, pc *v1.ProviderConfig, now time.Time) *v1.ProviderConfig {
	status, message := r.performCheck(ctx, pc)

	updated := pc.DeepCopy()

	updated.Status.Health.LastStatus = status
	updated.Status.Health.LastMessage = message

	lastCheckTime := metav1.NewTime(now)
	updated.Status.Health.LastCheckTime = &lastCheckTime
	return updated
}

func (r reconciler) performCheck(ctx context.Context, pc *v1.ProviderConfig) (bool, string) {
	credentials, err := credhelp.GetCredentialsFromProvider(ctx, pc, r.kube)
	if err != nil {
		return false, fmt.Sprintf("Extracting credentials: %v", err)
	}

	svc, err := r.newServiceFn(credentials.Username, credentials.Password, pc.Spec.Url)
	if err != nil {
		return false, fmt.Sprintf("Constructing service client: %v", err)
	}

	if err := svc.CheckAvailability(); err != nil {
		return false, err.Error()
	}

	return true, "Available"
}
