// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"compress/gzip"
	gocontext "context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/services/context"

	"github.com/go-chi/cors"
)

func HTTPGitEnabledHandler(ctx *context.Context) {
	if setting.Repository.DisableHTTPGit {
		ctx.Resp.WriteHeader(http.StatusForbidden)
		_, _ = ctx.Resp.Write([]byte("Interacting with repositories by HTTP protocol is not allowed"))
	}
}

func CorsHandler() func(next http.Handler) http.Handler {
	if setting.Repository.AccessControlAllowOrigin != "" {
		return cors.Handler(cors.Options{
			AllowedOrigins: []string{setting.Repository.AccessControlAllowOrigin},
			AllowedHeaders: []string{"Content-Type", "Authorization", "User-Agent"},
		})
	}
	return func(next http.Handler) http.Handler {
		return next
	}
}

// httpBase implementation git smart HTTP protocol
func httpBase(ctx *context.Context) serviceHandlerBase {
	var isPull, receivePack bool
	service := ctx.FormString("service")
	if service == "git-receive-pack" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-receive-pack") {
		isPull = false
		receivePack = true
	} else if service == "git-upload-pack" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-upload-pack") {
		isPull = true
	} else if service == "git-upload-archive" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-upload-archive") {
		isPull = true
	} else {
		isPull = ctx.Req.Method == "GET"
	}

	var handler serviceHandlerBase
	if ctx.Params(":gistuuid") != "" {
		handler = new(serviceHandlerGist)
	} else {
		handler = new(serviceHandlerRepo)
	}

	ok := handler.Init(ctx, isPull, receivePack)
	if !ok {
		return nil
	}

	return handler
}

var (
	infoRefsCache []byte
	infoRefsOnce  sync.Once
)

func dummyInfoRefs(ctx *context.Context) {
	infoRefsOnce.Do(func() {
		tmpDir, err := os.MkdirTemp(os.TempDir(), "gitea-info-refs-cache")
		if err != nil {
			log.Error("Failed to create temp dir for git-receive-pack cache: %v", err)
			return
		}

		defer func() {
			if err := util.RemoveAll(tmpDir); err != nil {
				log.Error("RemoveAll: %v", err)
			}
		}()

		if err := git.InitRepository(ctx, tmpDir, git.InitRepositoryOptions{Bare: true, ObjectFormatName: git.Sha1ObjectFormat.Name()}); err != nil {
			log.Error("Failed to init bare repo for git-receive-pack cache: %v", err)
			return
		}

		refs, _, err := git.NewCommand(ctx, "receive-pack", "--stateless-rpc", "--advertise-refs", ".").RunStdBytes(&git.RunOpts{Dir: tmpDir})
		if err != nil {
			log.Error(fmt.Sprintf("%v - %s", err, string(refs)))
		}

		log.Debug("populating infoRefsCache: \n%s", string(refs))
		infoRefsCache = refs
	})

	ctx.RespHeader().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	ctx.RespHeader().Set("Pragma", "no-cache")
	ctx.RespHeader().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	ctx.RespHeader().Set("Content-Type", "application/x-git-receive-pack-advertisement")
	_, _ = ctx.Write(packetWrite("# service=git-receive-pack\n"))
	_, _ = ctx.Write([]byte("0000"))
	_, _ = ctx.Write(infoRefsCache)
}

