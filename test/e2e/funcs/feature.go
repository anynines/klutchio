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

package funcs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/anynines/klutch/test/e2e/utils"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/yaml"
)

const (
	crossplaneHelmRepositoryURL = "https://charts.crossplane.io/stable"
	crossplaneHelmChartVersion  = "1.15.0"
)

// AllOf runs the supplied functions in order.
func AllOf(fns ...features.Func) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		for _, fn := range fns {
			ctx = fn(ctx, t, c)
		}
		return ctx
	}
}

func ForEach[T interface{}](samples []T, buildStep func(T) features.Func) features.Func {
	fns := make([]features.Func, 0, len(samples))
	for _, sample := range samples {
		fns = append(fns, buildStep(sample))
	}
	return AllOf(fns...)
}

// ResourcesCreatedWithin fails a test if the supplied resources are not found
// to exist within the supplied duration.
func ResourcesCreatedWithin(d time.Duration, dir, pattern string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Error(err)
			return ctx
		}

		list := &unstructured.UnstructuredList{}
		for _, o := range rs {
			u := asUnstructured(o)
			list.Items = append(list.Items, *u)
			t.Logf("Waiting %s for %s to exist...", d, identifier(u))
		}

		if err := wait.For(conditions.New(c.Client().Resources()).ResourcesFound(list), wait.WithTimeout(d)); err != nil {
			t.Errorf("resources did not exist: %v", err)
			return ctx
		}

		t.Logf("%d resources found to exist", len(rs))
		return ctx
	}
}

// ResourcesDeletedWithin fails a test if the supplied resources are not deleted
// within the supplied duration.
func ResourcesDeletedWithin(d time.Duration, dir, pattern string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {

		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Error(err)
			return ctx
		}

		list := &unstructured.UnstructuredList{}
		for _, o := range rs {
			u := asUnstructured(o)
			list.Items = append(list.Items, *u)
			t.Logf("Waiting %s for %s to be deleted...", d, identifier(u))
		}

		if err := wait.For(conditions.New(c.Client().Resources()).ResourcesDeleted(list), wait.WithTimeout(d)); err != nil {
			t.Errorf("resources not deleted: %v", err)
			return ctx
		}

		t.Logf("%d resources deleted", len(rs))
		return ctx
	}
}

// ResourcesHaveConditionWithin fails a test if the supplied resources do not
// have (i.e. become) the supplied conditions within the supplied duration.
func ResourcesHaveConditionWithin(d time.Duration, dir, pattern string, cds ...xpv1.Condition) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {

		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Error(err)
			return ctx
		}

		reasons := make([]string, len(cds))
		for i := range cds {
			reasons[i] = string(cds[i].Reason)
		}
		desired := strings.Join(reasons, ", ")

		list := &unstructured.UnstructuredList{}
		for _, o := range rs {
			u := asUnstructured(o)
			list.Items = append(list.Items, *u)
			t.Logf("Waiting %s for %s to become %s...", d, identifier(u), desired)
		}

		match := func(o k8s.Object) bool {
			u := asUnstructured(o)
			s := xpv1.ConditionedStatus{}
			_ = fieldpath.Pave(u.Object).GetValueInto("status", &s)

			for _, want := range cds {
				got := s.GetCondition(want.Type)
				t.Logf("for type: %v want status %v got status %v", want.Type, want.Status, got.Status)
				if got.Status != want.Status {
					return false
				}
			}

			return true
		}

		if err := wait.For(conditions.New(c.Client().Resources()).ResourcesMatch(list, match), wait.WithTimeout(d)); err != nil {
			y, _ := yaml.Marshal(list.Items)
			t.Errorf("resources did not have desired conditions: %s: %v:\n\n%s\n\n", desired, err, y)
			return ctx
		}

		t.Logf("%d resources have desired conditions: %s", len(rs), desired)
		return ctx
	}
}

