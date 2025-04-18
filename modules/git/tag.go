// Copyright 2015 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"strings"

	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
)

const (
	beginpgp = "\n-----BEGIN PGP SIGNATURE-----\n"
	endpgp   = "\n-----END PGP SIGNATURE-----"
	beginssh = "\n-----BEGIN SSH SIGNATURE-----\n"
	endssh   = "\n-----END SSH SIGNATURE-----"
)

// Tag represents a Git tag.
type Tag struct {
	Name                 string
	ID                   ObjectID
	Object               ObjectID // The id of this commit object
	Type                 string
	Tagger               *Signature
	Message              string
	Signature            *ObjectSignature
	ArchiveDownloadCount *api.TagArchiveDownloadCount
}

// Commit return the commit of the tag reference
func (tag *Tag) Commit(gitRepo *Repository) (*Commit, error) {
	return gitRepo.getCommit(tag.Object)
}

// Parse commit information from the (uncompressed) raw
// data from the commit object.
// \n\n separate headers from message
func parseTagData(objectFormat ObjectFormat, data []byte) (*Tag, error) {
	tag := new(Tag)
	tag.ID = objectFormat.EmptyObjectID()
	tag.Object = objectFormat.EmptyObjectID()
	tag.Tagger = &Signature{}
	// we now have the contents of the commit object. Let's investigate...
	nextline := 0
l:
	for {
		eol := bytes.IndexByte(data[nextline:], '\n')
		switch {
		case eol > 0:
			line := data[nextline : nextline+eol]
			spacepos := bytes.IndexByte(line, ' ')
			reftype := line[:spacepos]
			switch string(reftype) {
			case "object":
				id, err := NewIDFromString(string(line[spacepos+1:]))
				if err != nil {
					return nil, err
				}
				tag.Object = id
			case "type":
				// A commit can have one or more parents
				tag.Type = string(line[spacepos+1:])
			case "tagger":
				tag.Tagger = parseSignatureFromCommitLine(util.UnsafeBytesToString(line[spacepos+1:]))
			}
			nextline += eol + 1
		case eol == 0:
			tag.Message = string(data[nextline+1:])
			break l
		default:
			break l
		}
	}

	extractTagSignature := func(signatureBeginMark, signatureEndMark string) (bool, *ObjectSignature, string) {
		idx := strings.LastIndex(tag.Message, signatureBeginMark)
		if idx == -1 {
			return false, nil, ""
		}

		endSigIdx := strings.Index(tag.Message[idx:], signatureEndMark)
		if endSigIdx == -1 {
			return false, nil, ""
		}

		return true, &ObjectSignature{
			Signature: tag.Message[idx+1 : idx+endSigIdx+len(signatureEndMark)],
			Payload:   string(data[:bytes.LastIndex(data, []byte(signatureBeginMark))+1]),
		}, tag.Message[:idx+1]
	}

	// Try to find an OpenPGP signature
	found, sig, message := extractTagSignature(beginpgp, endpgp)
	if !found {
		// If not found, try an SSH one
		found, sig, message = extractTagSignature(beginssh, endssh)
	}
	// If either is found, update the tag Signature and Message
	if found {
		tag.Signature = sig
		tag.Message = message
	}

	return tag, nil
}
