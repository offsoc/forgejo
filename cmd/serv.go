// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	asymkey_model "forgejo.org/models/asymkey"
	git_model "forgejo.org/models/git"
	"forgejo.org/models/perm"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/pprof"
	"forgejo.org/modules/private"
	"forgejo.org/modules/process"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/services/lfs"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v3"
)

const (
	lfsAuthenticateVerb = "git-lfs-authenticate"
)

// CmdServ represents the available serv sub-command.
func cmdServ() *cli.Command {
	return &cli.Command{
		Name:        "serv",
		Usage:       "(internal) Should only be called by SSH shell",
		Description: "Serv provides access auth for repositories",
		Before:      PrepareConsoleLoggerLevel(log.FATAL),
		Action:      runServ,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "enable-pprof",
			},
			&cli.BoolFlag{
				Name: "debug",
			},
		},
	}
}

func setup(ctx context.Context, debug, gitNeeded bool) {
	if debug {
		setupConsoleLogger(log.TRACE, false, os.Stderr)
	} else {
		setupConsoleLogger(log.FATAL, false, os.Stderr)
	}
	setting.MustInstalled()
	// Sanity check to ensure path is not relative, see: https://github.com/go-gitea/gitea/pull/19317
	if _, err := os.Stat(setting.RepoRootPath); err != nil {
		_ = fail(ctx, "Unable to access repository path", "Unable to access repository path %q, err: %v", setting.RepoRootPath, err)
		return
	}
	if gitNeeded {
		if err := git.InitSimple(context.Background()); err != nil {
			_ = fail(ctx, "Failed to init git", "Failed to init git, err: %v", err)
		}
	}
}

var (
	allowedCommands = map[string]perm.AccessMode{
		"git-upload-pack":    perm.AccessModeRead,
		"git-upload-archive": perm.AccessModeRead,
		"git-receive-pack":   perm.AccessModeWrite,
		lfsAuthenticateVerb:  perm.AccessModeNone,
	}
	alphaDashDotPattern = regexp.MustCompile(`[^\w-\.]`)
)

// fail prints message to stdout, it's mainly used for git serv and git hook commands.
// The output will be passed to git client and shown to user.
func fail(ctx context.Context, userMessage, logMsgFmt string, args ...any) error {
	if userMessage == "" {
		userMessage = "Internal Server Error (no specific error)"
	}

	// There appears to be a chance to cause a zombie process and failure to read the Exit status
	// if nothing is outputted on stdout.
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stderr, "Forgejo:", userMessage)

	if logMsgFmt != "" {
		logMsg := fmt.Sprintf(logMsgFmt, args...)
		if !setting.IsProd {
			_, _ = fmt.Fprintln(os.Stderr, "Forgejo:", logMsg)
		}
		if userMessage != "" {
			if unicode.IsPunct(rune(userMessage[len(userMessage)-1])) {
				logMsg = userMessage + " " + logMsg
			} else {
				logMsg = userMessage + ". " + logMsg
			}
		}
		// Don't send an log if this is done in a test and no InternalToken is set.
		if !testing.Testing() || setting.InternalToken != "" {
			_ = private.SSHLog(ctx, true, logMsg)
		}
	}
	return cli.Exit("", 1)
}

// handleCliResponseExtra handles the extra response from the cli sub-commands
// If there is a user message it will be printed to stdout
// If the command failed it will return an error (the error will be printed by cli framework)
func handleCliResponseExtra(extra private.ResponseExtra) error {
	if extra.UserMsg != "" {
		_, _ = fmt.Fprintln(os.Stdout, extra.UserMsg)
	}
	if extra.HasError() {
		return cli.Exit(extra.Error, 1)
	}
	return nil
}

