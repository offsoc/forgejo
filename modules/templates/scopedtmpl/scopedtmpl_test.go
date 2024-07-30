// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package scopedtmpl

import (
	"bytes"
	"html/template"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedTemplateSetFuncMap(t *testing.T) {
	all := template.New("")

	all.Funcs(template.FuncMap{"CtxFunc": func(s string) string {
		return "default"
	}})

	_, err := all.New("base").Parse(`{{CtxFunc "base"}}`)
	require.NoError(t, err)

	_, err = all.New("test").Parse(strings.TrimSpace(`
{{template "base"}}
{{CtxFunc "test"}}
{{template "base"}}
{{CtxFunc "test"}}
`))
	require.NoError(t, err)

	ts, err := newScopedTemplateSet(all, "test")
	require.NoError(t, err)

	// try to use different CtxFunc to render concurrently

	funcMap1 := template.FuncMap{
		"CtxFunc": func(s string) string {
			time.Sleep(100 * time.Millisecond)
			return s + "1"
		},
	}

	funcMap2 := template.FuncMap{
		"CtxFunc": func(s string) string {
			time.Sleep(100 * time.Millisecond)
			return s + "2"
		},
	}

	out1 := bytes.Buffer{}
	out2 := bytes.Buffer{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		err := ts.newExecutor(funcMap1).Execute(&out1, nil)
		require.NoError(t, err)
		wg.Done()
	}()
	go func() {
		err := ts.newExecutor(funcMap2).Execute(&out2, nil)
		require.NoError(t, err)
		wg.Done()
	}()
	wg.Wait()
	assert.Equal(t, "base1\ntest1\nbase1\ntest1", out1.String())
	assert.Equal(t, "base2\ntest2\nbase2\ntest2", out2.String())
}

func TestScopedTemplateSetEscape(t *testing.T) {
	all := template.New("")
	_, err := all.New("base").Parse(`<a href="?q={{.param}}">{{.text}}</a>`)
	require.NoError(t, err)

	_, err = all.New("test").Parse(`{{template "base" .}}<form action="?q={{.param}}">{{.text}}</form>`)
	require.NoError(t, err)

	ts, err := newScopedTemplateSet(all, "test")
	require.NoError(t, err)

	out := bytes.Buffer{}
	err = ts.newExecutor(nil).Execute(&out, map[string]string{"param": "/", "text": "<"})
	require.NoError(t, err)

	assert.Equal(t, `<a href="?q=%2f">&lt;</a><form action="?q=%2f">&lt;</form>`, out.String())
}

func TestScopedTemplateSetUnsafe(t *testing.T) {
	all := template.New("")
	_, err := all.New("test").Parse(`<a href="{{if true}}?{{end}}a={{.param}}"></a>`)
	require.NoError(t, err)

	_, err = newScopedTemplateSet(all, "test")
	require.ErrorContains(t, err, "appears in an ambiguous context within a URL")
}
