/*
Copyright 2023 The Kube Bind Authors.

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

package exporttemplate

// TODO by namespace
// TODO cached client
// TODO find a better name or reorganize into different packages
import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crd "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/anynines/klutchio/bind/contrib/example-backend/apis/examplebackend/v1alpha1"
	templates "github.com/anynines/klutchio/bind/contrib/example-backend/client/clientset/versioned"
)

type Index struct {
	templates templates.Interface
	crds      crd.Interface
	clusterNs string
}

func NewCatalogue(r *rest.Config) Index {
	crdClient := crd.NewForConfigOrDie(r)

	templateClient := templates.NewForConfigOrDie(r)

	return Index{
		templates: templateClient,
		crds:      crdClient,
	}
}

func (i Index) GetExported(ctx context.Context) ([]apiextensionsv1.CustomResourceDefinition, error) {
	list, err := i.crds.ApiextensionsV1().CustomResourceDefinitions().List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	exports, err := i.templates.ExampleBackendV1alpha1().APIServiceExportTemplates(i.clusterNs).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	exported := []apiextensionsv1.CustomResourceDefinition{}

	for _, ex := range exports.Items {
		for _, c := range list.Items {
			s := ex.Spec.APIServiceSelector
			if s.Group == c.Spec.Group && s.Resource == c.Spec.Names.Plural {
				exported = append(exported, c)
			}
		}
	}

	if exported == nil {
		return nil, fmt.Errorf("no exported resources")
	}

	return exported, nil
}

func (i Index) TemplateFor(ctx context.Context, group, resource string) (v1alpha1.APIServiceExportTemplate, error) {
	exports, err := i.templates.ExampleBackendV1alpha1().APIServiceExportTemplates(i.clusterNs).List(ctx, v1.ListOptions{})
	if err != nil {
		return v1alpha1.APIServiceExportTemplate{}, nil
	}

	for _, e := range exports.Items {
		if e.Spec.APIServiceSelector.Resource == resource && e.Spec.APIServiceSelector.Group == group {
			return e, nil
		}
	}

	return v1alpha1.APIServiceExportTemplate{}, fmt.Errorf("not found: %s/%s", group, resource)
}
