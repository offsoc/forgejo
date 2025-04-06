package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestExploreCodeSearchIndexer(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Indexer.RepoIndexerEnabled, true)()

	req := NewRequest(t, "GET", "/explore/code?q=file&fuzzy=true")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find(".explore")

	doc.Find(".file-body").Each(func(i int, sel *goquery.Selection) {
		assert.Positive(t, sel.Find(".code-inner").Find(".search-highlight").Length(), 0)
	})
}
