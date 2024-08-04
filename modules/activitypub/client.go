// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// TODO: Think about whether this should be moved to services/activitypub (compare to exosy/services/activitypub/client.go)
package activitypub

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/proxy"
	"code.gitea.io/gitea/modules/setting"

	"github.com/go-fed/httpsig"
)

const (
	// ActivityStreamsContentType const
	ActivityStreamsContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`
	httpsigExpirationTime      = 60
)

func CurrentTime() string {
	return time.Now().UTC().Format(http.TimeFormat)
}

func containsRequiredHTTPHeaders(method string, headers []string) error {
	var hasRequestTarget, hasDate, hasDigest, hasHost bool
	for _, header := range headers {
		hasRequestTarget = hasRequestTarget || header == httpsig.RequestTarget
		hasDate = hasDate || header == "Date"
		hasDigest = hasDigest || header == "Digest"
		hasHost = hasHost || header == "Host"
	}
	if !hasRequestTarget {
		return fmt.Errorf("missing http header for %s: %s", method, httpsig.RequestTarget)
	} else if !hasDate {
		return fmt.Errorf("missing http header for %s: Date", method)
	} else if !hasHost {
		return fmt.Errorf("missing http header for %s: Host", method)
	} else if !hasDigest && method != http.MethodGet {
		return fmt.Errorf("missing http header for %s: Digest", method)
	}
	return nil
}

// Client struct
type Client struct {
	client      *http.Client
	algs        []httpsig.Algorithm
	digestAlg   httpsig.DigestAlgorithm
	getHeaders  []string
	postHeaders []string
	priv        *rsa.PrivateKey
	pubID       string
}

// NewClient function
func NewClient(ctx context.Context, user *user_model.User, pubID string) (c *Client, err error) {
	if err = containsRequiredHTTPHeaders(http.MethodGet, setting.Federation.GetHeaders); err != nil {
		return nil, err
	} else if err = containsRequiredHTTPHeaders(http.MethodPost, setting.Federation.PostHeaders); err != nil {
		return nil, err
	}

	priv, err := GetPrivateKey(ctx, user)
	if err != nil {
		return nil, err
	}
	privPem, _ := pem.Decode([]byte(priv))
	privParsed, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
	if err != nil {
		return nil, err
	}

	c = &Client{
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: proxy.Proxy(),
			},
			Timeout: 5 * time.Second,
		},
		algs:        setting.HttpsigAlgs,
		digestAlg:   httpsig.DigestAlgorithm(setting.Federation.DigestAlgorithm),
		getHeaders:  setting.Federation.GetHeaders,
		postHeaders: setting.Federation.PostHeaders,
		priv:        privParsed,
		pubID:       pubID,
	}
	return c, err
}

// NewRequest function
func (c *Client) newRequest(method string, b []byte, to string) (req *http.Request, err error) {
	buf := bytes.NewBuffer(b)
	req, err = http.NewRequest(method, to, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json, "+ActivityStreamsContentType)
	req.Header.Add("Date", CurrentTime())
	req.Header.Add("Host", req.URL.Host)
	req.Header.Add("User-Agent", "Gitea/"+setting.AppVer)
	req.Header.Add("Content-Type", ActivityStreamsContentType)

	return req, err
}

// Post function
func (c *Client) Post(b []byte, to string) (resp *http.Response, err error) {
	var req *http.Request
	if req, err = c.newRequest(http.MethodPost, b, to); err != nil {
		return nil, err
	}

	signer, _, err := httpsig.NewSigner(c.algs, c.digestAlg, c.postHeaders, httpsig.Signature, httpsigExpirationTime)
	if err != nil {
		return nil, err
	}
	if err := signer.SignRequest(c.priv, c.pubID, req, b); err != nil {
		return nil, err
	}

	resp, err = c.client.Do(req)
	return resp, err
}

// Create an http GET request with forgejo/gitea specific headers
func (c *Client) Get(to string) (resp *http.Response, err error) {
	var req *http.Request
	if req, err = c.newRequest(http.MethodGet, nil, to); err != nil {
		return nil, err
	}
	signer, _, err := httpsig.NewSigner(c.algs, c.digestAlg, c.getHeaders, httpsig.Signature, httpsigExpirationTime)
	if err != nil {
		return nil, err
	}
	if err := signer.SignRequest(c.priv, c.pubID, req, nil); err != nil {
		return nil, err
	}

	resp, err = c.client.Do(req)
	return resp, err
}

// Create an http GET request with forgejo/gitea specific headers
func (c *Client) GetBody(uri string) ([]byte, error) {
	response, err := c.Get(uri)
	if err != nil {
		return nil, err
	}
	log.Debug("Client: got status: %v", response.Status)
	if response.StatusCode != 200 {
		err = fmt.Errorf("got non 200 status code for id: %v", uri)
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	log.Debug("Client: got body: %v", charLimiter(string(body), 120))
	return body, nil
}

// Limit number of characters in a string (useful to prevent log injection attacks and overly long log outputs)
// Thanks to https://www.socketloop.com/tutorials/golang-characters-limiter-example
func charLimiter(s string, limit int) string {
	reader := strings.NewReader(s)
	buff := make([]byte, limit)
	n, _ := io.ReadAtLeast(reader, buff, limit)
	if n != 0 {
		return fmt.Sprint(string(buff), "...")
	}
	return s
}
