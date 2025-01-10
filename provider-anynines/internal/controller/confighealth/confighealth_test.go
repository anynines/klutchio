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
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	fakeosb "github.com/anynines/klutchio/clients/a9s-open-service-broker/fake"
	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	a9stest "github.com/anynines/klutchio/provider-anynines/internal/controller/test"
)

func TestMain(m *testing.M) {
	if err := apisv1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add API github.com/anynines/klutchio/provider-anynines/apis/v1 to scheme")
	}

	os.Exit(m.Run())
}

func successReaction() *fakeosb.CheckAvailabilityReaction {
	reaction := fakeosb.CheckAvailabilityReaction(func() error { return nil })
	return &reaction
}

func failureReaction(message string) *fakeosb.CheckAvailabilityReaction {
	reaction := fakeosb.CheckAvailabilityReaction(func() error { return errors.New(message) })
	return &reaction
}

func TestReconcile(t *testing.T) {
	t0 := time.Now().Truncate(time.Second)

	cases := map[string]struct {
		now                       time.Time
		pc                        *apisv1.ProviderConfig
		secret                    *corev1.Secret
		expectedHealth            *apisv1.ProviderConfigHealth
		checkAvailabilityReaction *fakeosb.CheckAvailabilityReaction
		expectedCreds             []string
		expectedEvents            []string
	}{
		"new config performs check initially": {
			now: t0,
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("test-secret", "test", "username"),
					a9stest.SecretRef("test-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0),
				a9stest.HealthLastStatus(true),
				a9stest.HealthLastMessage("Available"),
			),
			checkAvailabilityReaction: successReaction(),
			expectedCreds:             []string{"test", "secure-test-password"},
			expectedEvents:            []string{"Normal CheckSuccess ProviderConfig is now healthy"},
		},

		"successful check is not retried after half the successCheckInterval": {
			now: t0.Add(successCheckInterval / 2),
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("test-secret", "test", "username"),
					a9stest.SecretRef("test-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
				a9stest.WithProviderConfigHealth(
					a9stest.NewProviderConfigHealth(
						a9stest.HealthLastCheckTime(t0),
						a9stest.HealthLastStatus(true),
						a9stest.HealthLastMessage("Available"),
					)),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0),
				a9stest.HealthLastStatus(true),
				a9stest.HealthLastMessage("Available"),
			),
			// No event recorded, if no check is performed
			expectedEvents: []string{},
		},

		"successful check is retried after 1.5 times the successCheckInterval": {
			now: t0.Add(3 * successCheckInterval / 2),
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("test-secret", "test", "username"),
					a9stest.SecretRef("test-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
				a9stest.WithProviderConfigHealth(
					a9stest.NewProviderConfigHealth(
						a9stest.HealthLastCheckTime(t0),
						a9stest.HealthLastStatus(true),
						a9stest.HealthLastMessage("Available"),
					)),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0.Add(3*successCheckInterval/2)),
				a9stest.HealthLastStatus(true),
				a9stest.HealthLastMessage("Available"),
			),
			checkAvailabilityReaction: successReaction(),
			expectedCreds:             []string{"test", "secure-test-password"},
			// No event recorded if health status is unchanged (healthy -> healthy)
			expectedEvents: []string{},
		},

		"successful check can become failing after successCheckInterval": {
			now: t0.Add(3 * successCheckInterval / 2),
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("test-secret", "test", "username"),
					a9stest.SecretRef("test-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
				a9stest.WithProviderConfigHealth(
					a9stest.NewProviderConfigHealth(
						a9stest.HealthLastCheckTime(t0),
						a9stest.HealthLastStatus(true),
						a9stest.HealthLastMessage("Available"),
					)),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0.Add(3*successCheckInterval/2)),
				a9stest.HealthLastStatus(false),
				a9stest.HealthLastMessage("Nope!"),
			),
			checkAvailabilityReaction: failureReaction("Nope!"),
			expectedCreds:             []string{"test", "secure-test-password"},
			// Change from healthy to unhealthy records an event
			expectedEvents: []string{"Warning CheckFailure Health check failed: Nope!"},
		},

		"failing check is not retried after 0.5 times the failureCheckInterval": {
			now: t0.Add(failureCheckInterval / 2),
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("test-secret", "test", "username"),
					a9stest.SecretRef("test-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
				a9stest.WithProviderConfigHealth(
					a9stest.NewProviderConfigHealth(
						a9stest.HealthLastCheckTime(t0),
						a9stest.HealthLastStatus(false),
						a9stest.HealthLastMessage("Nope!"),
					)),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0),
				a9stest.HealthLastStatus(false),
				a9stest.HealthLastMessage("Nope!"),
			),
			expectedEvents: []string{},
		},

		"failing check is retried after 1.5 times the failureCheckInterval": {
			now: t0.Add(3 * failureCheckInterval / 2),
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("test-secret", "test", "username"),
					a9stest.SecretRef("test-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
				a9stest.WithProviderConfigHealth(
					a9stest.NewProviderConfigHealth(
						a9stest.HealthLastCheckTime(t0),
						a9stest.HealthLastStatus(false),
						a9stest.HealthLastMessage("Nope!"),
					)),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0.Add(3*failureCheckInterval/2)),
				a9stest.HealthLastStatus(false),
				a9stest.HealthLastMessage("Nope!"),
			),
			checkAvailabilityReaction: failureReaction("Nope!"),
			expectedCreds:             []string{"test", "secure-test-password"},
			// No event recorded if status is unchanged (unhealthy -> unhealthy)
			expectedEvents: []string{},
		},

		"referencing the wrong secret is unhealthy": {
			now: t0,
			pc: a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
				a9stest.WithProviderConfigSpec("test.com",
					a9stest.SecretRef("wrong-secret", "test", "username"),
					a9stest.SecretRef("wrong-secret", "test", "password"),
					xpv1.CredentialsSourceSecret),
			),
			secret: a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
				a9stest.Namespace[corev1.Secret]("test"),
				a9stest.WithKey("username", "test"),
				a9stest.WithKey("password", "secure-test-password"),
			),
			expectedHealth: a9stest.NewProviderConfigHealth(
				a9stest.HealthLastCheckTime(t0),
				a9stest.HealthLastStatus(false),
				a9stest.HealthLastMessage(`Extracting credentials: cannot get credentials secret: secrets "wrong-secret" not found`),
			),
			expectedEvents: []string{`Warning CheckFailure Health check failed: Extracting credentials: cannot get credentials secret: secrets "wrong-secret" not found`},
		},
	}

	for name, tc := range cases {
		// Rebind tc into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				CheckAvailabilityReaction: tc.checkAvailabilityReaction,
			})

			creds := []string{}

			eventRecorder := record.NewFakeRecorder(10)

			r := reconciler{
				kube: fake.NewClientBuilder().
					WithRuntimeObjects(tc.pc, tc.secret).
					WithStatusSubresource(tc.pc).
					WithScheme(scheme.Scheme).
					Build(),

				log: a9stest.TestLogger(t),

				nowFn: func() time.Time { return tc.now },

				newServiceFn: func(username, password []byte, url string) (osbclient.Client, error) {
					creds = append(creds, string(username), string(password))
					return fakeOSB, nil
				},

				recorder: eventRecorder,
			}

			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: k8sclient.ObjectKeyFromObject(tc.pc),
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			reloaded := &apisv1.ProviderConfig{}

			if err := r.kube.Get(context.Background(), k8sclient.ObjectKeyFromObject(tc.pc), reloaded); err != nil {
				t.Fatalf("Failed to reload providerconfig: %v", err)
			}

			if !reflect.DeepEqual(&reloaded.Status.Health, tc.expectedHealth) {
				t.Fatalf("Expected health to be %+v, but got %+v", tc.expectedHealth, reloaded.Status.Health)
			}

			if tc.expectedCreds != nil && !reflect.DeepEqual(creds, tc.expectedCreds) {
				t.Fatalf("Expected credentials to be %+v, but got %+v", tc.expectedCreds, creds)
			}

			capturedEvents := []string{}
			for {
				done := false
				select {
				case event := <-eventRecorder.Events:
					capturedEvents = append(capturedEvents, event)
				default:
					done = true
				}
				if done {
					break
				}
			}

			if !reflect.DeepEqual(capturedEvents, tc.expectedEvents) {
				t.Fatalf("Captured events differ from expected:\nExpected: %+v\nCaptured: %+v",
					tc.expectedEvents, capturedEvents)
			}
		})
	}
}
