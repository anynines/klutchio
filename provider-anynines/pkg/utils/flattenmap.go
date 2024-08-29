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
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Converts a nested map into a flat map.
func FlattenMap(inputMap map[string]interface{}, prefix string) map[string][]byte {
	flatMap := make(map[string][]byte)

	for key, value := range inputMap {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}
		newKey = generateValidKey(newKey)
		if v, ok := value.(map[string]interface{}); ok {
			// If the value is a nested map, recursively flatten it
			nestedMap := FlattenMap(v, newKey)
			for k, v := range nestedMap {
				flatMap[k] = v
			}
		} else {
			// If the value is not a map, add it to the flattened map as []byte
			flatMap[newKey] = []byte(fmt.Sprint(value))
		}
	}
	return flatMap
}

// Replaces root keys with the corresponding nested keys if there are duplicate keys with the same value.
func ReplaceRootKeyWithNestedKey(flatResult map[string][]byte) {
	for key := range flatResult {
		parts := strings.Split(key, ".")
		if len(parts) > 1 {
			rootKey := parts[len(parts)-1]
			if rootValue, ok := flatResult[rootKey]; ok && bytes.Equal(rootValue, flatResult[key]) {
				delete(flatResult, rootKey)
			}
		}
	}
}

// Replace any invalid character with underscores
func generateValidKey(s string) string {
	regex := regexp.MustCompile(`[^-._a-zA-Z0-9]`)
	return regex.ReplaceAllString(s, "_")
}