func runServ(ctx context.Context, c *cli.Command) error {
	ctx, cancel := installSignals(ctx)
	defer cancel()

	// FIXME: This needs to internationalised
	setup(ctx, c.Bool("debug"), true)

	if setting.SSH.Disabled {
		fmt.Println("Forgejo: SSH has been disabled")
		return nil
	}

	if c.NArg() < 1 {
		if err := cli.ShowSubcommandHelp(c); err != nil {
			fmt.Printf("error showing subcommand help: %v\n", err)
		}
		return nil
	}

	defer func() {
		if err := recover(); err != nil {
			_ = fail(ctx, "Internal Server Error", "Panic: %v\n%s", err, log.Stack(2))
		}
	}()

	keys := strings.Split(c.Args().First(), "-")
	if len(keys) != 2 || keys[0] != "key" {
		return fail(ctx, "Key ID format error", "Invalid key argument: %s", c.Args().First())
	}
	keyID, err := strconv.ParseInt(keys[1], 10, 64)
	if err != nil {
		return fail(ctx, "Key ID parsing error", "Invalid key argument: %s", c.Args().Get(1))
	}

	cmd := os.Getenv("SSH_ORIGINAL_COMMAND")
	if len(cmd) == 0 {
		key, user, err := private.ServNoCommand(ctx, keyID)
		if err != nil {
			return fail(ctx, "Key check failed", "Failed to check provided key: %v", err)
		}
		switch key.Type {
		case asymkey_model.KeyTypeDeploy:
			fmt.Println("Hi there! You've successfully authenticated with the deploy key named " + key.Name + ", but Forgejo does not provide shell access.")
		case asymkey_model.KeyTypePrincipal:
			fmt.Println("Hi there! You've successfully authenticated with the principal " + key.Content + ", but Forgejo does not provide shell access.")
		default:
			fmt.Println("Hi there, " + user.Name + "! You've successfully authenticated with the key named " + key.Name + ", but Forgejo does not provide shell access.")
		}
		fmt.Println("If this is unexpected, please log in with password and setup Forgejo under another user.")
		return nil
	} else if c.Bool("debug") {
		log.Debug("SSH_ORIGINAL_COMMAND: %s", os.Getenv("SSH_ORIGINAL_COMMAND"))
	}

	words, err := shellquote.Split(cmd)
	if err != nil {
		return fail(ctx, "Error parsing arguments", "Failed to parse arguments: %v", err)
	}

	if len(words) < 2 {
		if git.CheckGitVersionAtLeast("2.29") == nil {
			// for AGit Flow
			if cmd == "ssh_info" {
				fmt.Print(`{"type":"agit","version":1}`)
				return nil
			}
		}
		return fail(ctx, "Too few arguments", "Too few arguments in cmd: %s", cmd)
	}

	verb := words[0]
	repoPath := strings.TrimPrefix(words[1], "/")

	var lfsVerb string
	if verb == lfsAuthenticateVerb {
		if !setting.LFS.StartServer {
			return fail(ctx, "Unknown git command", "LFS authentication request over SSH denied, LFS support is disabled")
		}

		if len(words) > 2 {
			lfsVerb = words[2]
		}
	}

	rr := strings.SplitN(repoPath, "/", 2)
	if len(rr) != 2 {
		return fail(ctx, "Invalid repository path", "Invalid repository path: %v", repoPath)
	}

	username := rr[0]
	reponame := strings.TrimSuffix(rr[1], ".git")

	// LowerCase and trim the repoPath as that's how they are stored.
	// This should be done after splitting the repoPath into username and reponame
	// so that username and reponame are not affected.
	repoPath = strings.ToLower(strings.TrimSpace(repoPath))

	if alphaDashDotPattern.MatchString(reponame) {
		return fail(ctx, "Invalid repo name", "Invalid repo name: %s", reponame)
	}

	if c.Bool("enable-pprof") {
		if err := os.MkdirAll(setting.PprofDataPath, os.ModePerm); err != nil {
			return fail(ctx, "Error while trying to create PPROF_DATA_PATH", "Error while trying to create PPROF_DATA_PATH: %v", err)
		}

		stopCPUProfiler, err := pprof.DumpCPUProfileForUsername(setting.PprofDataPath, username)
		if err != nil {
			return fail(ctx, "Unable to start CPU profiler", "Unable to start CPU profile: %v", err)
		}
		defer func() {
			stopCPUProfiler()
			err := pprof.DumpMemProfileForUsername(setting.PprofDataPath, username)
			if err != nil {
				_ = fail(ctx, "Unable to dump Mem profile", "Unable to dump Mem Profile: %v", err)
			}
		}()
	}

	requestedMode, has := allowedCommands[verb]
	if !has {
		return fail(ctx, "Unknown git command", "Unknown git command %s", verb)
	}

	if verb == lfsAuthenticateVerb {
		switch lfsVerb {
		case "upload":
			requestedMode = perm.AccessModeWrite
		case "download":
			requestedMode = perm.AccessModeRead
		default:
			return fail(ctx, "Unknown LFS verb", "Unknown lfs verb %s", lfsVerb)
		}
	}

	results, extra := private.ServCommand(ctx, keyID, username, reponame, requestedMode, verb, lfsVerb)
	if extra.HasError() {
		return fail(ctx, extra.UserMsg, "ServCommand failed: %s", extra.Error)
	}

	// LFS token authentication
	if verb == lfsAuthenticateVerb {
		url := fmt.Sprintf("%s%s/%s.git/info/lfs", setting.AppURL, url.PathEscape(results.OwnerName), url.PathEscape(results.RepoName))

		now := time.Now()
		claims := lfs.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(setting.LFS.HTTPAuthExpiry)),
				NotBefore: jwt.NewNumericDate(now),
			},
			RepoID: results.RepoID,
			Op:     lfsVerb,
			UserID: results.UserID,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString(setting.LFS.JWTSecretBytes)
		if err != nil {
			return fail(ctx, "Failed to sign JWT Token", "Failed to sign JWT token: %v", err)
		}

		tokenAuthentication := &git_model.LFSTokenResponse{
			Header: make(map[string]string),
			Href:   url,
		}
		tokenAuthentication.Header["Authorization"] = fmt.Sprintf("Bearer %s", tokenString)

		enc := json.NewEncoder(os.Stdout)
		err = enc.Encode(tokenAuthentication)
		if err != nil {
			return fail(ctx, "Failed to encode LFS json response", "Failed to encode LFS json response: %v", err)
		}
		return nil
	}

	var gitcmd *exec.Cmd
	gitBinPath := filepath.Dir(git.GitExecutable) // e.g. /usr/bin
	gitBinVerb := filepath.Join(gitBinPath, verb) // e.g. /usr/bin/git-upload-pack
	if _, err := os.Stat(gitBinVerb); err != nil {
		// if the command "git-upload-pack" doesn't exist, try to split "git-upload-pack" to use the sub-command with git
		// ps: Windows only has "git.exe" in the bin path, so Windows always uses this way
		verbFields := strings.SplitN(verb, "-", 2)
		if len(verbFields) == 2 {
			// use git binary with the sub-command part: "C:\...\bin\git.exe", "upload-pack", ...
			gitcmd = exec.CommandContext(ctx, git.GitExecutable, verbFields[1], repoPath)
		}
	}
	if gitcmd == nil {
		// by default, use the verb (it has been checked above by allowedCommands)
		gitcmd = exec.CommandContext(ctx, gitBinVerb, repoPath)
	}

	process.SetSysProcAttribute(gitcmd)
	gitcmd.Dir = setting.RepoRootPath
	gitcmd.Stdout = os.Stdout
	gitcmd.Stdin = os.Stdin
	gitcmd.Stderr = os.Stderr
	gitcmd.Env = append(gitcmd.Env, os.Environ()...)
	gitcmd.Env = append(gitcmd.Env,
		repo_module.EnvRepoIsWiki+"="+strconv.FormatBool(results.IsWiki),
		repo_module.EnvRepoName+"="+results.RepoName,
		repo_module.EnvRepoUsername+"="+results.OwnerName,
		repo_module.EnvPusherName+"="+results.UserName,
		repo_module.EnvPusherEmail+"="+results.UserEmail,
		repo_module.EnvPusherID+"="+strconv.FormatInt(results.UserID, 10),
		repo_module.EnvRepoID+"="+strconv.FormatInt(results.RepoID, 10),
		repo_module.EnvPRID+"="+fmt.Sprintf("%d", 0),
		repo_module.EnvDeployKeyID+"="+fmt.Sprintf("%d", results.DeployKeyID),
		repo_module.EnvKeyID+"="+fmt.Sprintf("%d", results.KeyID),
		repo_module.EnvAppURL+"="+setting.AppURL,
	)
	// to avoid breaking, here only use the minimal environment variables for the "gitea serv" command.
	// it could be re-considered whether to use the same git.CommonGitCmdEnvs() as "git" command later.
	gitcmd.Env = append(gitcmd.Env, git.CommonCmdServEnvs()...)

	if err = gitcmd.Run(); err != nil {
		return fail(ctx, "Failed to execute git command", "Failed to execute git command: %v", err)
	}

	// Update user key activity.
	if results.KeyID > 0 {
		if err = private.UpdatePublicKeyInRepo(ctx, results.KeyID, results.RepoID); err != nil {
			return fail(ctx, "Failed to update public key", "UpdatePublicKeyInRepo: %v", err)
		}
	}

	return nil
}
