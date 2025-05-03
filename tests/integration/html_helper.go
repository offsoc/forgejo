// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTMLDoc struct
type HTMLDoc struct {
	doc *goquery.Document
}

// NewHTMLParser parse html file
func NewHTMLParser(t testing.TB, body *bytes.Buffer) *HTMLDoc {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(body)
	require.NoError(t, err)
	return &HTMLDoc{doc: doc}
}

// GetInputValueByID for get input value by id
func (doc *HTMLDoc) GetInputValueByID(id string) string {
	text, _ := doc.doc.Find("#" + id).Attr("value")
	return text
}

// GetInputValueByName for get input value by name
func (doc *HTMLDoc) GetInputValueByName(name string) string {
	text, _ := doc.doc.Find("input[name=\"" + name + "\"]").Attr("value")
	return text
}

func (doc *HTMLDoc) AssertDropdown(t testing.TB, name string) *goquery.Selection {
	t.Helper()

	dropdownGroup := doc.Find(fmt.Sprintf(".dropdown:has(input[name='%s'])", name))
	assert.Equal(t, 1, dropdownGroup.Length(), "%s dropdown does not exist", name)
	return dropdownGroup
}

// Assert that a dropdown has at least one non-empty option
func (doc *HTMLDoc) AssertDropdownHasOptions(t testing.TB, dropdownName string) {
	t.Helper()

	options := doc.AssertDropdown(t, dropdownName).Find(".menu [data-value]:not([data-value=''])")
	assert.Positive(t, options.Length(), "%s dropdown has no options", dropdownName)
}

func (doc *HTMLDoc) AssertDropdownHasSelectedOption(t testing.TB, dropdownName, expectedValue string) {
	t.Helper()

	dropdownGroup := doc.AssertDropdown(t, dropdownName)

	selectedValue, _ := dropdownGroup.Find(fmt.Sprintf("input[name='%s']", dropdownName)).Attr("value")
	assert.Equal(t, expectedValue, selectedValue, "%s dropdown doesn't have expected value selected", dropdownName)

	dropdownValues := dropdownGroup.Find(".menu [data-value]").Map(func(i int, s *goquery.Selection) string {
		value, _ := s.Attr("data-value")
		return value
	})
	assert.Contains(t, dropdownValues, expectedValue, "%s dropdown doesn't have an option with expected value", dropdownName)
}

// Find gets the descendants of each element in the current set of
// matched elements, filtered by a selector. It returns a new Selection
// object containing these matched elements.
func (doc *HTMLDoc) Find(selector string) *goquery.Selection {
	return doc.doc.Find(selector)
}

// GetCSRF for getting CSRF token value from input
func (doc *HTMLDoc) GetCSRF() string {
	return doc.GetInputValueByName("_csrf")
}

// AssertElement check if element by selector exists or does not exist depending on checkExists
func (doc *HTMLDoc) AssertElement(t testing.TB, selector string, checkExists bool) {
	sel := doc.doc.Find(selector)
	if checkExists {
		assert.Equal(t, 1, sel.Length())
	} else {
		assert.Equal(t, 0, sel.Length())
	}
}
