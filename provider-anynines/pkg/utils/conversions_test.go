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

import "testing"

func TestCamelCaseToUnderscore(t *testing.T) {
	examples := map[string]string{
		"helloWorld":                           "hello_world",
		"longTermWithAFewMoreUpperCaseLetters": "long_term_with_a_few_more_upper_case_letters",
		"nouppercaseletters":                   "nouppercaseletters",
	}

	for input, expected := range examples {
		actual := CamelCaseToUnderscore(input)
		if actual != expected {
			t.Fatalf("Expected %q, got %q", expected, actual)
		}
	}
}

func TestUnderscoreToCamelCase(t *testing.T) {
	examples := map[string]string{
		"hello_world": "helloWorld",
		"long_term_with_a_few_more_upper_case_letters": "longTermWithAFewMoreUpperCaseLetters",
		"nouppercaseletters":                           "nouppercaseletters",
	}

	for input, expected := range examples {
		actual := UnderscoreToCamelCase(input)
		if actual != expected {
			t.Fatalf("Expected %q, got %q", expected, actual)
		}
	}
}
