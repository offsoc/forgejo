// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/log"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/util"

	"github.com/gobwas/glob"
	"github.com/huandu/xstrings"
)

type transformer struct {
	Name      string
	Transform func(string) string
}

type expansion struct {
	Name         string
	Value        string
	Transformers []transformer
}

var defaultTransformers = []transformer{
	{Name: "SNAKE", Transform: xstrings.ToSnakeCase},
	{Name: "KEBAB", Transform: xstrings.ToKebabCase},
	{Name: "CAMEL", Transform: xstrings.ToCamelCase},
	{Name: "PASCAL", Transform: xstrings.ToPascalCase},
	{Name: "LOWER", Transform: strings.ToLower},
	{Name: "UPPER", Transform: strings.ToUpper},
	{Name: "TITLE", Transform: util.ToTitleCase},
}

func generateExpansion(src string, templateRepo, generateRepo *repo_model.Repository, sanitizeFileName bool) string {
	year, month, day := time.Now().Date()
	expansions := []expansion{
		{Name: "YEAR", Value: strconv.Itoa(year), Transformers: nil},
		{Name: "MONTH", Value: fmt.Sprintf("%02d", int(month)), Transformers: nil},
		{Name: "MONTH_ENGLISH", Value: month.String(), Transformers: defaultTransformers},
		{Name: "DAY", Value: fmt.Sprintf("%02d", day), Transformers: nil},
		{Name: "REPO_NAME", Value: generateRepo.Name, Transformers: defaultTransformers},
		{Name: "TEMPLATE_NAME", Value: templateRepo.Name, Transformers: defaultTransformers},
		{Name: "REPO_DESCRIPTION", Value: generateRepo.Description, Transformers: nil},
		{Name: "TEMPLATE_DESCRIPTION", Value: templateRepo.Description, Transformers: nil},
		{Name: "REPO_OWNER", Value: generateRepo.OwnerName, Transformers: defaultTransformers},
		{Name: "TEMPLATE_OWNER", Value: templateRepo.OwnerName, Transformers: defaultTransformers},
		{Name: "REPO_LINK", Value: generateRepo.Link(), Transformers: nil},
		{Name: "TEMPLATE_LINK", Value: templateRepo.Link(), Transformers: nil},
		{Name: "REPO_HTTPS_URL", Value: generateRepo.CloneLink().HTTPS, Transformers: nil},
		{Name: "TEMPLATE_HTTPS_URL", Value: templateRepo.CloneLink().HTTPS, Transformers: nil},
		{Name: "REPO_SSH_URL", Value: generateRepo.CloneLink().SSH, Transformers: nil},
		{Name: "TEMPLATE_SSH_URL", Value: templateRepo.CloneLink().SSH, Transformers: nil},
	}

	expansionMap := make(map[string]string)
	for _, e := range expansions {
		expansionMap[e.Name] = e.Value
		for _, tr := range e.Transformers {
			expansionMap[fmt.Sprintf("%s_%s", e.Name, tr.Name)] = tr.Transform(e.Value)
		}
	}

	return os.Expand(src, func(key string) string {
		if expansion, ok := expansionMap[key]; ok {
			if sanitizeFileName {
				return fileNameSanitize(expansion)
			}
			return expansion
		}
		return key
	})
}

// GiteaTemplate holds information about a .gitea/template file
type GiteaTemplate struct {
	Path    string
	Content []byte

	globs []glob.Glob
}

