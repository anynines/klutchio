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

package anynines

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

// This package contains utilities used across various provider controllers for interfacing with the
// anynines APIs. Anticipated enhancements to the Backup Manager API may facilitate the supply of
// backup IDs from our controller, providing a solution for handling issues related to leftover
// backups. The UUID takes the format suggested by the Anynines team. It has the added benefit
// of reducing the probability of UUID collisions.

// GenUID generates a unique identifier by combining a UUID with the current Unix timestamp.
func GenUID() string {
	return uuid.New().String() + "-" + unixTimestamp()
}

// unixTimestamp returns the current time represented as a Unix timestamp in string format.
func unixTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}
