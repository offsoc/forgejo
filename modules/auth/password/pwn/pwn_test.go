// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pwn

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTransport struct{}

func (mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "api.pwnedpasswords.com" {
		return nil, errors.New("unexpected host")
	}

	res := &http.Response{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Proto:      "HTTP/1.1",
		Request:    req,
		Header:     make(http.Header),
		StatusCode: 200,
	}

	switch req.URL.Path {
	case "/range/5c1d8":
		res.Body = io.NopCloser(strings.NewReader("EAF2F254732680E8AC339B84F3266ECCBB5:1\r\nFC446EB88938834178CB9322C1EE273C2A7:2"))
		return res, nil
	case "/range/ba189":
		res.Body = io.NopCloser(strings.NewReader("FD4CB34F0378BCB15D23F6FFD28F0775C9E:3\r\nFDF342FCD8C3611DAE4D76E8A992A3E4169:4"))
		return res, nil
	case "/range/a1733":
		res.Body = io.NopCloser(strings.NewReader("C4CE0F1F0062B27B9E2F41AF0C08218017C:1\r\nFC446EB88938834178CB9322C1EE273C2A7:2\r\nFE81480327C992FE62065A827429DD1318B:0"))
		return res, nil
	case "/range/5617b":
		res.Body = io.NopCloser(strings.NewReader("FD4CB34F0378BCB15D23F6FFD28F0775C9E:3\r\nFDF342FCD8C3611DAE4D76E8A992A3E4169:4\r\nFE81480327C992FE62065A827429DD1318B:0"))
		return res, nil
	case "/range/79082":
		res.Body = io.NopCloser(strings.NewReader("FDF342FCD8C3611DAE4D76E8A992A3E4169:4\r\nFE81480327C992FE62065A827429DD1318B:0\r\nAFEF386F56EB0B4BE314E07696E5E6E6536:0"))
		return res, nil
	}

	return nil, errors.New("unexpected path")
}

var client = New(WithHTTP(&http.Client{
	Timeout:   time.Second * 2,
	Transport: mockTransport{},
}))

func TestPassword(t *testing.T) {
	count, err := client.CheckPassword("", false)
	require.ErrorIs(t, err, ErrEmptyPassword, "blank input should return ErrEmptyPassword")
	assert.Equal(t, -1, count)

	count, err = client.CheckPassword("pwned", false)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = client.CheckPassword("notpwned", false)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	count, err = client.CheckPassword("paddedpwned", true)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = client.CheckPassword("paddednotpwned", true)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	count, err = client.CheckPassword("paddednotpwnedzero", true)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
