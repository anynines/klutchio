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

package utils

import (
	"reflect"
	"testing"
)

func TestFlattenMap(t *testing.T) {
	type testCase struct {
		name        string
		inputMap    map[string]interface{}
		prefix      string
		expectedMap map[string][]byte
	}

	testCases := []testCase{
		{
			name: "Flattening nested map with default prefix",
			inputMap: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"key3": "value3",
					"key4": map[string]interface{}{
						"key5": "value5",
					},
				},
			},
			prefix: "",
			expectedMap: map[string][]byte{
				"key1":           []byte("value1"),
				"key2.key3":      []byte("value3"),
				"key2.key4.key5": []byte("value5"),
			},
		},
		{
			name: "Flattening nested map with custom prefix",
			inputMap: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"key3": "value3",
					"key4": map[string]interface{}{
						"key5": "value5",
					},
				},
			},
			prefix: "root",
			expectedMap: map[string][]byte{
				"root.key1":           []byte("value1"),
				"root.key2.key3":      []byte("value3"),
				"root.key2.key4.key5": []byte("value5"),
			},
		},
		{
			name:        "Flattening empty map",
			inputMap:    map[string]interface{}{},
			prefix:      "",
			expectedMap: map[string][]byte{},
		},
		{
			name:        "Flattening map with nil value",
			inputMap:    map[string]interface{}{"key1": nil},
			prefix:      "",
			expectedMap: map[string][]byte{"key1": []byte("<nil>")},
		},
		{
			name:        "Flattening map with different data types",
			inputMap:    map[string]interface{}{"key1": "value1", "key2": 10, "key3": true},
			prefix:      "",
			expectedMap: map[string][]byte{"key1": []byte("value1"), "key2": []byte("10"), "key3": []byte("true")},
		},
		{
			name: "Flattening nested map with invalid characters in keys",
			inputMap: map[string]interface{}{
				"key+1": "value1",
				"ke^y2@": map[string]interface{}{
					"ke%%y+&3": "value3",
					"key4": map[string]interface{}{
						"key5": "value5",
					},
				},
			},
			prefix: "",
			expectedMap: map[string][]byte{
				"key_1":            []byte("value1"),
				"ke_y2_.ke__y__3":  []byte("value3"),
				"ke_y2_.key4.key5": []byte("value5"),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualMap := FlattenMap(testCase.inputMap, testCase.prefix)
			if !reflect.DeepEqual(actualMap, testCase.expectedMap) {
				t.Errorf("Expected flattened map to be %v, but got %v", testCase.expectedMap, actualMap)
			}
		})
	}
}

func TestReplaceRootKeyWithNestedKey(t *testing.T) {
	type testCase struct {
		name        string
		inputMap    map[string][]byte
		expectedMap map[string][]byte
	}

	testCases := []testCase{
		{
			name: "Replacing root key with nested key when root key value matches",
			inputMap: map[string][]byte{
				"key1":      []byte("value1"),
				"key2.key1": []byte("value1"),
				"key2.key3": []byte("value3"),
			},
			expectedMap: map[string][]byte{
				"key2.key1": []byte("value1"),
				"key2.key3": []byte("value3"),
			},
		},
		{
			name: "Preserving root key when root key value differs",
			inputMap: map[string][]byte{
				"key1":      []byte("value1"),
				"key2.key1": []byte("value21"),
				"key3":      []byte("value3"),
			},
			expectedMap: map[string][]byte{
				"key1":      []byte("value1"),
				"key2.key1": []byte("value21"),
				"key3":      []byte("value3"),
			},
		},
		{
			name: "Handling nested keys with duplicate root key values",
			inputMap: map[string][]byte{
				"key1":           []byte("value1"),
				"key2.key3":      []byte("value3"),
				"key2.key4.key1": []byte("value1"),
				"key2.key4":      []byte("value4"),
			},
			expectedMap: map[string][]byte{
				"key2.key4.key1": []byte("value1"),
				"key2.key3":      []byte("value3"),
				"key2.key4":      []byte("value4"),
			},
		},
		{
			name:        "Handling empty map",
			inputMap:    map[string][]byte{},
			expectedMap: map[string][]byte{},
		},
		{
			name: "Handling map with nil value",
			inputMap: map[string][]byte{
				"key1": []byte(nil),
			},
			expectedMap: map[string][]byte{
				"key1": []byte(nil),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			inputMapCopy := make(map[string][]byte)
			for key, value := range testCase.inputMap {
				inputMapCopy[key] = value
			}
			ReplaceRootKeyWithNestedKey(inputMapCopy)
			if !reflect.DeepEqual(inputMapCopy, testCase.expectedMap) {
				t.Errorf("Expected flat map to be %v, but got %v", testCase.expectedMap, inputMapCopy)
			}
		})
	}
}
