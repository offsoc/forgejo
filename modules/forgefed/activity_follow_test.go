package forgefed

import (
	"testing"

	"forgejo.org/modules/validation"
	ap "github.com/go-ap/activitypub"
)

func Test_NewForgeFollowValidation(t *testing.T) {
	sut := ForgeFollow{}
	sut.Type = "Follow"
	sut.Actor = ap.IRI("example.org/alice")
	sut.Target = ap.IRI("example.org/bob")

	if err, _ := validation.IsValid(sut); !err {
		t.Errorf("sut invalid: %v\n", sut.Validate())
	}
}
