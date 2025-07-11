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

import (
	"reflect"
	"testing"
)

func TestParameterUpdateForBroker(t *testing.T) {
	tests := []struct {
		name     string
		observed map[string]interface{}
		expected map[string]interface{}
		want     map[string]interface{}
	}{
		{
			name:     "no change",
			observed: map[string]interface{}{"a": 1},
			expected: map[string]interface{}{"a": 1},
			want:     nil,
		},
		{
			name:     "key removed",
			observed: map[string]interface{}{"a": 1, "b": 2},
			expected: map[string]interface{}{"a": 1},
			want:     map[string]interface{}{"b": nil},
		},
		{
			name:     "key added",
			observed: map[string]interface{}{"a": 1},
			expected: map[string]interface{}{"a": 1, "b": 2},
			want:     map[string]interface{}{"b": 2},
		},
		{
			name:     "value modified",
			observed: map[string]interface{}{"a": 1},
			expected: map[string]interface{}{"a": 2},
			want:     map[string]interface{}{"a": 2},
		},
		{
			name:     "slice order irrelevant",
			observed: map[string]interface{}{"list": []interface{}{1, 2, 3}},
			expected: map[string]interface{}{"list": []interface{}{3, 2, 1}},
			want:     nil,
		},
		{
			name:     "slice value changed",
			observed: map[string]interface{}{"list": []interface{}{1, 2, 3}},
			expected: map[string]interface{}{"list": []interface{}{1, 2, 4}},
			want:     map[string]interface{}{"list": []interface{}{1, 2, 4}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParameterUpdateForBroker(tc.observed, tc.expected)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
