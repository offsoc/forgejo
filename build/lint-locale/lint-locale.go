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

	"forgejo.org/modules/translation/localeiter"

	"github.com/microcosm-cc/bluemonday"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	policy     *bluemonday.Policy
	tagRemover *strings.Replacer
	safeURL    = "https://TO-BE-REPLACED.COM"

	// Matches href="", href="#", href="%s", href="#%s", href="%[1]s" and href="#%[1]s".
	placeHolderRegex = regexp.MustCompile(`href="#?(%s|%\[\d\]s)?"`)

	dmp = diffmatchpatch.New()
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
	policy.AllowElements("strong", "br", "b", "strike", "code", "i", "kbd")

	// TODO: Remove <c> in `actions.workflow.dispatch.trigger_found`.
	policy.AllowNoAttrs().OnElements("c")
}

func initRemoveTags() {
	oldnew := []string{}
	for _, el := range []string{
		"email@example.com", "correu@example.com", "epasts@domens.lv", "email@exemplo.com", "eposta@ornek.com", "email@példa.hu", "email@esempio.it",
		"user", "utente", "lietotājs", "gebruiker", "usuário", "Benutzer", "Bruker", "bruger", "użytkownik",
		"server", "servidor", "kiszolgáló", "serveris",
		"label", "etichetta", "etiķete", "rótulo", "Label", "utilizador", "etiket", "iezīme", "etykieta",
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

func checkValue(trKey, value string) []string {
	keyValue := preprocessTranslationValue(value)

	if html.UnescapeString(policy.Sanitize(keyValue)) == keyValue {
		return nil
	}

	// Create a nice diff of the difference.
	diffs := dmp.DiffMain(keyValue, html.UnescapeString(policy.Sanitize(keyValue)), false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCleanupEfficiency(diffs)

	return []string{trKey + ": " + dmp.DiffPrettyText(diffs)}
}

func checkLocaleContent(localeContent []byte) []string {
	errors := []string{}

	if err := localeiter.IterateMessagesContent(localeContent, func(trKey, trValue string) error {
		errors = append(errors, checkValue(trKey, trValue)...)
		return nil
	}); err != nil {
		panic(err)
	}

	return errors
}

func checkLocaleNextContent(localeContent []byte) []string {
	errors := []string{}

	if err := localeiter.IterateMessagesNextContent(localeContent, func(trKey, pluralForm, trValue string) error {
		fullKey := trKey
		if pluralForm != "" {
			fullKey = trKey + "." + pluralForm
		}
		errors = append(errors, checkValue(fullKey, trValue)...)
		return nil
	}); err != nil {
		panic(err)
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

	// Safety check that we are not reading the wrong directory.
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
			fmt.Println(localeFile.Name())
			panic(err)
		}

		if err := checkLocaleContent(localeContent); len(err) > 0 {
			fmt.Println(localeFile.Name())
			fmt.Println(strings.Join(err, "\n"))
			fmt.Println()
			exitCode = 1
		}
	}

	// Check the locale next.
	localeDir = filepath.Join("options", "locale_next")
	localeFiles, err = os.ReadDir(localeDir)
	if err != nil {
		panic(err)
	}

	// Safety check that we are not reading the wrong directory.
	if !slices.ContainsFunc(localeFiles, func(e fs.DirEntry) bool { return strings.HasSuffix(e.Name(), ".json") }) {
		fmt.Println("No locale_next files found")
		os.Exit(1)
	}

	for _, localeFile := range localeFiles {
		localeContent, err := os.ReadFile(filepath.Join(localeDir, localeFile.Name()))
		if err != nil {
			fmt.Println(localeFile.Name())
			panic(err)
		}

		if err := checkLocaleNextContent(localeContent); len(err) > 0 {
			fmt.Println(localeFile.Name())
			fmt.Println(strings.Join(err, "\n"))
			fmt.Println()
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