func MatchingResourcesDeletedWithin(d time.Duration, kind, apiVersion string, opts ...resources.ListOption) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		list := &unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"kind":       kind,
				"apiVersion": apiVersion,
			},
		}
		err := wait.For(func(context.Context) (bool, error) {
			err := c.Client().Resources().List(ctx, list, opts...)

			if err != nil {
				t.Logf("List failed: %v", err)
				return false, err
			}

			t.Logf("List success, have %d items", len(list.Items))

			return len(list.Items) == 0, nil
		})
		if err != nil {
			t.Fatalf("Resources weren't deleted within %v. Still have: %v -- error: %v",
				d, list, err)
		}
		return ctx
	}
}

// ResourceHasConditionWithin fails a test if the resource identified by the given name, namespace, apiVersion
// and kind does not reach the supplied conditions within the supplied duration
func ResourceHasConditionWithin(d time.Duration, name, namespace, apiVersion, kind string, cds ...xpv1.Condition) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		reasons := make([]string, len(cds))
		for i := range cds {
			reasons[i] = fmt.Sprintf("%s=%s", cds[i].Type, cds[i].Status)
		}
		desired := strings.Join(reasons, ", ")

		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": apiVersion,
				"kind":       kind,
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": namespace,
				},
			},
		}

		t.Logf("Waiting %s for %s to become %s...", d, identifier(obj), desired)

		match := func(o k8s.Object) bool {
			u := asUnstructured(o)
			s := xpv1.ConditionedStatus{}
			_ = fieldpath.Pave(u.Object).GetValueInto("status", &s)

			for _, want := range cds {
				have := s.GetCondition(want.Type)
				t.Logf("check %s=%s: %v", want.Type, want.Status, have.Status == want.Status)
				if have.Status != want.Status {
					return false
				}
			}

			return true
		}
		if err := wait.For(conditions.New(c.Client().Resources()).ResourceMatch(obj, match), wait.WithTimeout(d)); err != nil {
			y, _ := yaml.Marshal(obj)
			t.Errorf("resource did not reach desired conditions: %s: %v:\n\n%s\n\n", desired, err, y)
		}
		return ctx
	}
}

// ApplyResources applies all manifests under the supplied directory that match
// the supplied glob pattern (e.g. *.yaml). It uses server-side apply - fields
// are managed by the supplied field manager. It fails the test if any supplied
// resource cannot be applied successfully.
func ApplyResources(manager, dir, pattern string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		dfs := os.DirFS(dir)

		if err := decoder.DecodeEachFile(ctx, dfs, pattern, ApplyHandler(c.Client().Resources(), manager)); err != nil {
			t.Fatal(err)
			return ctx
		}

		files, _ := fs.Glob(dfs, pattern)
		t.Logf("Applied resources from %s (matched %d manifests)", filepath.Join(dir, pattern), len(files))
		return ctx
	}
}

// ApplyInvalid does the same as ApplyResources, except it expects all operations to fail
func ApplyInvalid(manager, dir, pattern, expectedMessage string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		dfs := os.DirFS(dir)

		if err := decoder.DecodeEachFile(ctx, dfs, pattern, invalidApplyHandler(c.Client().Resources(), manager, t, expectedMessage)); err != nil {
			t.Fatal(err)
			return ctx
		}

		files, _ := fs.Glob(dfs, pattern)
		t.Logf(" resources from %s (matched %d manifests)", filepath.Join(dir, pattern), len(files))
		return ctx
	}
}

// DeleteResources deletes (from the environment) all resources defined by the
// manifests under the supplied directory that match the supplied glob pattern
// (e.g. *.yaml).
func DeleteResources(dir, pattern string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		dfs := os.DirFS(dir)

		if err := decoder.DecodeEachFile(ctx, dfs, pattern, decoder.DeleteHandler(c.Client().Resources())); err != nil {
			t.Fatal(err)
			return ctx
		}

		files, _ := fs.Glob(dfs, pattern)
		t.Logf("Deleted resources from %s (matched %d manifests)", filepath.Join(dir, pattern), len(files))
		return ctx
	}
}

// ApplyHandler is a decoder.Handler that uses server-side apply to apply the
// supplied object.
func ApplyHandler(r *resources.Resources, manager string) decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		return r.GetControllerRuntimeClient().Patch(ctx, obj, client.Apply, client.FieldOwner(manager), client.ForceOwnership)
	}
}

