package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"
)

func TestLandingPageDetailsDefault(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/")
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	htmlDoc.AssertElement(t, "a[href='https://forgejo.org/download/#installation-from-binary']", true)
}

func TestLandingPageDetailsTrue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.LandingPageDetails, true)()

	req := NewRequest(t, "GET", "/")
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	htmlDoc.AssertElement(t, "a[href='https://forgejo.org/download/#installation-from-binary']", true)
}

func TestLandingPageDetailsFalse(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.LandingPageDetails, false)()

	req := NewRequest(t, "GET", "/")
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	htmlDoc.AssertElement(t, "a[href='https://forgejo.org/download/#installation-from-binary']", false)
}