func setHeaderNoCache(ctx *context.Context) {
	ctx.Resp.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	ctx.Resp.Header().Set("Pragma", "no-cache")
	ctx.Resp.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func setHeaderCacheForever(ctx *context.Context) {
	now := time.Now().Unix()
	expires := now + 31536000
	ctx.Resp.Header().Set("Date", fmt.Sprintf("%d", now))
	ctx.Resp.Header().Set("Expires", fmt.Sprintf("%d", expires))
	ctx.Resp.Header().Set("Cache-Control", "public, max-age=31536000")
}

func containsParentDirectorySeparator(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

func sendFile(ctx *context.Context, h serviceHandlerBase, contentType, file string) {
	if containsParentDirectorySeparator(file) {
		log.Error("request file path contains invalid path: %v", file)
		ctx.Resp.WriteHeader(http.StatusBadRequest)
		return
	}
	reqFile := filepath.Join(h.GetRepoPath(), file)

	fi, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		ctx.Resp.WriteHeader(http.StatusNotFound)
		return
	}

	ctx.Resp.Header().Set("Content-Type", contentType)
	ctx.Resp.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	// http.TimeFormat required a UTC time, refer to https://pkg.go.dev/net/http#TimeFormat
	ctx.Resp.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
	http.ServeFile(ctx.Resp, ctx.Req, reqFile)
}

// one or more key=value pairs separated by colons
var safeGitProtocolHeader = regexp.MustCompile(`^[0-9a-zA-Z]+=[0-9a-zA-Z]+(:[0-9a-zA-Z]+=[0-9a-zA-Z]+)*$`)

func prepareGitCmdWithAllowedService(ctx *context.Context, service string) (*git.Command, error) {
	if service == "receive-pack" {
		return git.NewCommand(ctx, "receive-pack"), nil
	}
	if service == "upload-pack" {
		return git.NewCommand(ctx, "upload-pack"), nil
	}

	return nil, fmt.Errorf("service %q is not allowed", service)
}

func serviceRPC(ctx *context.Context, h serviceHandlerBase, service string) {
	defer func() {
		if err := ctx.Req.Body.Close(); err != nil {
			log.Error("serviceRPC: Close: %v", err)
		}
	}()

	expectedContentType := fmt.Sprintf("application/x-git-%s-request", service)
	if ctx.Req.Header.Get("Content-Type") != expectedContentType {
		log.Error("Content-Type (%q) doesn't match expected: %q", ctx.Req.Header.Get("Content-Type"), expectedContentType)
		ctx.Resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	cmd, err := prepareGitCmdWithAllowedService(ctx, service)
	if err != nil {
		log.Error("Failed to prepareGitCmdWithService: %v", err)
		ctx.Resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	ctx.Resp.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

	reqBody := ctx.Req.Body

	// Handle GZIP.
	if ctx.Req.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			log.Error("Fail to create gzip reader: %v", err)
			ctx.Resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	environ := h.GetEnviron()

	// set this for allow pre-receive and post-receive execute
	environ = append(environ, "SSH_ORIGINAL_COMMAND="+service)

	if protocol := ctx.Req.Header.Get("Git-Protocol"); protocol != "" && safeGitProtocolHeader.MatchString(protocol) {
		environ = append(environ, "GIT_PROTOCOL="+protocol)
	}

	var stderr bytes.Buffer
	cmd.AddArguments("--stateless-rpc").AddDynamicArguments(h.GetRepoPath())
	cmd.SetDescription(fmt.Sprintf("%s %s %s [repo_path: %s]", git.GitExecutable, service, "--stateless-rpc", h.GetRepoPath()))
	if err := cmd.Run(&git.RunOpts{
		Dir:               h.GetRepoPath(),
		Env:               append(os.Environ(), environ...),
		Stdout:            ctx.Resp,
		Stdin:             reqBody,
		Stderr:            &stderr,
		UseContextTimeout: true,
	}); err != nil {
		if !git.IsErrCanceledOrKilled(err) {
			log.Error("Fail to serve RPC(%s) in %s: %v - %s", service, h.GetRepoPath(), err, stderr.String())
		}
		return
	}
}

// ServiceUploadPack implements Git Smart HTTP protocol
func ServiceUploadPack(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		serviceRPC(ctx, h, "upload-pack")
	}
}

// ServiceReceivePack implements Git Smart HTTP protocol
func ServiceReceivePack(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		serviceRPC(ctx, h, "receive-pack")
	}
}

func getServiceType(ctx *context.Context) string {
	serviceType := ctx.Req.FormValue("service")
	if !strings.HasPrefix(serviceType, "git-") {
		return ""
	}
	return strings.TrimPrefix(serviceType, "git-")
}

func updateServerInfo(ctx gocontext.Context, dir string) []byte {
	out, _, err := git.NewCommand(ctx, "update-server-info").RunStdBytes(&git.RunOpts{Dir: dir})
	if err != nil {
		log.Error(fmt.Sprintf("%v - %s", err, string(out)))
	}
	return out
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}

// GetInfoRefs implements Git dumb HTTP
func GetInfoRefs(ctx *context.Context) {
	h := httpBase(ctx)
	if h == nil {
		return
	}

	environ := h.GetEnviron()

	setHeaderNoCache(ctx)
	service := getServiceType(ctx)
	cmd, err := prepareGitCmdWithAllowedService(ctx, service)
	if err == nil {
		if protocol := ctx.Req.Header.Get("Git-Protocol"); protocol != "" && safeGitProtocolHeader.MatchString(protocol) {
			environ = append(environ, "GIT_PROTOCOL="+protocol)
		}
		environ = append(os.Environ(), environ...)

		refs, _, err := cmd.AddArguments("--stateless-rpc", "--advertise-refs", ".").RunStdBytes(&git.RunOpts{Env: environ, Dir: h.GetRepoPath()})
		if err != nil {
			log.Error(fmt.Sprintf("%v - %s", err, string(refs)))
		}

		ctx.Resp.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
		ctx.Resp.WriteHeader(http.StatusOK)
		_, _ = ctx.Resp.Write(packetWrite("# service=git-" + service + "\n"))
		_, _ = ctx.Resp.Write([]byte("0000"))
		_, _ = ctx.Resp.Write(refs)
	} else {
		updateServerInfo(ctx, h.GetRepoPath())
		sendFile(ctx, h, "text/plain; charset=utf-8", "info/refs")
	}
}

// GetTextFile implements Git dumb HTTP
func GetTextFile(p string) func(*context.Context) {
	return func(ctx *context.Context) {
		h := httpBase(ctx)
		if h != nil {
			setHeaderNoCache(ctx)
			file := ctx.Params("file")
			if file != "" {
				sendFile(ctx, h, "text/plain", "objects/info/"+file)
			} else {
				sendFile(ctx, h, "text/plain", p)
			}
		}
	}
}

// GetInfoPacks implements Git dumb HTTP
func GetInfoPacks(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		setHeaderCacheForever(ctx)
		sendFile(ctx, h, "text/plain; charset=utf-8", "objects/info/packs")
	}
}

// GetLooseObject implements Git dumb HTTP
func GetLooseObject(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		setHeaderCacheForever(ctx)
		sendFile(ctx, h, "application/x-git-loose-object", fmt.Sprintf("objects/%s/%s",
			ctx.Params("head"), ctx.Params("hash")))
	}
}

// GetPackFile implements Git dumb HTTP
func GetPackFile(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		setHeaderCacheForever(ctx)
		sendFile(ctx, h, "application/x-git-packed-objects", "objects/pack/pack-"+ctx.Params("file")+".pack")
	}
}

// GetIdxFile implements Git dumb HTTP
func GetIdxFile(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		setHeaderCacheForever(ctx)
		sendFile(ctx, h, "application/x-git-packed-objects-toc", "objects/pack/pack-"+ctx.Params("file")+".idx")
	}
}
