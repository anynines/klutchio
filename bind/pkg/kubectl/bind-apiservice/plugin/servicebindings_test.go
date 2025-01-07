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

package plugin

import (
	"bytes"
	"os"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

func TestHumanReadablePromt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		testData       bindv1alpha1.PermissionClaim
		expectedOutput string
	}{
		{"Owner=Provider",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,Required=false",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: false,
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is optional.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,Selector.Names={foo}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar"},
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,GroupResource.Group",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "example.com",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
			},
			"The provider wants to write foo objects (apiVersion: \"example.com/v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,Selector.Names={bar},GroupResource.Group",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "example.com",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar"},
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
			},
			"The provider wants to write foo objects (apiVersion: \"example.com/v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,CreateOptions={}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Create:   &bindv1alpha1.CreateOptions{},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,AutoDonate=false",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required:   true,
				AutoDonate: false,
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,AutoDonate=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required:   true,
				AutoDonate: true,
			},
			"The provider wants to create user owned foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,OnConflict={}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required:   true,
				OnConflict: &bindv1alpha1.OnConflictOptions{},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,Create.ReplaceExisting=false",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Create: &bindv1alpha1.CreateOptions{
					ReplaceExisting: false,
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,Create.ReplaceExisting=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Create: &bindv1alpha1.CreateOptions{
					ReplaceExisting: true,
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Conflicting objects will be replaced by the provider. " +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,OnConflict.RecreateWhenConsumerSideDeleted=false",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				OnConflict: &bindv1alpha1.OnConflictOptions{
					RecreateWhenConsumerSideDeleted: false,
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,OnConflict.RecreateWhenConsumerSideDeleted=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				OnConflict: &bindv1alpha1.OnConflictOptions{
					RecreateWhenConsumerSideDeleted: true,
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Created objects will be recreated upon deletion. " +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,UpdateOptions={}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Update:   &bindv1alpha1.UpdateOptions{},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,UpdateOptions.Fields",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Update: &bindv1alpha1.UpdateOptions{
					Fields: []string{"foo", "bar"},
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will still be able to be changed by the provider:\n" + // TODO
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,UpdateOptions.Preserving",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Update: &bindv1alpha1.UpdateOptions{
					Preserving: []string{"foo", "bar"},
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will be preserved by the provider:\n" +
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,UpdateOptions.AlwaysRecreate=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
				Update: &bindv1alpha1.UpdateOptions{
					AlwaysRecreate: true,
				},
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Modification of said objects will by handled by deletion and recreation of said objects.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,UpdateOptions.Fields,AutoDonate=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required:   true,
				AutoDonate: true,
				Update: &bindv1alpha1.UpdateOptions{
					Fields: []string{"foo", "bar"},
				},
			},
			"The provider wants to create user owned foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will still be able to be changed by the provider:\n" +
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,UpdateOptions.Preserving,AutoDonate=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Provider,
				},
				Required:   true,
				AutoDonate: true,
				Update: &bindv1alpha1.UpdateOptions{
					Preserving: []string{"foo", "bar"},
				},
			},
			"The provider wants to create user owned foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will be preserved by the provider:\n" +
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,Selector.Names={bar}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar"},
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,GroupResource.Group",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "example.com",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
			},
			"The provider wants to read foo objects (apiVersion: \"example.com/v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,Selector.Names={bar},GroupResource.Group",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "example.com",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar"},
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
			},
			"The provider wants to read foo objects (apiVersion: \"example.com/v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,Adopt=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				AutoAdopt: true,
				Required:  true,
			},
			"The provider wants to have ownership of foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,Selector.Names={bar},Adopt=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar"},
					Owner: bindv1alpha1.Consumer,
				},
				AutoAdopt: true,
				Required:  true,
			},
			"The provider wants to have ownership of foo objects (apiVersion: \"v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,OnConflict={}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required:   true,
				OnConflict: &bindv1alpha1.OnConflictOptions{},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,Create.ReplaceExisting=false",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
				Create: &bindv1alpha1.CreateOptions{
					ReplaceExisting: false,
				},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,Create.ReplaceExisting=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
				Create: &bindv1alpha1.CreateOptions{
					ReplaceExisting: true,
				},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Conflicting objects will be replaced by the provider. " +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,UpdateOptions={}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
				Update:   &bindv1alpha1.UpdateOptions{},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,UpdateOptions.Fields",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
				Update: &bindv1alpha1.UpdateOptions{
					Fields: []string{"foo", "bar"},
				},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will still be able to be changed by the provider:\n" +
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,UpdateOptions.Preserving",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
				Update: &bindv1alpha1.UpdateOptions{
					Preserving: []string{"foo", "bar"},
				},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will be preserved by the provider:\n" +
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,UpdateOptions.AlwaysRecreate=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required: true,
				Update: &bindv1alpha1.UpdateOptions{
					AlwaysRecreate: true,
				},
			},
			"The provider wants to read foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Modification of said objects will by handled by deletion and recreation of said objects.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,UpdateOptions.Fields,Adopt=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required:  true,
				AutoAdopt: true,
				Update: &bindv1alpha1.UpdateOptions{
					Fields: []string{"foo", "bar"},
				},
			},
			"The provider wants to have ownership of foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will still be able to be changed by the provider:\n" +
				"\t\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Consumer,UpdateOptions.Preserving,Adopt=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Owner: bindv1alpha1.Consumer,
				},
				Required:  true,
				AutoAdopt: true,
				Update: &bindv1alpha1.UpdateOptions{
					Preserving: []string{"foo", "bar"},
				},
			},
			"The provider wants to have ownership of foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will be preserved by the provider:\n" + "	\"foo\"\n" +
				"\t\"bar\"\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Selector={}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version:  "v1",
				Selector: &bindv1alpha1.ResourceSelector{},
				Required: true,
			},
			"The provider wants to read and write foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Selector.Owner=\"\",Selector.Names={bar}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar"},
				},
				Required: true,
			},
			"The provider wants to read and write foo objects (apiVersion: \"v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Selector={},AutoDonate=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version:    "v1",
				Selector:   &bindv1alpha1.ResourceSelector{},
				Required:   true,
				AutoDonate: true,
			},
			"The provider wants to create user owned foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Selector={},AutoDonate=true,update.Fields=[\"spec\"]",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version:    "v1",
				Selector:   &bindv1alpha1.ResourceSelector{},
				AutoDonate: true,
				Update: &bindv1alpha1.UpdateOptions{
					Fields: []string{"spec"},
				},
			},
			"The provider wants to create user owned foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will still be able to be changed by the provider:\n" +
				"\t\"spec\"\n" +
				"Accepting this Permission is optional.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Selector={},adopt=true",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version:   "v1",
				Selector:  &bindv1alpha1.ResourceSelector{},
				Required:  true,
				AutoAdopt: true,
			},
			"The provider wants to have ownership of foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Selector={},adopt=true,update.Fields=[\"spec\"]",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version:   "v1",
				Selector:  &bindv1alpha1.ResourceSelector{},
				AutoAdopt: true,
				Update: &bindv1alpha1.UpdateOptions{
					Fields: []string{"spec"},
				},
			},
			"The provider wants to have ownership of foo objects (apiVersion: \"v1\") on your cluster.\n" +
				"The following fields of the objects will still be able to be changed by the provider:\n" +
				"\t\"spec\"\n" +
				"Accepting this Permission is optional.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
		{"Owner=Provider,Selector.Names={bar,baz}",
			bindv1alpha1.PermissionClaim{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "",
					Resource: "foo",
				},
				Version: "v1",
				Selector: &bindv1alpha1.ResourceSelector{
					Names: []string{"bar", "baz"},
					Owner: bindv1alpha1.Provider,
				},
				Required: true,
			},
			"The provider wants to write foo objects (apiVersion: \"v1\") which are referenced with:\n" +
				"\t- name: \"bar\"\n" +
				"\t- name: \"baz\"\n" +
				"on your cluster.\n" +
				"Accepting this Permission is required in order to proceed.\n" +
				"Do you accept this Permission? [No,Yes]\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			var input bytes.Buffer
			input.WriteString("y\n")
			opts := NewBindAPIServiceOptions(genericclioptions.IOStreams{In: &input, Out: &output, ErrOut: os.Stderr})
			b, err := opts.promptYesNo(tt.testData)
			if output.String() != tt.expectedOutput {
				t.Errorf("Expected IO Output did not match. got: \"\n%s\"\nwanted: \"\n%s\"\n", output.String(), tt.expectedOutput)
			}
			if b == false || (err != nil) {
				t.Errorf("Expected Return value did not match. got: \"%v\", \"%v\"", b, err)
			}
		})
	}
}
