// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// ----------------------------- ActorID --------------------------------------------
type ActorID struct {
	ID               string
	Source           string
	HostSchema       string
	Path             string
	Host             string
	HostPort         string
	UnvalidatedInput string
}

// Factory function for ActorID. Created struct is asserted to be valid
func NewActorID(uri string) (ActorID, error) {
	result, err := newActorID(uri)
	if err != nil {
		return ActorID{}, err
	}

	if valid, err := validation.IsValid(result); !valid {
		return ActorID{}, err
	}

	return result, nil
}

//Todo: add id.HostPort in case of given https://an.other.host:443/api/v1/activitypub/user-id/1
func (id ActorID) AsURI() string {
	return fmt.Sprintf("%s://%s/%s/%s", id.HostSchema, id.Host, id.Path, id.ID)
}

func (id ActorID) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(id.ID, "userId")...)
	result = append(result, validation.ValidateNotEmpty(id.HostSchema, "schema")...)
	result = append(result, validation.ValidateNotEmpty(id.Path, "path")...)
	result = append(result, validation.ValidateNotEmpty(id.Host, "host")...)
	result = append(result, validation.ValidateNotEmpty(id.HostPort, "HostPort")...)
	result = append(result, validation.ValidateNotEmpty(id.UnvalidatedInput, "unvalidatedInput")...)

	fmt.Println("id.UnvalidatedInput: ", id.UnvalidatedInput)
	fmt.Println("id.AsURI: ", id.AsURI())

	//add or 
	if id.UnvalidatedInput != id.AsURI() || id.HostPort != "443" {
		result = append(result, fmt.Sprintf("not all input was parsed, \nUnvalidated Input:%q \nParsed URI: %q", id.UnvalidatedInput, id.AsURI()))
	}
	return result
}

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
	result = append(result, validation.ValidateOneOf(id.Source, []any{"forgejo", "gitea"}, "Source")...)
	result = append(result, validation.ValidateOneOf(id.HostSchema, []any{"https", "http", "HTTPS", "HTTP"}, "Source")...)

	switch id.Source {
	case "forgejo", "gitea":
		if strings.ToLower(id.Path) != "api/v1/activitypub/user-id" && strings.ToLower(id.Path) != "api/activitypub/user-id" {
			result = append(result, fmt.Sprintf("path: %q has to be a person specific api path", id.Path))
		}
	}
	switch id.HostSchema {
	case "HTTPS", "HTTP":
		if strings.ToLower(id.HostSchema) == "https" {
			result = append(result, fmt.Sprintf("-%s", strings.ToLower(id.HostSchema)))
		} else if strings.ToLower(id.HostSchema) == "http" {
			result = append(result, fmt.Sprintf("-%s", strings.ToLower(id.HostSchema)))
		}
	}
	return result
}

// ----------------------------- RepositoryID --------------------------------------------

type RepositoryID struct {
	ActorID
}

// Factory function for RepositoryID. Created struct is asserted to be valid.
func NewRepositoryID(uri, source string) (RepositoryID, error) {
	result, err := newActorID(uri)
	if err != nil {
		return RepositoryID{}, err
	}
	result.Source = source

	// validate Person specific
	repoID := RepositoryID{result}
	if valid, err := validation.IsValid(repoID); !valid {
		return RepositoryID{}, err
	}

	return repoID, nil
}

func (id RepositoryID) Validate() []string {
	result := id.ActorID.Validate()
	result = append(result, validation.ValidateNotEmpty(id.Source, "source")...)
	result = append(result, validation.ValidateOneOf(id.Source, []any{"forgejo", "gitea"}, "Source")...)
	switch id.Source {
	case "forgejo", "gitea":
		if strings.ToLower(id.Path) != "api/v1/activitypub/repository-id" && strings.ToLower(id.Path) != "api/activitypub/repository-id" {
			result = append(result, fmt.Sprintf("path: %q has to be a repo specific api path", id.Path))
		}
	}
	return result
}

func containsEmptyString(ar []string) bool {
	for _, elem := range ar {
		if elem == "" {
			return true
		}
	}
	return false
}

func removeEmptyStrings(ls []string) []string {
	var rs []string
	for _, str := range ls {
		if str != "" {
			rs = append(rs, str)
		}
	}
	return rs
}

func newActorID(uri string) (ActorID, error) {
	validatedURI, err := url.ParseRequestURI(uri)
	
	if err != nil {
		return ActorID{}, err
	}
	pathWithActorID := strings.Split(validatedURI.Path, "/")
	if containsEmptyString(pathWithActorID) {
		pathWithActorID = removeEmptyStrings(pathWithActorID)
	}
	length := len(pathWithActorID)
	pathWithoutActorID := strings.Join(pathWithActorID[0:length-1], "/")
	id := pathWithActorID[length-1]
	
	result := ActorID{}
	result.ID = id
	result.HostSchema = validatedURI.Scheme
	result.Host = validatedURI.Hostname()
	result.Path = pathWithoutActorID
	
	if validatedURI.Port() == "" && result.HostSchema == "https" {
		result.HostPort = "443"
		result.UnvalidatedInput = fmt.Sprintf("%s://%s/%s/%s", result.HostSchema, result.Host, result.Path, result.ID)
		return result, nil
	} else if validatedURI.Port() == "" && result.HostSchema == "http" {
		result.HostPort = "80"
		result.UnvalidatedInput = fmt.Sprintf("%s://%s/%s/%s", result.HostSchema, result.Host, result.Path, result.ID)
		return result, nil
	}

	result.HostPort = validatedURI.Port()
	result.UnvalidatedInput = uri
	return result, nil
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
