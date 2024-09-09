package integration

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestRepoLastUpdatedTime(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := "user2"
		session := loginUser(t, user)

		req := NewRequest(t, "GET", path.Join("explore", "repos"))
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		node := doc.doc.Find(".flex-item-body").First()

		{
			buf := ""
			findTextNonNested(t, node, &buf)
			assert.Equal(t, "Updated", strings.TrimSpace(buf))
		}

		// Relative time should be present as a descendent
		{
			relativeTime := node.Find("relative-time").Text()
			assert.True(t, strings.HasPrefix(relativeTime, "19")) // ~1970, might underflow with timezone
		}
	})
}

func TestBranchLastUpdatedTime(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := "user2"
		repo := "repo1"
		session := loginUser(t, user)

		req := NewRequest(t, "GET", path.Join(user, repo, "branches"))
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		node := doc.doc.Find("p:has(span.commit-message)")

		{
			buf := ""
			findTextNonNested(t, node, &buf)
			assert.True(t, strings.Contains(buf, "Updated"))
		}

		{
			relativeTime := node.Find("relative-time").Text()
			assert.True(t, strings.HasPrefix(relativeTime, "2017"))
		}
	})
}

// Find all text that are direct descendents
func findTextNonNested(t *testing.T, n *goquery.Selection, buf *string) {
	t.Helper()

	n.Contents().Each(func(i int, s *goquery.Selection) {
		if goquery.NodeName(s) == "#text" {
			*buf += s.Text()
		}
	})
}
