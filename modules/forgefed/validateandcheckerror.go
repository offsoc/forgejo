// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"code.gitea.io/gitea/modules/validation"
)

// Validates the subject and checks if expectedError occurred.
// If expectedError occurred, then nil is returned else a string is returned with detailed information.
func validateAndCheckError(subject validation.Validateable, expectedError string) *string {
	errors := subject.Validate()
	if len(errors) < 1 {
		val := "Validation error should have been returned, but was not."
		return &val
	} else {
		var err = errors[0]
		if err != expectedError {
			val := fmt.Sprintf("Validation error should be [%v] but was: %v\n", expectedError, err)
			return &val
		}
	}
	return nil
}