// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
)

// Signature represents the Author, Committer or Tagger information.
type Signature struct {
	Name  string    // the committer name, it can be anything
	Email string    // the committer email, it can be anything
	When  time.Time // the timestamp of the signature
}

func (s *Signature) String() string {
	return fmt.Sprintf("%s <%s>", s.Name, s.Email)
}

// Decode decodes a byte array representing a signature to signature
func (s *Signature) Decode(b []byte) {
	*s = *parseSignatureFromCommitLine(util.UnsafeBytesToString(b))
}

// Helper to get a signature from the commit line, which looks like:
//
//	full name <user@example.com> 1378823654 +0200
//
// Haven't found the official reference for the standard format yet.
// This function never fails, if the "line" can't be parsed, it returns a default Signature with "zero" time.
func parseSignatureFromCommitLine(line string) *Signature {
	sig := &Signature{}
	s1, sx, ok1 := strings.Cut(line, " <")
	s2, s3, ok2 := strings.Cut(sx, "> ")
	if !ok1 || !ok2 {
		sig.Name = line
		return sig
	}
	sig.Name, sig.Email = s1, s2

	if strings.Count(s3, " ") == 1 {
		ts, tz, _ := strings.Cut(s3, " ")
		seconds, _ := strconv.ParseInt(ts, 10, 64)
		if tzTime, err := time.Parse("-0700", tz); err == nil {
			sig.When = time.Unix(seconds, 0).In(tzTime.Location())
		}
	} else {
		// the old gitea code tried to parse the date in a few different formats, but it's not clear why.
		// according to public document, only the standard format "timestamp timezone" could be found, so drop other formats.
		log.Error("suspicious commit line format: %q", line)
		for _, fmt := range []string{ /*"Mon Jan _2 15:04:05 2006 -0700"*/ } {
			if t, err := time.Parse(fmt, s3); err == nil {
				sig.When = t
				break
			}
		}
	}
	return sig
}
