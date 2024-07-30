package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestExploreCodeSearchIndexer(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Indexer.RepoIndexerEnabled, true)()

	req := NewRequest(t, "GET", "/explore/code")
	resp := MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	msg := doc.Find(".explore").Find(".ui.container").Find(".ui.message[data-test-tag=grep]")

	assert.Empty(t, msg.Nodes)
}
