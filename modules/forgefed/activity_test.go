// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

func Test_NewForgeLike(t *testing.T) {
	want := []byte(`{"type":"Like","startTime":"2024-03-07T00:00:00Z","actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1","object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}`)

	actorIRI := "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"
	objectIRI := "https://codeberg.org/api/v1/activitypub/repository-id/1"
	startTime, _ := time.Parse("2006-Jan-02", "2024-Mar-07")
	sut, err := NewForgeLike(actorIRI, objectIRI, startTime)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}
	if valid, _ := validation.IsValid(sut); !valid {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}

	got, err := sut.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() error = \"%v\"", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MarshalJSON() got = %q, want %q", got, want)
	}
}

func Test_NewForgeUndoLike(t *testing.T) {
	actorIRI := "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"
	objectIRI := "https://codeberg.org/api/v1/activitypub/repository-id/1"
	want := []byte(`{"type":"Undo","startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":{` +
		`"type":"Like",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`)

	startTime, _ := time.Parse("2006-Jan-02", "2024-Mar-27")
	sut, err := NewForgeUndoLike(actorIRI, objectIRI, startTime)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}
	if valid, _ := validation.IsValid(sut); !valid {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}

	got, err := sut.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() error = \"%v\"", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MarshalJSON() got = %q, want %q", got, want)
	}
}

func Test_LikeMarshalJSON(t *testing.T) {
	type testPair struct {
		item    ForgeLike
		want    []byte
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: ForgeLike{},
			want: nil,
		},
		"with ID": {
			item: ForgeLike{
				Activity: ap.Activity{
					Actor:  ap.IRI("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"),
					Type:   "Like",
					Object: ap.IRI("https://codeberg.org/api/v1/activitypub/repository-id/1"),
				},
			},
			want: []byte(`{"type":"Like","actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1","object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.item.MarshalJSON()
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("MarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_LikeUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    *ForgeLike
		wantErr error
	}

	//revive:disable
	tests := map[string]testPair{
		"with ID": {
			item: []byte(`{"type":"Like","actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1","object":"https://codeberg.org/api/activitypub/repository-id/1"}`),
			want: &ForgeLike{
				Activity: ap.Activity{
					Actor:  ap.IRI("https://repo.prod.meissa.de/api/activitypub/user-id/1"),
					Type:   "Like",
					Object: ap.IRI("https://codeberg.org/api/activitypub/repository-id/1"),
				},
			},
			wantErr: nil,
		},
		"invalid": {
			item:    []byte(`{"type":"Invalid","actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1","object":"https://codeberg.org/api/activitypub/repository-id/1"`),
			want:    &ForgeLike{},
			wantErr: fmt.Errorf("cannot parse JSON:"),
		},
	}
	//revive:enable

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := new(ForgeLike)
			err := got.UnmarshalJSON(test.item)
			if (err != nil || test.wantErr != nil) && !strings.Contains(err.Error(), test.wantErr.Error()) {
				t.Errorf("UnmarshalJSON() error = \"%v\", wantErr \"%v\"", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalJSON() got = %q, want %q, err %q", got, test.want, err.Error())
			}
		})
	}
}

func Test_UndoLikeMarshalJSON(t *testing.T) {
	type testPair struct {
		item    ForgeUndoLike
		want    []byte
		wantErr error
	}

	startTime, _ := time.Parse("2006-Jan-02", "2024-Mar-27")
	like, _ := NewForgeLike("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1", "https://codeberg.org/api/v1/activitypub/repository-id/1", startTime)
	tests := map[string]testPair{
		"empty": {
			item: ForgeUndoLike{},
			want: nil,
		},
		"valid": {
			item: ForgeUndoLike{
				Activity: ap.Activity{
					StartTime: startTime,
					Actor:     ap.IRI("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"),
					Type:      "Undo",
					Object:    like,
				},
			},
			want: []byte(`{"type":"Undo",` +
				`"startTime":"2024-03-27T00:00:00Z",` +
				`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
				`"object":{` +
				`"type":"Like",` +
				`"startTime":"2024-03-27T00:00:00Z",` +
				`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
				`"object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.item.MarshalJSON()
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("MarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %q\nwant %q", got, tt.want)
			}
		})
	}
}

