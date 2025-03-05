// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type FederationServerMockPerson struct {
	ID     int64
	Name   string
	PubKey string
}
type FederationServerMockRepository struct {
	ID int64
}
type FederationServerMock struct {
	Persons      []FederationServerMockPerson
	Repositories []FederationServerMockRepository
	LastPost     string
}

func NewFederationServerMockPerson(id int64, name string) FederationServerMockPerson {
	return FederationServerMockPerson{
		ID:   id,
		Name: name,
		PubKey: `"-----BEGIN PUBLIC KEY-----\nMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEA18H5s7N6ItZUAh9tneII\nIuZdTTa3cZlLa/9ejWAHTkcp3WLW+/zbsumlMrWYfBy2/yTm56qasWt38iY4D6ul\n` +
			`CPiwhAqX3REvVq8tM79a2CEqZn9ka6vuXoDgBg/sBf/BUWqf7orkjUXwk/U0Egjf\nk5jcurF4vqf1u+rlAHH37dvSBaDjNj6Qnj4OP12bjfaY/yvs7+jue/eNXFHjzN4E\n` +
			`T2H4B/yeKTJ4UuAwTlLaNbZJul2baLlHelJPAsxiYaziVuV5P+IGWckY6RSerRaZ\nAkc4mmGGtjAyfN9aewe+lNVfwS7ElFx546PlLgdQgjmeSwLX8FWxbPE5A/PmaXCs\n` +
			`nx+nou+3dD7NluULLtdd7K+2x02trObKXCAzmi5/Dc+yKTzpFqEz+hLNCz7TImP/\ncK//NV9Q+X67J9O27baH9R9ZF4zMw8rv2Pg0WLSw1z7lLXwlgIsDapeMCsrxkVO4\n` +
			`LXX5AQ1xQNtlssnVoUBqBrvZsX2jUUKUocvZqMGuE4hfAgMBAAE=\n-----END PUBLIC KEY-----\n"`,
	}
}

func NewFederationServerMockRepository(id int64) FederationServerMockRepository {
	return FederationServerMockRepository{
		ID: id,
	}
}

func (p FederationServerMockPerson) marshal(host string) string {
	return fmt.Sprintf(`{"@context":["https://www.w3.org/ns/activitystreams","https://w3id.org/security/v1"],`+
		`"id":"http://%[1]v/api/activitypub/user-id/%[2]v",`+
		`"type":"Person",`+
		`"icon":{"type":"Image","mediaType":"image/png","url":"http://%[1]v/avatars/1bb05d9a5f6675ed0272af9ea193063c"},`+
		`"url":"http://%[1]v/%[2]v",`+
		`"inbox":"http://%[1]v/api/activitypub/user-id/%[2]v/inbox",`+
		`"outbox":"http://%[1]v/api/activitypub/user-id/%[2]v/outbox",`+
		`"preferredUsername":"%[3]v",`+
		`"publicKey":{"id":"http://%[1]v/api/activitypub/user-id/%[2]v#main-key",`+
		`"owner":"http://%[1]v/api/activitypub/user-id/%[2]v",`+
		`"publicKeyPem":%[4]v}}`, host, p.ID, p.Name, p.PubKey)
}

func NewFederationServerMock() *FederationServerMock {
	return &FederationServerMock{
		Persons: []FederationServerMockPerson{
			NewFederationServerMockPerson(15, "stargoose1"),
			NewFederationServerMockPerson(30, "stargoose2"),
		},
		Repositories: []FederationServerMockRepository{
			NewFederationServerMockRepository(1),
		},
		LastPost: "",
	}
}

func (mock *FederationServerMock) DistantServer(t *testing.T) *httptest.Server {
	federatedRoutes := http.NewServeMux()
	federatedRoutes.HandleFunc("/.well-known/nodeinfo",
		func(res http.ResponseWriter, req *http.Request) {
			// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/.well-known/nodeinfo
			// TODO: as soon as content-type will become important:  content-type: application/json;charset=utf-8
			fmt.Fprintf(res, `{"links":[{"href":"http://%s/api/v1/nodeinfo","rel":"http://nodeinfo.diaspora.software/ns/schema/2.1"}]}`, req.Host)
		})
	federatedRoutes.HandleFunc("/api/v1/nodeinfo",
		func(res http.ResponseWriter, req *http.Request) {
			// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/api/v1/nodeinfo
			fmt.Fprint(res, `{"version":"2.1","software":{"name":"forgejo","version":"1.20.0+dev-3183-g976d79044",`+
				`"repository":"https://codeberg.org/forgejo/forgejo.git","homepage":"https://forgejo.org/"},`+
				`"protocols":["activitypub"],"services":{"inbound":[],"outbound":["rss2.0"]},`+
				`"openRegistrations":true,"usage":{"users":{"total":14,"activeHalfyear":2}},"metadata":{}}`)
		})
	for _, person := range mock.Persons {
		federatedRoutes.HandleFunc(fmt.Sprintf("/api/v1/activitypub/user-id/%v", person.ID),
			func(res http.ResponseWriter, req *http.Request) {
				// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/2
				fmt.Fprint(res, person.marshal(req.Host))
			})
	}
	for _, repository := range mock.Repositories {
		federatedRoutes.HandleFunc(fmt.Sprintf("/api/v1/activitypub/repository-id/%v/inbox/", repository.ID),
			func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "POST" {
					t.Errorf("POST expected at: %q", req.URL.EscapedPath())
				}
				buf := new(strings.Builder)
				_, err := io.Copy(buf, req.Body)
				if err != nil {
					t.Errorf("Error reading body: %q", err)
				}
				mock.LastPost = buf.String()
			})
	}
	federatedRoutes.HandleFunc("/",
		func(res http.ResponseWriter, req *http.Request) {
			t.Errorf("Unhandled request: %q", req.URL.EscapedPath())
		})
	federatedSrv := httptest.NewServer(federatedRoutes)
	return federatedSrv
}