// invalidApplyHandler is a decoder.Handler that expects a validation error when applying the resource
func invalidApplyHandler(r *resources.Resources, manager string, t *testing.T, expectedMessage string) decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		err := r.GetControllerRuntimeClient().Patch(ctx, obj, client.Apply, client.FieldOwner(manager), client.ForceOwnership)
		if err == nil {
			t.Fatalf("Expected object %v not to apply cleanly, but it did!", obj)
			return nil
		}
		error := err.Error()
		if !strings.Contains(error, expectedMessage) {
			t.Fatalf("Expected error to contain %v, but it did not: %v", expectedMessage, error)
		}

		t.Logf("Validation success! Got (expected) error: %v", error)

		return nil
	}
}

// asUnstructured turns an arbitrary runtime.Object into an *Unstructured. If
// it's already a concrete *Unstructured it just returns it, otherwise it
// round-trips it through JSON encoding. This is necessary because types that
// are registered with our scheme will be returned as Objects backed by the
// concrete type, whereas types that are not will be returned as *Unstructured.
func AsUnstructured(o runtime.Object) *unstructured.Unstructured {
	if u, ok := o.(*unstructured.Unstructured); ok {
		return u
	}

	u := &unstructured.Unstructured{}
	j, _ := json.Marshal(o)
	_ = json.Unmarshal(j, u)
	return u
}

func asUnstructured(o runtime.Object) *unstructured.Unstructured {
	if u, ok := o.(*unstructured.Unstructured); ok {
		return u
	}

	u := &unstructured.Unstructured{}
	j, _ := json.Marshal(o)
	_ = json.Unmarshal(j, u)
	return u
}

// identifier returns the supplied resource's kind, name, and (if any)
// namespace.
func identifier(o k8s.Object) string {
	k := o.GetObjectKind().GroupVersionKind().Kind
	if k == "" {
		k = reflect.TypeOf(o).Elem().Name()
	}
	if o.GetNamespace() == "" {
		return fmt.Sprintf("%s %s", k, o.GetName())
	}
	return fmt.Sprintf("%s %s/%s", k, o.GetNamespace(), o.GetName())
}

// ManagedResourceOfClaimHasConditionWithin fails a test if the MR of the
// supplied resources do not have the supplied conditions within the supplied
// duration.
func ManagedResourceOfClaimHasConditionWithin(d time.Duration, dir, pattern string, cds ...xpv1.Condition) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {

		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Error(err)
			return ctx
		}

		reasons := make([]string, len(cds))
		for i := range cds {
			reasons[i] = string(cds[i].Reason)
		}
		desired := strings.Join(reasons, ", ")

		list := &unstructured.UnstructuredList{}
		for _, o := range rs {
			u := asUnstructured(o)

			claimName := o.GetName()
			claimNamespace := o.GetNamespace()
			claimKind := o.GetObjectKind().GroupVersionKind().Kind
			claimGroupVersionKind := o.GetObjectKind().GroupVersionKind().Group + "/" + o.GetObjectKind().GroupVersionKind().Version

			rg := utils.NewResourceGetter(ctx, t, c)
			claim := rg.Get(claimName, claimNamespace, claimGroupVersionKind, claimKind)
			r := utils.ResourceValue(t, claim, "spec", "resourceRef")

			xr := rg.Get(r["name"], "", r["apiVersion"], r["kind"])
			mrefs := utils.ResourceSliceValue(t, xr, "spec", "resourceRefs")

			if len(mrefs) == 0 {
				t.Fatalf("Found no managed resources for %s", identifier(u))
			}

			t.Logf("Waiting %s for %d MRs of %s to become %s...", d, len(mrefs), identifier(u), desired)

			for _, mref := range mrefs {
				t.Logf("Waiting for %s (%s/%s)...", mref["name"], mref["apiVersion"], mref["kind"])
				err := wait.For(func(context.Context) (done bool, err error) {
					mr := rg.Get(mref["name"], "", mref["apiVersion"], mref["kind"])
					if mr == nil {
						return false, nil
					}

					conditions, found, err := unstructured.NestedSlice(mr.Object, "status", "conditions")
					if err != nil {
						return false, fmt.Errorf("failed to retrieve conditions: %w", err)
					}
					if !found {
						return false, nil
					}

					for _, want := range cds {
						var matching map[string]interface{}
						for _, condition := range conditions {
							conditionMap := condition.(map[string]interface{})
							typeValue := conditionMap["type"].(string)
							if typeValue == string(want.Type) {
								matching = conditionMap
								break
							}
						}
						if matching == nil {
							t.Fatalf("Failed to find condition of type %v, have %v", string(want.Type), conditions)
						}

						haveStatus := matching["status"].(string)
						haveReason := matching["reason"].(string)

						t.Logf("%s: want.Type:%v want.Status:%s want.Reason:%s got.Status:%s got.Reason:%s", mref["name"], want.Type, string(want.Status), string(want.Reason), haveStatus, haveReason)

						if string(want.Status) != haveStatus || string(want.Reason) != haveReason {
							return false, nil
						}
					}
					return true, nil
				}, wait.WithTimeout(d), wait.WithInterval(10*time.Second))
				if err != nil {
					t.Fatalf("managed resources did not have desired conditions")
				}
			}
			list.Items = append(list.Items, *u)
		}
		return ctx
	}
}

