// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package wiki

import (
	"net/url"
	"path"
	"strings"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/services/convert"
)

// To define the wiki related concepts:
// * Display Segment: the text what user see for a wiki page (aka, the title):
//   - "Home Page"
//   - "100% Free"
//   - "2000-01-02 meeting"
// * Web Path:
//   - "/wiki/Home-Page"
//   - "/wiki/100%25+Free"
//   - "/wiki/2000-01-02+meeting.-"
//   - If a segment has a suffix "DashMarker(.-)", it means that there is no dash-space conversion for this segment.
//   - If a WebPath is a "*.md" pattern, then use the unescaped value directly as GitPath, to make users can access the raw file.
// * Git Path (only space doesn't need to be escaped):
//   - "/.wiki.git/Home-Page.md"
//   - "/.wiki.git/100%25 Free.md"
//   - "/.wiki.git/2000-01-02 meeting.-.md"
// TODO: support subdirectory in the future
//
// Although this package now has the ability to support subdirectory, but the route package doesn't:
// * Double-escaping problem: the URL "/wiki/abc%2Fdef" becomes "/wiki/abc/def" by ctx.Params, which is incorrect
//   * This problem should have been 99% fixed, but it needs more tests.
// * The old wiki code's behavior is always using %2F, instead of subdirectory, so there are a lot of legacy "%2F" files in user wikis.

type (
	WebPath string
	GitPath string
	URLPath string
)

type Path struct {
	file WebPath
}

func WebPathToPath(web WebPath) Path {
	return Path{
		file: web,
	}
}

func RequestToPath(req string) Path {
	s := util.PathJoinRelX(req)
	// The old wiki code's behavior is always using %2F, instead of subdirectory.
	s = strings.ReplaceAll(s, "/", "%2F")
	return Path{
		file: WebPath(s),
	}
}

func TitleToPath(title string) Path {
	// TODO: no support for subdirectory, because the old wiki code's behavior is always using %2F, instead of subdirectory.
	// So we do not add the support for writing slashes in title at the moment.
	title = strings.TrimSpace(title)
	title = util.PathJoinRelX(escapeSegToWeb(title, false))
	if title == "" || title == "." {
		title = "unnamed"
	}
	return Path{
		WebPath(title),
	}
}

func GitPathToPath(g GitPath) (*Path, error) {
	gitPath := string(g)
	if !strings.HasSuffix(gitPath, ".md") {
		return nil, repo_model.ErrWikiInvalidFileName{FileName: gitPath}
	}
	gitPath = strings.TrimSuffix(gitPath, ".md")
	a := strings.Split(gitPath, "/")
	for i := range a {
		shouldAddDashMarker := hasDashMarker(a[i])
		s, err := unescapeSegment(a[i])
		if err != nil {
			return nil, err
		}
		a[i] = s
		a[i] = escapeSegToWeb(a[i], shouldAddDashMarker)
	}
	return &Path{
		file: WebPath(strings.Join(a, "/")),
	}, nil
}

func (w Path) DisplayName() (dir, display string) {
	dir = path.Dir(string(w.file))
	display = path.Base(string(w.file))
	if strings.HasSuffix(display, ".md") {
		display = strings.TrimSuffix(display, ".md")
		display, _ = url.PathUnescape(display)
	}
	display, _ = unescapeSegment(display)
	return dir, display
}

func (w Path) WebPath() WebPath {
	return w.file
}

func (w Path) GitPath() GitPath {
	if strings.HasSuffix(string(w.file), ".md") {
		ret, _ := url.PathUnescape(string(w.file))
		return GitPath(util.PathJoinRelX(ret))
	}

	a := strings.Split(string(w.file), "/")
	for i := range a {
		shouldAddDashMarker := hasDashMarker(a[i])
		a[i], _ = unescapeSegment(a[i])
		a[i] = escapeSegToWeb(a[i], shouldAddDashMarker)
		a[i] = strings.ReplaceAll(a[i], "%20", " ") // space is safe to be kept in git path
		a[i] = strings.ReplaceAll(a[i], "+", " ")
	}
	return GitPath(strings.Join(a, "/") + ".md")
}

func (w Path) URLPath() URLPath {
	return URLPath(w.file)
}

func WebPathSegments(s WebPath) []string {
	a := strings.Split(string(s), "/")
	for i := range a {
		a[i], _ = unescapeSegment(a[i])
	}
	return a
}

// ToWikiPageMetaData converts meta information to a WikiPageMetaData
func ToWikiPageMetaData(wikiPath Path, lastCommit *git.Commit, repo *repo_model.Repository) *api.WikiPageMetaData {
	subURL := string(wikiPath.WebPath())
	_, title := wikiPath.DisplayName()
	return &api.WikiPageMetaData{
		Title:      title,
		HTMLURL:    util.URLJoin(repo.HTMLURL(), "wiki", subURL),
		SubURL:     subURL,
		LastCommit: convert.ToWikiCommit(lastCommit),
	}
}

var reservedWikiNames = []string{"_pages", "_new", "_edit", "raw"}

func validateWebPath(name Path) error {
	for _, s := range WebPathSegments(name.WebPath()) {
		if util.SliceContainsString(reservedWikiNames, s) {
			return repo_model.ErrWikiReservedName{Title: s}
		}
	}
	return nil
}

func hasDashMarker(s string) bool {
	return strings.HasSuffix(s, ".-")
}

func removeDashMarker(s string) string {
	return strings.TrimSuffix(s, ".-")
}

func addDashMarker(s string) string {
	return s + ".-"
}

func unescapeSegment(s string) (string, error) {
	if hasDashMarker(s) {
		s = removeDashMarker(s)
	} else {
		s = strings.ReplaceAll(s, "-", " ")
	}
	unescaped, err := url.QueryUnescape(s)
	if err != nil {
		return s, err // un-escaping failed, but it's still safe to return the original string, because it is only a title for end users
	}
	return unescaped, nil
}

func escapeSegToWeb(s string, hadDashMarker bool) string {
	if hadDashMarker || strings.Contains(s, "-") || strings.HasSuffix(s, ".md") {
		s = addDashMarker(s)
	} else {
		s = strings.ReplaceAll(s, " ", "-")
	}
	s = url.QueryEscape(s)
	return s
}
