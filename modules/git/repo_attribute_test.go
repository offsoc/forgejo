// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCheckAttrStdoutReader(t *testing.T) {
	t.Run("two_times", func(t *testing.T) {
		read := newCheckAttrStdoutReader(strings.NewReader(
			".gitignore\x00linguist-vendored\x00unspecified\x00"+
				".gitignore\x00linguist-vendored\x00specified",
		), 1)

		// first read
		attr, err := read()
		require.NoError(t, err)
		assert.Equal(t, map[string]GitAttribute{
			"linguist-vendored": GitAttribute("unspecified"),
		}, attr)

		// second read
		attr, err = read()
		require.NoError(t, err)
		assert.Equal(t, map[string]GitAttribute{
			"linguist-vendored": GitAttribute("specified"),
		}, attr)
	})
	t.Run("incomplete", func(t *testing.T) {
		read := newCheckAttrStdoutReader(strings.NewReader(
			"filename\x00linguist-vendored",
		), 1)

		_, err := read()
		assert.Equal(t, io.ErrUnexpectedEOF, err)
	})
	t.Run("three_times", func(t *testing.T) {
		read := newCheckAttrStdoutReader(strings.NewReader(
			"shouldbe.vendor\x00linguist-vendored\x00set\x00"+
				"shouldbe.vendor\x00linguist-generated\x00unspecified\x00"+
				"shouldbe.vendor\x00linguist-language\x00unspecified\x00",
		), 1)

		// first read
		attr, err := read()
		require.NoError(t, err)
		assert.Equal(t, map[string]GitAttribute{
			"linguist-vendored": GitAttribute("set"),
		}, attr)

		// second read
		attr, err = read()
		require.NoError(t, err)
		assert.Equal(t, map[string]GitAttribute{
			"linguist-generated": GitAttribute("unspecified"),
		}, attr)

		// third read
		attr, err = read()
		require.NoError(t, err)
		assert.Equal(t, map[string]GitAttribute{
			"linguist-language": GitAttribute("unspecified"),
		}, attr)
	})
}

func TestGitAttributeBareNonBare(t *testing.T) {
	if !SupportCheckAttrOnBare {
		t.Skip("git check-attr supported on bare repo starting with git 2.40")
	}

	repoPath := filepath.Join(testReposDir, "language_stats_repo")
	gitRepo, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer gitRepo.Close()

	for _, commitID := range []string{
		"8fee858da5796dfb37704761701bb8e800ad9ef3",
		"341fca5b5ea3de596dc483e54c2db28633cd2f97",
	} {
		bareStats, err := gitRepo.GitAttributes(commitID, "i-am-a-python.p", LinguistAttributes...)
		require.NoError(t, err)

		defer test.MockVariableValue(&SupportCheckAttrOnBare, false)()
		cloneStats, err := gitRepo.GitAttributes(commitID, "i-am-a-python.p", LinguistAttributes...)
		require.NoError(t, err)

		assert.Equal(t, cloneStats, bareStats)
		refStats := cloneStats

		t.Run("GitAttributeChecker/"+commitID+"/SupportBare", func(t *testing.T) {
			bareChecker, err := gitRepo.GitAttributeChecker(commitID, LinguistAttributes...)
			require.NoError(t, err)
			defer bareChecker.Close()

			bareStats, err := bareChecker.CheckPath("i-am-a-python.p")
			require.NoError(t, err)
			assert.Equal(t, refStats, bareStats)
		})
		t.Run("GitAttributeChecker/"+commitID+"/NoBareSupport", func(t *testing.T) {
			defer test.MockVariableValue(&SupportCheckAttrOnBare, false)()
			cloneChecker, err := gitRepo.GitAttributeChecker(commitID, LinguistAttributes...)
			require.NoError(t, err)
			defer cloneChecker.Close()

			cloneStats, err := cloneChecker.CheckPath("i-am-a-python.p")
			require.NoError(t, err)

			assert.Equal(t, refStats, cloneStats)
		})
	}
}

func TestGitAttributes(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "language_stats_repo")
	gitRepo, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer gitRepo.Close()

	attr, err := gitRepo.GitAttributes("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", LinguistAttributes...)
	require.NoError(t, err)
	assert.Equal(t, map[string]GitAttribute{
		"gitlab-language":        "unspecified",
		"linguist-detectable":    "unspecified",
		"linguist-documentation": "unspecified",
		"linguist-generated":     "unspecified",
		"linguist-language":      "Python",
		"linguist-vendored":      "unspecified",
	}, attr)

	attr, err = gitRepo.GitAttributes("341fca5b5ea3de596dc483e54c2db28633cd2f97", "i-am-a-python.p", LinguistAttributes...)
	require.NoError(t, err)
	assert.Equal(t, map[string]GitAttribute{
		"gitlab-language":        "unspecified",
		"linguist-detectable":    "unspecified",
		"linguist-documentation": "unspecified",
		"linguist-generated":     "unspecified",
		"linguist-language":      "Cobra",
		"linguist-vendored":      "unspecified",
	}, attr)
}

func TestGitAttributeFirst(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "language_stats_repo")
	gitRepo, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer gitRepo.Close()

	t.Run("first is specified", func(t *testing.T) {
		language, err := gitRepo.GitAttributeFirst("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", "linguist-language", "gitlab-language")
		require.NoError(t, err)
		assert.Equal(t, "Python", language.String())
	})

	t.Run("second is specified", func(t *testing.T) {
		language, err := gitRepo.GitAttributeFirst("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", "gitlab-language", "linguist-language")
		require.NoError(t, err)
		assert.Equal(t, "Python", language.String())
	})

	t.Run("none is specified", func(t *testing.T) {
		language, err := gitRepo.GitAttributeFirst("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", "linguist-detectable", "gitlab-language", "non-existing")
		require.NoError(t, err)
		assert.Empty(t, language.String())
	})
}

