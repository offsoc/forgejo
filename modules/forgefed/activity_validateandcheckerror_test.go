// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"

	"forgejo.org/modules/validation"
)

func validateAndCheckError(subject validation.Validateable, expectedError string) *string {
	errors := subject.Validate()
	err := errors[0]
	if len(errors) < 1 {
		val := "Validation error should have been returned, but was not."
		return &val
	} else if err != expectedError {
		val := fmt.Sprintf("Validation error should be [%v] but was: %v\n", expectedError, err)
		return &val
	}
	return nil
}
