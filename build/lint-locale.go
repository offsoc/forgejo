// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//nolint:forbidigo
package main

import (
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/ini.v1" //nolint:depguard
)

var (
	policy     *bluemonday.Policy
	tagRemover *strings.Replacer
	safeURL    = "https://TO-BE-REPLACED.COM"

	// Matches href="", href="#", href="%s", href="#%s", href="%[1]s" and href="#%[1]s".
	placeHolderRegex = regexp.MustCompile(`href="#?(%s|%\[\d\]s)?"`)
)

func initBlueMondayPolicy() {
	policy = bluemonday.NewPolicy()

	policy.RequireParseableURLs(true)
	policy.AllowURLSchemes("https")

	// Only allow safe URL on href.
	// Only allow target="_blank".
	// Only allow rel="nopener noreferrer", rel="noopener" and rel="noreferrer".
	// Only allow placeholder on id and class.
	policy.AllowAttrs("href").Matching(regexp.MustCompile("^" + regexp.QuoteMeta(safeURL) + "$")).OnElements("a")
	policy.AllowAttrs("target").Matching(regexp.MustCompile("^_blank$")).OnElements("a")
	policy.AllowAttrs("rel").Matching(regexp.MustCompile("^(noopener|noreferrer|noopener noreferrer)$")).OnElements("a")
	policy.AllowAttrs("id", "class").Matching(regexp.MustCompile(`^%s|%\[\d\]s$`)).OnElements("a")

	// Only allow positional placeholder as class.
	positionalPlaceholderRe := regexp.MustCompile(`^%\[\d\]s$`)
	policy.AllowAttrs("class").Matching(positionalPlaceholderRe).OnElements("strong")
	policy.AllowAttrs("id").Matching(positionalPlaceholderRe).OnElements("code")

	// Allowed elements with no attributes. Must be a recognized tagname.
	policy.AllowElements("strong", "br", "b", "strike", "code", "i")

	// TODO: Remove <c> in `actions.workflow.dispatch.trigger_found`.
	policy.AllowNoAttrs().OnElements("c")
}

func initRemoveTags() {
	oldnew := []string{}
	for _, el := range []string{
		"email@example.com", "correu@example.com", "epasts@domens.lv", "email@exemplo.com", "eposta@ornek.com", "email@példa.hu", "email@esempio.it",
		"user", "utente", "lietotājs", "gebruiker", "usuário", "Benutzer", "Bruker",
		"server", "servidor", "kiszolgáló", "serveris",
		"label", "etichetta", "etiķete", "rótulo", "Label", "utilizador",
		"filename", "bestandsnaam", "dosyaadi", "fails", "nome do arquivo",
	} {
		oldnew = append(oldnew, "<"+el+">", "REPLACED-TAG")
	}

	tagRemover = strings.NewReplacer(oldnew...)
}

func preprocessTranslationValue(value string) string {
	// href should be a parsable URL, replace placeholder strings with a safe url.
	value = placeHolderRegex.ReplaceAllString(value, `href="`+safeURL+`"`)

	// Remove tags that aren't tags but will be parsed as tags. We already know they are safe and sound.
	value = tagRemover.Replace(value)

	return value
}

func checkLocaleContent(localeContent []byte) []string {
	// Same configuration as Forgejo uses.
	cfg := ini.Empty(ini.LoadOptions{
		IgnoreContinuation: true,
	})
	cfg.NameMapper = ini.SnackCase

	if err := cfg.Append(localeContent); err != nil {
		panic(err)
	}

	dmp := diffmatchpatch.New()
	errors := []string{}

	for _, section := range cfg.Sections() {
		for _, key := range section.Keys() {
			var trKey string
			if section.Name() == "" || section.Name() == "DEFAULT" || section.Name() == "common" {
				trKey = key.Name()
			} else {
				trKey = section.Name() + "." + key.Name()
			}

			keyValue := preprocessTranslationValue(key.Value())

			if html.UnescapeString(policy.Sanitize(keyValue)) != keyValue {
				// Create a nice diff of the difference.
				diffs := dmp.DiffMain(keyValue, html.UnescapeString(policy.Sanitize(keyValue)), false)
				diffs = dmp.DiffCleanupSemantic(diffs)
				diffs = dmp.DiffCleanupEfficiency(diffs)

				errors = append(errors, trKey+": "+dmp.DiffPrettyText(diffs))
			}
		}
	}
	return errors
}

func main() {
	initBlueMondayPolicy()
	initRemoveTags()

	localeDir := filepath.Join("options", "locale")
	localeFiles, err := os.ReadDir(localeDir)
	if err != nil {
		panic(err)
	}

	if !slices.ContainsFunc(localeFiles, func(e fs.DirEntry) bool { return strings.HasSuffix(e.Name(), ".ini") }) {
		fmt.Println("No locale files found")
		os.Exit(1)
	}

	exitCode := 0
	for _, localeFile := range localeFiles {
		if !strings.HasSuffix(localeFile.Name(), ".ini") {
			continue
		}

		localeContent, err := os.ReadFile(filepath.Join(localeDir, localeFile.Name()))
		if err != nil {
			panic(err)
		}

		if err := checkLocaleContent(localeContent); len(err) > 0 {
			fmt.Println(localeFile.Name())
			fmt.Println(strings.Join(err, "\n"))
			fmt.Println()
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
