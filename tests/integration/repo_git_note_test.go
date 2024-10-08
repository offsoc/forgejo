package integration

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoModifyGitNotes(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		session := loginUser(t, "user2")

		req := NewRequest(t, "GET", "/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d")
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Contains(t, resp.Body.String(), "<pre class=\"commit-body\">This is a test note\n</pre>")
		assert.Contains(t, resp.Body.String(), "commit-notes-display-area")

		t.Run("Set", func(t *testing.T) {
			req = NewRequestWithValues(t, "POST", "/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d/notes", map[string]string{
				"_csrf": GetCSRF(t, session, "/user2/repo1"),
				"notes": "This is a new note",
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			req = NewRequest(t, "GET", "/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d")
			resp = MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Body.String(), "<pre class=\"commit-body\">This is a new note\n</pre>")
			assert.Contains(t, resp.Body.String(), "commit-notes-display-area")
		})

		t.Run("Delete", func(t *testing.T) {
			req = NewRequestWithValues(t, "POST", "/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d/notes/remove", map[string]string{
				"_csrf": GetCSRF(t, session, "/user2/repo1"),
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			req = NewRequest(t, "GET", "/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d")
			resp = MakeRequest(t, req, http.StatusOK)
			assert.NotContains(t, resp.Body.String(), "commit-notes-display-area")
		})
	})
}