func Test_UndoLikeUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    *ForgeUndoLike
		wantErr error
	}

	startTime, _ := time.Parse("2006-Jan-02", "2024-Mar-27")
	like, _ := NewForgeLike("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1", "https://codeberg.org/api/v1/activitypub/repository-id/1", startTime)

	//revive:disable
	tests := map[string]testPair{
		"valid": {
			item: []byte(`{"type":"Undo",` +
				`"startTime":"2024-03-27T00:00:00Z",` +
				`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
				`"object":{` +
				`"type":"Like",` +
				`"startTime":"2024-03-27T00:00:00Z",` +
				`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
				`"object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`),
			want: &ForgeUndoLike{
				Activity: ap.Activity{
					StartTime: startTime,
					Actor:     ap.IRI("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"),
					Type:      "Undo",
					Object:    like,
				},
			},
			wantErr: nil,
		},
		"invalid": {
			item:    []byte(`invalid JSON`),
			want:    nil,
			wantErr: fmt.Errorf("cannot parse JSON:"),
		},
	}
	//revive:enable

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := new(ForgeUndoLike)
			err := got.UnmarshalJSON(test.item)
			if test.wantErr != nil {
				if err == nil {
					t.Errorf("UnmarshalJSON() error = nil, wantErr \"%v\"", test.wantErr)
				} else {
					if !strings.Contains(err.Error(), test.wantErr.Error()) {
						t.Errorf("UnmarshalJSON() error = \"%v\", wantErr \"%v\"", err, test.wantErr)
					}
				}
				return
			}
			// remarshalling due to problems of DeepEqual for struct ForgeUndoLike
			remarshalled_got, _ := got.MarshalJSON()
			remarshalled_want, _ := test.want.MarshalJSON()
			if !reflect.DeepEqual(remarshalled_got, remarshalled_want) {
				t.Errorf("UnmarshalJSON() got = %#v\nwant %#v", got, test.want)
			}
		})
	}
}

func getValidationError(subject validation.Validateable, expectedError string) *string {
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

func Test_ForgeLikeValidation(t *testing.T) {

	// Successful

	sut := new(ForgeLike)
	sut.UnmarshalJSON([]byte(`{"type":"Like",
	"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if res, _ := validation.IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}

	// Errors

	sut.UnmarshalJSON([]byte(`{"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if err := getValidationError(sut, "type should not be empty"); err != nil {
		t.Error(err)
	}

	sut.UnmarshalJSON([]byte(`{"type":"bad-type",
		"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if err := getValidationError(sut, "Value bad-type is not contained in allowed values [Like]"); err != nil {
		t.Error(err)
	}

	sut.UnmarshalJSON([]byte(`{"type":"Like",
		"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "not a date"}`))
	if err := getValidationError(sut, "StartTime was invalid."); err != nil {
		t.Error(err)
	}

	sut.UnmarshalJSON([]byte(`{"type":"Wrong",
		"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if err := getValidationError(sut, "Value Wrong is not contained in allowed values [Like]"); err != nil {
		t.Error(err)
	}
}

func TestActivityValidationUndo(t *testing.T) {
	sut := new(ForgeUndoLike)

	// Success

	_ = sut.UnmarshalJSON([]byte(`{"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":{` +
		`  "type":"Like",` +
		`  "actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`  "object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`))
	if res, _ := validation.IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}

	// Errors

	_ = sut.UnmarshalJSON([]byte(`{"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":{` +
		`  "type":"Like",` +
		`  "startTime":"2024-03-27T00:00:00Z",` +
		`  "actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`  "object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`))
	if err := getValidationError(sut, "type should not be empty"); err != nil {
		t.Error(*err)
	}

	_ = sut.UnmarshalJSON([]byte(`{"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"object":{` +
		`  "type":"Like",` +
		`  "startTime":"2024-03-27T00:00:00Z",` +
		`  "actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`  "object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`))
	if err := getValidationError(sut, "Actor should not be nil."); err != nil {
		t.Error(*err)
	}

	_ = sut.UnmarshalJSON([]byte(`{"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"string",` +
		`"object":{` +
		`  "type":"Like",` +
		`  "startTime":"2024-03-27T00:00:00Z",` +
		`  "actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`  "object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`))
	if err := getValidationError(sut, "Actor should not be nil."); err != nil {
		t.Error(*err)
	}

	_ = sut.UnmarshalJSON([]byte(`{` +
		`"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"` +
		`}`))
	if err := getValidationError(sut, "object should not be empty."); err != nil {
		t.Error(*err)
	}

	_ = sut.UnmarshalJSON([]byte(`{"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":{` +
		`  "startTime":"2024-03-27T00:00:00Z",` +
		`  "actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`  "object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}}`))
	// Without type object is not parsed as Activity
	if err := getValidationError(sut, "object is not of type Activity"); err != nil {
		t.Error(*err)
	}

	_ = sut.UnmarshalJSON([]byte(`{"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":{` +
		`  "type":"Like",` +
		`  "object":""}}`))
	if err := getValidationError(sut, "Object.Actor should not be nil."); err != nil {
		t.Error(*err)
	}

	_ = sut.UnmarshalJSON([]byte(`{"type":"Undo",` +
		`"startTime":"2024-03-27T00:00:00Z",` +
		`"actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1",` +
		`"object":{` +
		`  "type":"Like",` +
		`  "startTime":"2024-03-27T00:00:00Z",` +
		`  "actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"}}`))
	if err := getValidationError(sut, "Object.Object should not be nil."); err != nil {
		t.Error(*err)
	}
}

func TestActivityValidation_Attack(t *testing.T) {
	sut := new(ForgeLike)
	sut.UnmarshalJSON([]byte(`{rubbish}`))
	if len(sut.Validate()) != 5 {
		t.Errorf("5 validation errors expected but was: %v\n", len(sut.Validate()))
	}
}
