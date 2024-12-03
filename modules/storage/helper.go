// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package storage

import (
	"fmt"
	"io"
	"net/url"
	"os"
)

var UninitializedStorage = DiscardStorage("uninitialized storage")

type DiscardStorage string

func (s DiscardStorage) Open(_ string) (Object, error) {
	return nil, fmt.Errorf("%s", s)
}

func (s DiscardStorage) Save(_ string, _ io.Reader, _ int64) (int64, error) {
	return 0, fmt.Errorf("%s", s)
}

func (s DiscardStorage) Stat(_ string) (os.FileInfo, error) {
	return nil, fmt.Errorf("%s", s)
}

func (s DiscardStorage) Delete(_ string) error {
	return fmt.Errorf("%s", s)
}

func (s DiscardStorage) URL(_, _ string, _ url.Values) (*url.URL, error) {
	return nil, fmt.Errorf("%s", s)
}

func (s DiscardStorage) IterateObjects(_ string, _ func(string, Object) error) error {
	return fmt.Errorf("%s", s)
}
