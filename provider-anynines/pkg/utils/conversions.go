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
	"regexp"
	"strings"
)

var camelToUnderscoreRE = regexp.MustCompile("[A-Z]")
var underscoreToCamelRE = regexp.MustCompile("_[a-z]")

func CamelCaseToUnderscore(s string) string {
	return camelToUnderscoreRE.ReplaceAllStringFunc(s, func(match string) string {
		return "_" + strings.ToLower(match)
	})
}

func UnderscoreToCamelCase(s string) string {
	return underscoreToCamelRE.ReplaceAllStringFunc(s, func(match string) string {
		return strings.ToUpper(string(match[1]))
	})
}
