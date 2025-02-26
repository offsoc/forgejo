package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func DistantFederationServer(t *testing.T) *httptest.Server {
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
	federatedRoutes.HandleFunc("/api/v1/activitypub/user-id/15",
		func(res http.ResponseWriter, req *http.Request) {
			// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/2
			fmt.Fprint(res, `{"@context":["https://www.w3.org/ns/activitystreams","https://w3id.org/security/v1"],`+
				`"id":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/15","type":"Person",`+
				`"icon":{"type":"Image","mediaType":"image/png","url":"https://federated-repo.prod.meissa.de/avatars/1bb05d9a5f6675ed0272af9ea193063c"},`+
				`"url":"https://federated-repo.prod.meissa.de/stargoose1","inbox":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/15/inbox",`+
				`"outbox":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/15/outbox","preferredUsername":"stargoose1",`+
				`"publicKey":{"id":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/15#main-key","owner":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/15",`+
				`"publicKeyPem":"-----BEGIN PUBLIC KEY-----\nMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEA18H5s7N6ItZUAh9tneII\nIuZdTTa3cZlLa/9ejWAHTkcp3WLW+/zbsumlMrWYfBy2/yTm56qasWt38iY4D6ul\n`+
				`CPiwhAqX3REvVq8tM79a2CEqZn9ka6vuXoDgBg/sBf/BUWqf7orkjUXwk/U0Egjf\nk5jcurF4vqf1u+rlAHH37dvSBaDjNj6Qnj4OP12bjfaY/yvs7+jue/eNXFHjzN4E\n`+
				`T2H4B/yeKTJ4UuAwTlLaNbZJul2baLlHelJPAsxiYaziVuV5P+IGWckY6RSerRaZ\nAkc4mmGGtjAyfN9aewe+lNVfwS7ElFx546PlLgdQgjmeSwLX8FWxbPE5A/PmaXCs\n`+
				`nx+nou+3dD7NluULLtdd7K+2x02trObKXCAzmi5/Dc+yKTzpFqEz+hLNCz7TImP/\ncK//NV9Q+X67J9O27baH9R9ZF4zMw8rv2Pg0WLSw1z7lLXwlgIsDapeMCsrxkVO4\n`+
				`LXX5AQ1xQNtlssnVoUBqBrvZsX2jUUKUocvZqMGuE4hfAgMBAAE=\n-----END PUBLIC KEY-----\n"}}`)
		})
	federatedRoutes.HandleFunc("/api/v1/activitypub/user-id/30",
		func(res http.ResponseWriter, req *http.Request) {
			// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/3
			fmt.Fprint(res, `{"@context":["https://www.w3.org/ns/activitystreams","https://w3id.org/security/v1"],`+
				`"id":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/30","type":"Person",`+
				`"icon":{"type":"Image","mediaType":"image/png","url":"https://federated-repo.prod.meissa.de/avatars/9c03f03d1c1f13f21976a22489326fe1"},`+
				`"url":"https://federated-repo.prod.meissa.de/stargoose2","inbox":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/30/inbox",`+
				`"outbox":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/30/outbox","preferredUsername":"stargoose2",`+
				`"publicKey":{"id":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/30#main-key","owner":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/30",`+
				`"publicKeyPem":"-----BEGIN PUBLIC KEY-----\nMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAyv5NytsfqpWXSrwuk8a3\n0W1zE13QJioXb/e3opgN2CfKZkdm3hb+4+mGKoU/rCqegnL9/AO0Aw+R8fCHXx44\n`+
				`iNkdVpdY8Dzq+tQ9IetPWbyVIBvSzGgvpqfS05JuVPsy8cBX9wByODjr5kq7k1/v\nY1G7E3uh0a/XJc+mZutwGC3gPgR93NSrqsvTPN4wdhCCu9uj02S8OBoKuSYaPkU+\n`+
				`tZ4CEDpnclAOw/eNiH4x2irMvVtruEgtlTA5K2I4YJrmtGLidus47FCyc8/zEKUh\nAeiD8KWDvqsQgOhUwcQgRxAnYVCoMD9cnE+WFFRHTuQecNlmdNFs3Cr0yKcWjDde\n`+
				`trvnehW7LfPveGb0tHRHPuVAJpncTOidUR5h/7pqMyvKHzuAHWomm9rEaGUxd/7a\nL1CFjAf39+QIEgu0Anj8mIc7CTiz+DQhDz+0jBOsQ0iDXc5GeBz7X9Xv4Jp966nq\n`+
				`MUR0GQGXvfZQN9IqMO+WoUVy10Ddhns1EWGlA0x4fecnAgMBAAE=\n-----END PUBLIC KEY-----\n"}}`)
		})
	federatedRoutes.HandleFunc("/",
		func(res http.ResponseWriter, req *http.Request) {
			t.Errorf("Unhandled request: %q", req.URL.EscapedPath())
		})
	federatedSrv := httptest.NewServer(federatedRoutes)
	return federatedSrv
}
