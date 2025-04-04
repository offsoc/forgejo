// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"cmp"
	"io"
	"strings"
	"unicode"

	"forgejo.org/modules/analyze"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"

	"github.com/go-enry/go-enry/v2"
)

const (
	fileSizeLimit int64 = 16 * 1024   // 16 KiB
	bigFileSize   int64 = 1024 * 1024 // 1 MiB
)

// mergeLanguageStats mergers language names with different cases. The name with most upper case letters is used.
func mergeLanguageStats(stats map[string]int64) map[string]int64 {
	names := map[string]struct {
		uniqueName string
		upperCount int
	}{}

	countUpper := func(s string) (count int) {
		for _, r := range s {
			if unicode.IsUpper(r) {
				count++
			}
		}
		return count
	}

	for name := range stats {
		cnt := countUpper(name)
		lower := strings.ToLower(name)
		if cnt >= names[lower].upperCount {
			names[lower] = struct {
				uniqueName string
				upperCount int
			}{uniqueName: name, upperCount: cnt}
		}
	}

	res := make(map[string]int64, len(names))
	for name, num := range stats {
		res[names[strings.ToLower(name)].uniqueName] += num
	}
	return res
}

// GetLanguageStats calculates language stats for git repository at specified commit
func (repo *Repository) GetLanguageStats(commitID string) (map[string]int64, error) {
	// We will feed the commit IDs in order into cat-file --batch, followed by blobs as necessary.
	// so let's create a batch stdin and stdout
	batchStdinWriter, batchReader, cancel, err := repo.CatFileBatch(repo.Ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	writeID := func(id string) error {
		_, err := batchStdinWriter.Write([]byte(id + "\n"))
		return err
	}

	if err := writeID(commitID); err != nil {
		return nil, err
	}
	shaBytes, typ, size, err := ReadBatchLine(batchReader)
	if typ != "commit" {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, ErrNotExist{commitID, ""}
	}

	sha, err := NewIDFromString(string(shaBytes))
	if err != nil {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, ErrNotExist{commitID, ""}
	}

	commit, err := CommitFromReader(repo, sha, io.LimitReader(batchReader, size))
	if err != nil {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, err
	}
	if _, err = batchReader.Discard(1); err != nil {
		return nil, err
	}

	tree := commit.Tree

	entries, err := tree.ListEntriesRecursiveWithSize()
	if err != nil {
		return nil, err
	}

	checker, err := repo.GitAttributeChecker(commitID, LinguistAttributes...)
	if err != nil {
		return nil, err
	}
	defer checker.Close()

	contentBuf := bytes.Buffer{}
	var content []byte

	// sizes contains the current calculated size of all files by language
	sizes := make(map[string]int64)
	// by default we will only count the sizes of programming languages or markup languages
	// unless they are explicitly set using linguist-language
	includedLanguage := map[string]bool{}
	// or if there's only one language in the repository
	firstExcludedLanguage := ""
	firstExcludedLanguageSize := int64(0)

	isTrue := func(v optional.Option[bool]) bool {
		return v.ValueOrDefault(false)
	}
	isFalse := func(v optional.Option[bool]) bool {
		return !v.ValueOrDefault(true)
	}

	for _, f := range entries {
		select {
		case <-repo.Ctx.Done():
			return sizes, repo.Ctx.Err()
		default:
		}

		contentBuf.Reset()
		content = contentBuf.Bytes()

		if f.Size() == 0 {
			continue
		}

		isVendored := optional.None[bool]()
		isGenerated := optional.None[bool]()
		isDocumentation := optional.None[bool]()
		isDetectable := optional.None[bool]()

		attrs, err := checker.CheckPath(f.Name())
		if err == nil {
			isVendored = attrs["linguist-vendored"].Bool()
			isGenerated = attrs["linguist-generated"].Bool()
			isDocumentation = attrs["linguist-documentation"].Bool()
			isDetectable = attrs["linguist-detectable"].Bool()
			if language := cmp.Or(
				attrs["linguist-language"].String(),
				attrs["gitlab-language"].Prefix(),
			); language != "" {
				// group languages, such as Pug -> HTML; SCSS -> CSS
				group := enry.GetLanguageGroup(language)
				if len(group) != 0 {
					language = group
				}

				// this language will always be added to the size
				sizes[language] += f.Size()
				continue
			}
		}

		if isFalse(isDetectable) || isTrue(isVendored) || isTrue(isDocumentation) ||
			(!isFalse(isVendored) && analyze.IsVendor(f.Name())) ||
			enry.IsDotFile(f.Name()) ||
			enry.IsConfiguration(f.Name()) ||
			(!isFalse(isDocumentation) && enry.IsDocumentation(f.Name())) {
			continue
		}

		// If content can not be read or file is too big just do detection by filename

		if f.Size() <= bigFileSize {
			if err := writeID(f.ID.String()); err != nil {
				return nil, err
			}
			_, _, size, err := ReadBatchLine(batchReader)
			if err != nil {
				log.Debug("Error reading blob: %s Err: %v", f.ID.String(), err)
				return nil, err
			}

			sizeToRead := size
			discard := int64(1)
			if size > fileSizeLimit {
				sizeToRead = fileSizeLimit
				discard = size - fileSizeLimit + 1
			}

			_, err = contentBuf.ReadFrom(io.LimitReader(batchReader, sizeToRead))
			if err != nil {
				return nil, err
			}
			content = contentBuf.Bytes()
			if err := DiscardFull(batchReader, discard); err != nil {
				return nil, err
			}
		}
		if !isTrue(isGenerated) && enry.IsGenerated(f.Name(), content) {
			continue
		}

		// FIXME: Why can't we split this and the IsGenerated tests to avoid reading the blob unless absolutely necessary?
		// - eg. do the all the detection tests using filename first before reading content.
		language := analyze.GetCodeLanguage(f.Name(), content)
		if language == "" {
			continue
		}

		// group languages, such as Pug -> HTML; SCSS -> CSS
		group := enry.GetLanguageGroup(language)
		if group != "" {
			language = group
		}

		included, checked := includedLanguage[language]
		langType := enry.GetLanguageType(language)
		if !checked {
			included = langType == enry.Programming || langType == enry.Markup
			if !included && (isTrue(isDetectable) || (langType == enry.Prose && isFalse(isDocumentation))) {
				included = true
			}
			includedLanguage[language] = included
		}
		if included {
			sizes[language] += f.Size()
		} else if len(sizes) == 0 && (firstExcludedLanguage == "" || firstExcludedLanguage == language) {
			// Only consider Programming or Markup languages as fallback
			if langType != enry.Programming && langType != enry.Markup {
				continue
			}
			firstExcludedLanguage = language
			firstExcludedLanguageSize += f.Size()
		}
	}

	// If there are no included languages add the first excluded language
	if len(sizes) == 0 && firstExcludedLanguage != "" {
		sizes[firstExcludedLanguage] = firstExcludedLanguageSize
	}

	return mergeLanguageStats(sizes), nil
}
