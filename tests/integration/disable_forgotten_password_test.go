package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestDisableForgottenPasswordTrueTrue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.RequireExternalRegistrationPassword, true)()
	defer test.MockVariableValue(&setting.Service.AllowOnlyExternalRegistration, true)()

	req := NewRequest(t, "GET", "/user/login/")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find("a")
	var counterInstances int = 0
	for i := range doc.Nodes {
		oneElement := doc.Eq(i)
		attValue, attExists := oneElement.Attr("href")
		if attExists {
			if attValue == "/user/forgot_password" {
				counterInstances += 1
			}
		}
	}
	assert.EqualValues(t, 0, counterInstances)
}

func TestDisableForgottenPasswordFalseTrue(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.RequireExternalRegistrationPassword, false)()
	defer test.MockVariableValue(&setting.Service.AllowOnlyExternalRegistration, true)()

	req := NewRequest(t, "GET", "/user/login/")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find("a")
	var counterInstances int = 0
	for i := range doc.Nodes {
		oneElement := doc.Eq(i)
		attValue, attExists := oneElement.Attr("href")
		if attExists {
			if attValue == "/user/forgot_password" {
				counterInstances += 1
			}
		}
	}
	assert.EqualValues(t, 0, counterInstances)
}

func TestDisableForgottenPasswordTrueFalse(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.RequireExternalRegistrationPassword, true)()
	defer test.MockVariableValue(&setting.Service.AllowOnlyExternalRegistration, false)()

	req := NewRequest(t, "GET", "/user/login/")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find("a")
	var counterInstances int = 0
	for i := range doc.Nodes {
		oneElement := doc.Eq(i)
		attValue, attExists := oneElement.Attr("href")
		if attExists {
			if attValue == "/user/forgot_password" {
				counterInstances += 1
			}
		}
	}
	assert.EqualValues(t, 0, counterInstances)
}

func TestDisableForgottenPasswordDefault(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user/login/")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find("a")
	var counterInstances int = 0
	for i := range doc.Nodes {
		oneElement := doc.Eq(i)
		attValue, attExists := oneElement.Attr("href")
		if attExists {
			if attValue == "/user/forgot_password" {
				counterInstances += 1
			}
		}
	}
	assert.EqualValues(t, 1, counterInstances)
}

func TestDisableForgottenPasswordFalseFalse(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.RequireExternalRegistrationPassword, false)()
	defer test.MockVariableValue(&setting.Service.AllowOnlyExternalRegistration, false)()

	req := NewRequest(t, "GET", "/user/login/")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find("a")
	var counterInstances int = 0
	for i := range doc.Nodes {
		oneElement := doc.Eq(i)
		attValue, attExists := oneElement.Attr("href")
		if attExists {
			if attValue == "/user/forgot_password" {
				counterInstances += 1
			}
		}
	}
	assert.EqualValues(t, 1, counterInstances)
}
