// Copyright 2023, 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"strings"

	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// ----------------------------- PersonID --------------------------------------------
type PersonID struct {
	ActorID
}

// Factory function for PersonID. Created struct is asserted to be valid
func NewPersonID(uri, source string) (PersonID, error) {
	result, err := newActorID(uri)
	if err != nil {
		return PersonID{}, err
	}
	result.Source = source

	// validate Person specific path
	personID := PersonID{result}
	if valid, err := validation.IsValid(personID); !valid {
		return PersonID{}, err
	}

	return personID, nil
}

func (id PersonID) AsWebfinger() string {
	result := fmt.Sprintf("@%s@%s", strings.ToLower(id.ID), strings.ToLower(id.Host))
	return result
}

func (id PersonID) AsLoginName() string {
	result := fmt.Sprintf("%s%s", strings.ToLower(id.ID), id.HostSuffix())
	return result
}

func (id PersonID) HostSuffix() string {
	result := fmt.Sprintf("-%s", strings.ToLower(id.Host))
	return result
}

func (id PersonID) Validate() []string {
	result := id.ActorID.Validate()
	result = append(result, validation.ValidateNotEmpty(id.Source, "source")...)
	result = append(result, validation.ValidateOneOf(id.Source, []any{"forgejo", "gitea", "mastodon", "gotosocial"}, "Source")...)
	switch id.Source {
	case "forgejo", "gitea":
		if strings.ToLower(id.Path) != "api/v1/activitypub/user-id" && strings.ToLower(id.Path) != "api/activitypub/user-id" {
			result = append(result, fmt.Sprintf("path: %q has to be a person specific api path", id.Path))
		}
	}

	return result
}

// ----------------------------- ForgePerson -------------------------------------

// ForgePerson activity data type
// swagger:model
type ForgePerson struct {
	// swagger:ignore
	ap.Actor
}

func (s ForgePerson) MarshalJSON() ([]byte, error) {
	return s.Actor.MarshalJSON()
}

func (s *ForgePerson) UnmarshalJSON(data []byte) error {
	return s.Actor.UnmarshalJSON(data)
}

func (s ForgePerson) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(s.Type), "Type")...)
	result = append(result, validation.ValidateOneOf(string(s.Type), []any{string(ap.PersonType)}, "Type")...)
	result = append(result, validation.ValidateNotEmpty(s.PreferredUsername.String(), "PreferredUsername")...)

	return result
}