func ManagedResourceOfClaimSatisfiesWithin(d time.Duration, dir, pattern string, check func(*unstructured.Unstructured) bool) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {

		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Error(err)
			return ctx
		}

		list := &unstructured.UnstructuredList{}
		for _, o := range rs {
			u := asUnstructured(o)

			claimName := o.GetName()
			claimNamespace := o.GetNamespace()
			claimKind := o.GetObjectKind().GroupVersionKind().Kind
			claimGroupVersionKind := o.GetObjectKind().GroupVersionKind().Group + "/" + o.GetObjectKind().GroupVersionKind().Version

			rg := utils.NewResourceGetter(ctx, t, c)
			claim := rg.Get(claimName, claimNamespace, claimGroupVersionKind, claimKind)
			r := utils.ResourceValue(t, claim, "spec", "resourceRef")

			xr := rg.Get(r["name"], "", r["apiVersion"], r["kind"])
			mrefs := utils.ResourceSliceValue(t, xr, "spec", "resourceRefs")

			if len(mrefs) == 0 {
				t.Fatalf("Found no managed resources for %s", identifier(u))
			}

			t.Logf("Waiting %s for %d MRs of %s to pass the check...", d, len(mrefs), identifier(u))

			for _, mref := range mrefs {
				err := wait.For(func(context.Context) (done bool, err error) {
					mr := rg.Get(mref["name"], "", mref["apiVersion"], mref["kind"])
					if mr == nil {
						return false, nil
					}

					return check(mr), nil
				}, wait.WithTimeout(d))
				if err != nil {
					t.Fatalf("managed resources did not have desired conditions")
				}
			}
			list.Items = append(list.Items, *u)
		}
		return ctx
	}
}

// CompositeResourceEstablishedAndOffered ensures that a CompositeResourceDefinition (aka XRD) with the given xrdName
// exists, and has it's `Established` and `Offered` conditions set to `True`.
func CompositeResourceEstablishedAndOffered(xrdName string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Logf("Checking CompositeResourceDefinition %s", xrdName)
		err := wait.For(func(context.Context) (done bool, err error) {
			rg := utils.NewResourceGetter(ctx, t, c)
			xrd := rg.Get(xrdName, "", "apiextensions.crossplane.io/v1", "CompositeResourceDefinition")

			s := xpv1.ConditionedStatus{}
			_ = fieldpath.Pave(xrd.Object).GetValueInto("status", &s)

			cds := []xpv1.Condition{
				{Type: "Established", Status: "True"},
				{Type: "Offered", Status: "True"},
			}

			for _, want := range cds {
				have := s.GetCondition(want.Type)
				t.Logf("check %s=%s: %v", want.Type, want.Status, have.Status == want.Status)
				if have.Status != want.Status {
					return false, nil
				}
			}

			return true, nil
		}, wait.WithInterval(1*time.Second))
		if err != nil {
			t.Fatalf("Failed to wait for CompositeResourceDefinition: %v", err)
		}
		return ctx
	}
}