// Globs parses the .gitea/template globs or returns them if they were already parsed
func (gt *GiteaTemplate) Globs() []glob.Glob {
	if gt.globs != nil {
		return gt.globs
	}

	gt.globs = make([]glob.Glob, 0)
	scanner := bufio.NewScanner(bytes.NewReader(gt.Content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		g, err := glob.Compile(line, '/')
		if err != nil {
			log.Info("Invalid glob expression '%s' (skipped): %v", line, err)
			continue
		}
		gt.globs = append(gt.globs, g)
	}
	return gt.globs
}

func checkGiteaTemplate(tmpDir string) (*GiteaTemplate, error) {
	configDirs := []string{".forgejo", ".gitea"}
	var templateFilePath string

	for _, dir := range configDirs {
		candidatePath := filepath.Join(tmpDir, dir, "template")
		if _, err := os.Stat(candidatePath); err == nil {
			templateFilePath = candidatePath
			break
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	if templateFilePath == "" {
		return nil, nil
	}

	content, err := os.ReadFile(templateFilePath)
	if err != nil {
		return nil, err
	}

	return &GiteaTemplate{
		Path:    templateFilePath,
		Content: content,
	}, nil
}

func generateRepoCommit(ctx context.Context, repo, templateRepo, generateRepo *repo_model.Repository, tmpDir string) error {
	commitTimeStr := time.Now().Format(time.RFC3339)
	authorSig := repo.Owner.NewGitSig()

	// Because this may call hooks we should pass in the environment
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME="+authorSig.Name,
		"GIT_AUTHOR_EMAIL="+authorSig.Email,
		"GIT_AUTHOR_DATE="+commitTimeStr,
		"GIT_COMMITTER_NAME="+authorSig.Name,
		"GIT_COMMITTER_EMAIL="+authorSig.Email,
		"GIT_COMMITTER_DATE="+commitTimeStr,
	)

	// Clone to temporary path and do the init commit.
	templateRepoPath := templateRepo.RepoPath()
	if err := git.Clone(ctx, templateRepoPath, tmpDir, git.CloneRepoOptions{
		Depth:  1,
		Branch: templateRepo.DefaultBranch,
	}); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	if err := util.RemoveAll(path.Join(tmpDir, ".git")); err != nil {
		return fmt.Errorf("remove git dir: %w", err)
	}

	// Variable expansion
	gt, err := checkGiteaTemplate(tmpDir)
	if err != nil {
		return fmt.Errorf("checkGiteaTemplate: %w", err)
	}

	if gt != nil {
		if err := util.Remove(gt.Path); err != nil {
			return fmt.Errorf("remove .giteatemplate: %w", err)
		}

		// Avoid walking tree if there are no globs
		if len(gt.Globs()) > 0 {
			tmpDirSlash := strings.TrimSuffix(filepath.ToSlash(tmpDir), "/") + "/"
			if err := filepath.WalkDir(tmpDirSlash, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				if d.IsDir() {
					return nil
				}

				base := strings.TrimPrefix(filepath.ToSlash(path), tmpDirSlash)
				for _, g := range gt.Globs() {
					if g.Match(base) {
						content, err := os.ReadFile(path)
						if err != nil {
							return err
						}

						if err := os.WriteFile(path,
							[]byte(generateExpansion(string(content), templateRepo, generateRepo, false)),
							0o644); err != nil {
							return err
						}

						substPath := filepath.FromSlash(filepath.Join(tmpDirSlash,
							generateExpansion(base, templateRepo, generateRepo, true)))

						// Create parent subdirectories if needed or continue silently if it exists
						if err := os.MkdirAll(filepath.Dir(substPath), 0o755); err != nil {
							return err
						}

						// Substitute filename variables
						if err := os.Rename(path, substPath); err != nil {
							return err
						}

						break
					}
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}

	if err := git.InitRepository(ctx, tmpDir, false, templateRepo.ObjectFormatName); err != nil {
		return err
	}

	repoPath := repo.RepoPath()
	if stdout, _, err := git.NewCommand(ctx, "remote", "add", "origin").AddDynamicArguments(repoPath).
		SetDescription(fmt.Sprintf("generateRepoCommit (git remote add): %s to %s", templateRepoPath, tmpDir)).
		RunStdString(&git.RunOpts{Dir: tmpDir, Env: env}); err != nil {
		log.Error("Unable to add %v as remote origin to temporary repo to %s: stdout %s\nError: %v", repo, tmpDir, stdout, err)
		return fmt.Errorf("git remote add: %w", err)
	}

	// set default branch based on whether it's specified in the newly generated repo or not
	defaultBranch := repo.DefaultBranch
	if strings.TrimSpace(defaultBranch) == "" {
		defaultBranch = templateRepo.DefaultBranch
	}

	return initRepoCommit(ctx, tmpDir, repo, repo.Owner, defaultBranch)
}

func generateGitContent(ctx context.Context, repo, templateRepo, generateRepo *repo_model.Repository) (err error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "gitea-"+repo.Name)
	if err != nil {
		return fmt.Errorf("Failed to create temp dir for repository %s: %w", repo.RepoPath(), err)
	}

	defer func() {
		if err := util.RemoveAll(tmpDir); err != nil {
			log.Error("RemoveAll: %v", err)
		}
	}()

	if err = generateRepoCommit(ctx, repo, templateRepo, generateRepo, tmpDir); err != nil {
		return fmt.Errorf("generateRepoCommit: %w", err)
	}

	// re-fetch repo
	if repo, err = repo_model.GetRepositoryByID(ctx, repo.ID); err != nil {
		return fmt.Errorf("getRepositoryByID: %w", err)
	}

	// if there was no default branch supplied when generating the repo, use the default one from the template
	if strings.TrimSpace(repo.DefaultBranch) == "" {
		repo.DefaultBranch = templateRepo.DefaultBranch
	}

	if err = gitrepo.SetDefaultBranch(ctx, repo, repo.DefaultBranch); err != nil {
		return fmt.Errorf("setDefaultBranch: %w", err)
	}
	if err = UpdateRepository(ctx, repo, false); err != nil {
		return fmt.Errorf("updateRepository: %w", err)
	}

	return nil
}

// GenerateGitContent generates git content from a template repository
func GenerateGitContent(ctx context.Context, templateRepo, generateRepo *repo_model.Repository) error {
	if err := generateGitContent(ctx, generateRepo, templateRepo, generateRepo); err != nil {
		return err
	}

	if err := repo_module.UpdateRepoSize(ctx, generateRepo); err != nil {
		return fmt.Errorf("failed to update size for repository: %w", err)
	}

	if err := git_model.CopyLFS(ctx, generateRepo, templateRepo); err != nil {
		return fmt.Errorf("failed to copy LFS: %w", err)
	}
	return nil
}

// GenerateRepoOptions contains the template units to generate
type GenerateRepoOptions struct {
	Name            string
	DefaultBranch   string
	Description     string
	Private         bool
	GitContent      bool
	Topics          bool
	GitHooks        bool
	Webhooks        bool
	Avatar          bool
	IssueLabels     bool
	ProtectedBranch bool
}

// IsValid checks whether at least one option is chosen for generation
func (gro GenerateRepoOptions) IsValid() bool {
	return gro.GitContent || gro.Topics || gro.GitHooks || gro.Webhooks || gro.Avatar ||
		gro.IssueLabels || gro.ProtectedBranch // or other items as they are added
}

// generateRepository generates a repository from a template
func generateRepository(ctx context.Context, doer, owner *user_model.User, templateRepo *repo_model.Repository, opts GenerateRepoOptions) (_ *repo_model.Repository, err error) {
	generateRepo := &repo_model.Repository{
		OwnerID:          owner.ID,
		Owner:            owner,
		OwnerName:        owner.Name,
		Name:             opts.Name,
		LowerName:        strings.ToLower(opts.Name),
		Description:      opts.Description,
		DefaultBranch:    opts.DefaultBranch,
		IsPrivate:        opts.Private,
		IsEmpty:          !opts.GitContent || templateRepo.IsEmpty,
		IsFsckEnabled:    templateRepo.IsFsckEnabled,
		TemplateID:       templateRepo.ID,
		TrustModel:       templateRepo.TrustModel,
		ObjectFormatName: templateRepo.ObjectFormatName,
	}

	if err = repo_module.CreateRepositoryByExample(ctx, doer, owner, generateRepo, false, false); err != nil {
		return nil, err
	}

	repoPath := generateRepo.RepoPath()
	isExist, err := util.IsExist(repoPath)
	if err != nil {
		log.Error("Unable to check if %s exists. Error: %v", repoPath, err)
		return nil, err
	}
	if isExist {
		return nil, repo_model.ErrRepoFilesAlreadyExist{
			Uname: generateRepo.OwnerName,
			Name:  generateRepo.Name,
		}
	}

	if err = repo_module.CheckInitRepository(ctx, owner.Name, generateRepo.Name, generateRepo.ObjectFormatName); err != nil {
		return generateRepo, err
	}

	if err = repo_module.CheckDaemonExportOK(ctx, generateRepo); err != nil {
		return generateRepo, fmt.Errorf("checkDaemonExportOK: %w", err)
	}

	if stdout, _, err := git.NewCommand(ctx, "update-server-info").
		SetDescription(fmt.Sprintf("GenerateRepository(git update-server-info): %s", repoPath)).
		RunStdString(&git.RunOpts{Dir: repoPath}); err != nil {
		log.Error("GenerateRepository(git update-server-info) in %v: Stdout: %s\nError: %v", generateRepo, stdout, err)
		return generateRepo, fmt.Errorf("error in GenerateRepository(git update-server-info): %w", err)
	}

	return generateRepo, nil
}

var fileNameSanitizeRegexp = regexp.MustCompile(`(?i)\.\.|[<>:\"/\\|?*\x{0000}-\x{001F}]|^(con|prn|aux|nul|com\d|lpt\d)$`)

// Sanitize user input to valid OS filenames
//
//		Based on https://github.com/sindresorhus/filename-reserved-regex
//	 Adds ".." to prevent directory traversal
func fileNameSanitize(s string) string {
	return strings.TrimSpace(fileNameSanitizeRegexp.ReplaceAllString(s, "_"))
}