func TestGitAttributeStruct(t *testing.T) {
	assert.Empty(t, GitAttribute("").String())
	assert.Empty(t, GitAttribute("unspecified").String())

	assert.Equal(t, "python", GitAttribute("python").String())

	assert.Equal(t, "text?token=Error", GitAttribute("text?token=Error").String())
	assert.Equal(t, "text", GitAttribute("text?token=Error").Prefix())
}

func TestGitAttributeCheckerError(t *testing.T) {
	prepareRepo := func(t *testing.T) *Repository {
		t.Helper()
		path := t.TempDir()

		// we can't use unittest.CopyDir because of an import cycle (git.Init in unittest)
		require.NoError(t, CopyFS(path, os.DirFS(filepath.Join(testReposDir, "language_stats_repo"))))

		gitRepo, err := openRepositoryWithDefaultContext(path)
		require.NoError(t, err)
		return gitRepo
	}

	t.Run("RemoveAll/BeforeRun", func(t *testing.T) {
		gitRepo := prepareRepo(t)
		defer gitRepo.Close()

		require.NoError(t, os.RemoveAll(gitRepo.Path))

		ac, err := gitRepo.GitAttributeChecker("", "linguist-language")
		require.NoError(t, err)

		_, err = ac.CheckPath("i-am-a-python.p")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `git check-attr (stderr: ""):`)
	})

	t.Run("RemoveAll/DuringRun", func(t *testing.T) {
		gitRepo := prepareRepo(t)
		defer gitRepo.Close()

		ac, err := gitRepo.GitAttributeChecker("", "linguist-language")
		require.NoError(t, err)

		// calling CheckPath before would allow git to cache part of it and successfully return later
		require.NoError(t, os.RemoveAll(gitRepo.Path))

		_, err = ac.CheckPath("i-am-a-python.p")
		if err == nil {
			t.Skip(
				"git check-attr started too fast and CheckPath was successful (and likely cached)",
				"https://codeberg.org/forgejo/forgejo/issues/2948",
			)
		}
		// Depending on the order of execution, the returned error can be:
		// - a launch error "fork/exec /usr/bin/git: no such file or directory" (when the removal happens before the Run)
		// - a git error (stderr: "fatal: Unable to read current working directory: No such file or directory"): exit status 128 (when the removal happens after the Run)
		// (pipe error "write |1: broken pipe" should be replaced by one of the Run errors above)
		assert.Contains(t, err.Error(), `git check-attr`)
	})

	t.Run("Cancelled/BeforeRun", func(t *testing.T) {
		gitRepo := prepareRepo(t)
		defer gitRepo.Close()

		var cancel context.CancelFunc
		gitRepo.Ctx, cancel = context.WithCancel(gitRepo.Ctx)
		cancel()

		ac, err := gitRepo.GitAttributeChecker("8fee858da5796dfb37704761701bb8e800ad9ef3", "linguist-language")
		require.NoError(t, err)

		_, err = ac.CheckPath("i-am-a-python.p")
		require.Error(t, err)
	})

	t.Run("Cancelled/DuringRun", func(t *testing.T) {
		gitRepo := prepareRepo(t)
		defer gitRepo.Close()

		var cancel context.CancelFunc
		gitRepo.Ctx, cancel = context.WithCancel(gitRepo.Ctx)

		ac, err := gitRepo.GitAttributeChecker("8fee858da5796dfb37704761701bb8e800ad9ef3", "linguist-language")
		require.NoError(t, err)

		attr, err := ac.CheckPath("i-am-a-python.p")
		require.NoError(t, err)
		assert.Equal(t, "Python", attr["linguist-language"].String())

		errCh := make(chan error)
		go func() {
			cancel()

			for err == nil {
				_, err = ac.CheckPath("i-am-a-python.p")
				runtime.Gosched() // the cancellation must have time to propagate
			}
			errCh <- err
		}()

		select {
		case <-time.After(time.Second):
			t.Error("CheckPath did not complete within 1s")
		case err = <-errCh:
			require.ErrorIs(t, err, context.Canceled)
		}
	})

	t.Run("Closed/BeforeRun", func(t *testing.T) {
		gitRepo := prepareRepo(t)
		defer gitRepo.Close()

		ac, err := gitRepo.GitAttributeChecker("8fee858da5796dfb37704761701bb8e800ad9ef3", "linguist-language")
		require.NoError(t, err)

		require.NoError(t, ac.Close())

		_, err = ac.CheckPath("i-am-a-python.p")
		require.ErrorIs(t, err, fs.ErrClosed)
	})

	t.Run("Closed/DuringRun", func(t *testing.T) {
		gitRepo := prepareRepo(t)
		defer gitRepo.Close()

		ac, err := gitRepo.GitAttributeChecker("8fee858da5796dfb37704761701bb8e800ad9ef3", "linguist-language")
		require.NoError(t, err)

		attr, err := ac.CheckPath("i-am-a-python.p")
		require.NoError(t, err)
		assert.Equal(t, "Python", attr["linguist-language"].String())

		require.NoError(t, ac.Close())

		_, err = ac.CheckPath("i-am-a-python.p")
		require.ErrorIs(t, err, fs.ErrClosed)
	})
}

// CopyFS is adapted from https://github.com/golang/go/issues/62484
// which should be available with go1.23
func CopyFS(dir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, _ error) error {
		targ := filepath.Join(dir, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(targ, 0o777)
		}
		r, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		info, err := r.Stat()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(targ, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666|info.Mode()&0o777)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, r); err != nil {
			w.Close()
			return fmt.Errorf("copying %s: %v", path, err)
		}
		return w.Close()
	})
}