func ResourceExists(name, namespace, apiVersion, kind string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": apiVersion,
				"kind":       kind,
			},
		}
		if err := c.Client().Resources().Get(ctx, name, namespace, obj); err != nil {
			t.Fatalf("Failed to get resource: name=%v, namespace=%v, apiVersion=%v, kind=%v",
				name, namespace, apiVersion, kind)
		}
		t.Logf("Found expected resource: name=%v, namespace=%v, apiVersion=%v, kind=%v",
			name, namespace, apiVersion, kind)
		return ctx
	}
}

func ResourceDoesNotExist(name, namespace, apiVersion, kind string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": apiVersion,
				"kind":       kind,
			},
		}

		err := c.Client().Resources().Get(ctx, name, namespace, obj)
		if err != nil && !errors.IsNotFound(err) {
			t.Fatalf("Failed to get resource: name=%v, namespace=%v, apiVersion=%v, kind=%v",
				name, namespace, apiVersion, kind)
		} else if err == nil {
			t.Fatalf("Resource exists: name=%v, namespace=%v, apiVersion=%v, kind=%v",
				name, namespace, apiVersion, kind)
		}

		t.Logf("Resource not found: name=%v, namespace=%v, apiVersion=%v, kind=%v",
			name, namespace, apiVersion, kind)

		return ctx
	}
}

// ShellCommand runs a command with the given arguments to completion.
// Fails the test if the command fails to start or if it returns a non-zero status.
func ShellCommand(name string, args ...string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Logf("Running shell command %v with args %v", name, args)
		cmd := exec.Command(name, args...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			t.Fatalf("Failed to create stderr pipe")
		}

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start shell command %v with args %v: %v", name, args, err)
		}

		errOutput, _ := io.ReadAll(stderr)

		if err := cmd.Wait(); err != nil {
			t.Fatalf("Failed to complete shell command %v with args %v: %v\nStderr output: %s",
				name, args, err, errOutput)
		}
		return ctx
	}
}

// AwaitDeployments waits for the given deployment to reach condition=available
func AwaitDeployment(name string, namespace string, waitOpts ...wait.Option) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Logf("Waiting for deployment %s in namespace %s to become ready...", name, namespace)
		condition := conditions.New(c.Client().Resources()).
			DeploymentAvailable(name, namespace)
		if err := wait.For(condition, waitOpts...); err != nil {
			t.Fatalf("Failed to wait for deployment %s in namespace %s to become ready: %v",
				name, namespace, err)
		}
		return ctx
	}
}

// InstallCrossplane installs the crossplane helm chart and waits for it's deployments to become available.
func InstallCrossplane() features.Func {
	return AllOf(
		ShellCommand("helm", "repo", "add", "crossplane-stable", crossplaneHelmRepositoryURL),
		ShellCommand("helm", "repo", "update"),
		ShellCommand(
			"helm",
			"install",
			"crossplane",
			"--namespace", "crossplane-system",
			"--create-namespace", "crossplane-stable/crossplane",
			"--version", crossplaneHelmChartVersion,
		),
		AwaitDeployment("crossplane", "crossplane-system"),
		AwaitDeployment("crossplane-rbac-manager", "crossplane-system"),
	)
}

// TODO can be used as a placeholder for an actual test step.
// It does not check anything, but adds the given message to the test log.
func TODO(message string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Logf("TODO: %s", message)
		return ctx
	}
}

func CheckCompositeResourceLabels(dir, pattern string) features.Func {
	return func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Error(err)
			return ctx
		}
		list := &unstructured.UnstructuredList{}
		for _, o := range rs {
			u := asUnstructured(o)
			claimName := o.GetName()
			claimNamespace := o.GetNamespace()
			claimKind := o.GetObjectKind().GroupVersionKind().Kind
			claimGroupVersionKind := o.GetObjectKind().GroupVersionKind().Group + "/" + o.GetObjectKind().GroupVersionKind().Version
			rg := utils.NewResourceGetter(ctx, t, config)
			claim := rg.Get(claimName, claimNamespace, claimGroupVersionKind, claimKind)
			// Getting XR name
			r := utils.ResourceValue(t, claim, "spec", "resourceRef")
			xRName := r["name"]
			// Define the expected key-value pairs
			expectedLabels := map[string]string{
				"crossplane.io/claim-name":      claimName,
				"crossplane.io/claim-namespace": claimNamespace,
				"crossplane.io/composite":       xRName,
			}
			// Get and compare the labels
			xr := rg.Get(r["name"], "", r["apiVersion"], r["kind"])
			mrefs := utils.ResourceSliceValue(t, xr, "spec", "resourceRefs")
			for _, mref := range mrefs {
				mr := rg.Get(mref["name"], "", mref["apiVersion"], mref["kind"])
				labels := mr.GetLabels()
				for key, expectedValue := range expectedLabels {
					actualValue, ok := labels[key]
					if !ok || actualValue != expectedValue {
						t.Errorf("Label %s doesn't have the expected value", key)
					}
				}
			}
			t.Logf("All Composite Resource labels have the expected values")
			list.Items = append(list.Items, *u)
		}
		return ctx
	}
}

// SecretCreatedWithCredentials checks that the Secret has the same fields as
// the provided Secret in a yaml file and ensures that these fields are not empty.
func SecretCreatedWithCredentials(dir, pattern string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {

		// Read the Secret defined in the yaml file
		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), pattern)
		if err != nil {
			t.Fatalf("Error decoding the yaml file with the Servicebinding Secret.")
			return ctx
		}

		var expectedSecret *unstructured.Unstructured
		var expectedNamespace, expectedName string

		for _, o := range rs {
			u := asUnstructured(o)
			if u.GetKind() == "Secret" {
				expectedSecret = u
				expectedNamespace = u.GetNamespace()
				expectedName = u.GetName()
				break
			}
		}

		if expectedSecret == nil {
			t.Fatalf("No Secret object found in the provided yaml file.")
			return ctx
		}

		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       "Secret",
				"apiVersion": "v1",
			},
		}

		//	Retrieve the Secret from the Kubernetes cluster.
		if err := c.Client().Resources().Get(ctx, expectedName, expectedNamespace, obj); err != nil {
			t.Fatalf("Failed to get Secret: name=%s, namespace=%s", expectedName, expectedNamespace)
			return ctx
		}
		t.Logf("Secret with name %s is found in namespace %s", expectedName, expectedNamespace)

		secretData := obj.Object["data"]

		// Compare the data fields of the existing Secret with the data fields of the expectedSecret.
		if utils.HaveSameFields(secretData.(map[string]interface{}), expectedSecret.Object["stringData"].(map[string]interface{})) {
			t.Logf("Secret in cluster has the same fields as the provided Secret in yaml file.")
			return ctx
		}
		t.Fatalf("No matching Secret found within the specified timeout.")
		return ctx
	}
}

// Check that the value of the specified field matches between the provided yaml
// file and the corresponding object in the Kubernetes cluster
func CheckFieldValueMatch(dir, yamlFileName string, field ...string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {

		rs, err := decoder.DecodeAllFiles(ctx, os.DirFS(dir), yamlFileName)
		if err != nil {
			t.Errorf("Error decoding the yaml file.")
			return ctx
		}
		for _, r := range rs {
			obj := r.(*unstructured.Unstructured)
			claimName := obj.GetName()
			claimNamespace := obj.GetNamespace()
			claimKind := obj.GetObjectKind().GroupVersionKind().Kind
			claimGroupVersionKind := obj.GetObjectKind().GroupVersionKind().Group + "/" + obj.GetObjectKind().GroupVersionKind().Version
			// Get the value of the field from the Kubernetes object.
			claim := utils.NewResourceGetter(ctx, t, c).Get(claimName, claimNamespace, claimGroupVersionKind, claimKind)
			objectField := utils.ResourceStringValue(t, claim, field...)
			// Get the value of the field from the yaml file.
			yamlField, _, err := unstructured.NestedString(obj.Object, field...)
			if err != nil {
				t.Errorf("Error extracting %v from the yaml file: %v\n", field, err)
				return ctx
			}
			if fmt.Sprintf("%v", objectField) != yamlField {
				t.Fatalf("Field %v has different values in yaml file (%v) and in Kubernetes object (%v)", field, yamlField, objectField)
			}
		}
		return ctx
	}
}
