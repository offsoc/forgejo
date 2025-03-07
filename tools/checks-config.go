// Copyright 2025 The Forgejo Authors c/o Codeberg e.V. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
    "reflect"
	"strings"
	"sort"
	"time"
	"unicode"
	
	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/tools/go/packages"
	"go/types"
)

var ignoredKeys = []string {
	"cache.HOST",
}

var ignoredNames = map[string]([]string) {
	"actions": []string{
		"ArtifactStorage",
		"LogStorage",
	},
	"api": []string{
		"SwaggerURL",
	},
	"attachment": []string{
		"Storage",
	},
	"cache": []string{
		"Conn",
	},
	"indexer": []string{
		"IssueConnAuth",
	},
	"lfs": []string{
		"Storage",
	},
	"packages": []string{
		"Storage",
	},
	"picture": []string{
		"Storage",
	},
	"proxy": []string{
        "ProxyURLFixed",
	},
	"service": []string{
		"DefaultUserVisibilityMode",
		"DefaultOrgVisibilityMode",
		"EnableOpenIDSignIn",
		"EnableOpenIDSignUp",
		"OpenIDWhitelist",
		"OpenIDBlacklist",
	},
	"session": []string{
		"Provider",
		"OriginalProvider",
		"CookiePath",
	},
	"webhook": []string{
		"ProxyURLFixed",
	},
}

var mapNamesToTags = map[string](any) {
	"admin": map[string](string) {
		"DefaultEmailNotification": "DEFAULT_EMAIL_NOTIFICATIONS",
	},
	"database": map[string](string) {
		"Type": "DB_TYPE",
		"Timeout": "SQLITE_TIMEOUT",
		"SQLiteJournalMode": "SQLITE_JOURNAL_MODE",
		"DBConnectRetries": "DB_RETRIES",
		"DBConnectBackoff": "DB_RETRY_BACKOFF",
		"ConnMaxIdleTime": "CONN_MAX_IDLETIME",
	},
	"indexer": map[string](string) {
		"IssueType": "ISSUE_INDEXER_TYPE",
		"IssuePath": "ISSUE_INDEXER_PATH",
		"IssueConnStr": "ISSUE_INDEXER_CONN_STR",
		"RepoType": "REPO_INDEXER_TYPE",
		"RepoPath": "REPO_INDEXER_PATH",
		"RepoConnStr": "REPO_INDEXER_CONN_STR",
		"IncludePatterns": "REPO_INDEXER_INCLUDE",
		"ExcludePatterns": "REPO_INDEXER_EXCLUDE",
		"ExcludeVendored": "REPO_INDEXER_EXCLUDE_VENDORED",
		"MaxIndexerFileSize": "MAX_FILE_SIZE",
	},
	"lfs": map[string](string) {
        "Type": "STORAGE_TYPE",
    },
	"log": map[string](string) {
		"StacktraceLogLevel": "STACKTRACE_LEVEL",
	},
	"migrations": map[string](string) {
		"AllowLocalNetworks": "ALLOW_LOCALNETWORKS",
	},
	"picture": map[string](string) {
		"MaxWidth": "AVATAR_MAX_WIDTH",
		"MaxHeight": "AVATAR_MAX_HEIGHT",
		"MaxFileSize": "AVATAR_MAX_FILE_SIZE",
		"MaxOriginSize": "AVATAR_MAX_ORIGIN_SIZE",
		"RenderedSizeFactor": "AVATAR_RENDERED_SIZE_FACTOR",
	},
	"proxy": map[string](string) {
		"Enabled": "PROXY_ENABLED",
	},
	"repo-archive": map[string](string) {
        "Type": "STORAGE_TYPE",
    },
	"repository": map[string](string) {
		"UseCompatSSHURI": "USE_COMPAT_SSH_URI",
	},
	"service": map[string](string) {
		"ActiveCodeLives": "ACTIVE_CODE_LIVE_MINUTES",
		"ResetPwdCodeLives": "RESET_PASSWD_CODE_LIVE_MINUTES",
		"EmailDomainAllowList": "EMAIL_DOMAIN_ALLOWLIST",
		"EmailDomainBlockList": "EMAIL_DOMAIN_BLOCKLIST",
		"EnableInternalSignIn": "ENABLE_INTERNAL_SIGNIN",
		"RequireSignInView": "REQUIRE_SIGNIN_VIEW",
		"EnableBasicAuth": "ENABLE_BASIC_AUTHENTICATION",
		"EnableReverseProxyAuth": "ENABLE_REVERSE_PROXY_AUTHENTICATION",
		"EnableReverseProxyAuthAPI": "ENABLE_REVERSE_PROXY_AUTHENTICATION_API",
		"EnableReverseProxyAutoRegister": "ENABLE_REVERSE_PROXY_AUTO_REGISTRATION",
	},
	"session": map[string](string) {
		"Secure": "COOKIE_SECURE",
		"Gclifetime": "GC_INTERVAL_TIME",
		"Maxlifetime": "SESSION_LIFE_TIME",
	},
	"storage": map[string](string) {
		"Type": "STORAGE_TYPE",
	},
	"storage.actions_artifacts": map[string](string) {
		"Type": "STORAGE_TYPE",
	},
	"storage.actions_log": map[string](string) {
        "Type": "STORAGE_TYPE",
    },
	"storage.packages": map[string](string) {
        "Type": "STORAGE_TYPE",
    },
	"storage.repo-archive": map[string](string) {
        "Type": "STORAGE_TYPE",
    },
}

func GetReplaceValues() map[string]string {
	return map[string]string{
		"%(RUN_USER)s": setting.RunUser,
		"%(APP_DATA_PATH)s": setting.AppDataPath,
		"%(AppDataPath)s": setting.AppDataPath,
		"%(WORK_PATH)s": setting.AppWorkPath,
		"%(AppWorkPath)s": setting.AppWorkPath,
		"%(DOMAIN)s": setting.Domain,
		"%(server.DOMAIN)s": setting.Domain,
		"%(USE_PROXY_PROTOCOL)s": fmt.Sprintf("%v", setting.UseProxyProtocol),
		"%(server.USE_PROXY_PROTOCOL)s": fmt.Sprintf("%v", setting.UseProxyProtocol),
	}
}

var replaceValues map[string]string

func printSection(section setting.ConfigSection) {
	fmt.Printf("\n[%s]\n", section.Name())
	for _, key := range section.Keys() {
		fmt.Printf("%s = %s\n", key.Name(), key.Value())
	}
	for _, child := range section.ChildSections() {
		printSection(child)
	}
	//if reflect.TypeOf(cfg) == reflect.TypeFor[setting.ConfigProvider] {
	//	fmt.Println("provider")
	//}
}

type Setting struct {
    Tags        []string
    SettingObj  any
    Name string
}

func main() {
	setting.InitCfgProvider("")
	setting.LoadCommonSettings()
	setting.LoadSettings()
	cfg := setting.CfgProvider
	section_mailer, _ := cfg.NewSection("mailer")
	section_mailer.NewKey("ENABLED", "true")
	setting.LoadSettingsForInstall()
	queueSettings, _ := setting.GetQueueSettings(cfg, "")
	replaceValues = GetReplaceValues()
	/*//fmt.Println(reflect.TypeOf(cfg))
	//fmt.Println(cfg.Sections())
	//print()
	for _, section := range cfg.Sections() {
		printSection(section)
	}
	fmt.Println()
	//section, _ := cfg.GetSection("federation")
	//fmt.Println(section.HasKey("Enabled"))
	//fmt.Println(setting.CfgProvider.file)
	//t := reflect.TypeOf(setting.Federation)*/

	data, err := os.ReadFile("options/setting/config.toml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	var tomlConfig map[string]any
	err = toml.Unmarshal(data, &tomlConfig)
	if err != nil {
		fmt.Println("Error parsing TOML:", err)
		return
	}
	//tomlConfig, _ := toml.LoadFile("options/setting/config.toml")
	//fmt.Println(tomlConfig.Get("ui.notification"))

	/*pkg, _ := importer.Default().Import("code.gitea.io/gitea/modules/setting")
	for _, declName := range pkg.Scope().Names() {
        fmt.Println(declName)
    }*/

	/*fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, "modules/setting/setting.go", nil, parser.AllErrors)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Package:", node.Name.Name)
	fmt.Println("Declarations:")
	for _, decl := range node.Decls {
		fmt.Printf("%#v\n", decl)
	}*/

	cfgP := &packages.Config{Mode: packages.LoadTypes}
	pkgs, _ := packages.Load(cfgP, "code.gitea.io/gitea/modules/setting")

	setting_names := []string{}
	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if _, ok := obj.(*types.Var); ok && obj.Exported() {
				setting_names = append(setting_names, name)
			}
		}
	}
	settings := []Setting{
		{[]string{"APP_NAME"}, setting.AppName, "AppName"},
		{[]string{"APP_SLOGAN"}, setting.AppSlogan, "AppSlogan"},
		{[]string{"APP_DISPLAY_NAME_FORMAT"}, setting.AppDisplayNameFormat, "AppDisplayNameFormat"},
		{[]string{"RUN_USER"}, setting.RunUser, "RunUser"},
		{[]string{"RUN_MODE"}, setting.RunMode, "RunMode"},
		{[]string{"WORK_PATH"}, setting.AppWorkPath, "AppWorkPath"},
		{[]string{"F3"}, setting.F3, "F3"},
		{[]string{"actions"}, setting.Actions, "Actions"},
		{[]string{"admin"}, setting.Admin, "Admin"},
		{[]string{"api"}, setting.API, "API"},
		{[]string{"attachment"}, setting.Attachment, "Attachment"},
		{[]string{"avatars"}, *setting.Avatar.Storage, ""},
		{[]string{"badges"}, setting.Badges, "Badges"},
		{[]string{"cache"}, setting.CacheService.Cache, "CacheService"},
		{[]string{"cache", "last_commit"}, setting.CacheService.LastCommit, ""},
		{[]string{"camo"}, setting.Camo, "Camo"},
		{[]string{"cors"}, setting.CORSConfig, "CORSConfig"},
		//"cron" TODO
		{[]string{"database"}, setting.Database, "Database"},
		{[]string{"email", "incoming"}, setting.IncomingEmail, "IncomingEmail"},
		{[]string{"federation"}, setting.Federation, "Federation"},
		{[]string{"git"}, setting.Git, "Git"},
		{[]string{"git", "config", "core.logAllRefUpdates"}, setting.GitConfig.GetOption("core.logAllRefUpdates"), "GitConfig"},
		{[]string{"git", "config", "diff.algorithm"}, setting.GitConfig.GetOption("diff.algorithm"), ""},
		{[]string{"git", "config", "gc.reflogExpire"}, setting.GitConfig.GetOption("gc.reflogExpire"), ""},
		{[]string{"git", "timeout"}, setting.Git.Timeout, ""},
		//"highlight" TODO
		{[]string{"i18n", "LANGS"}, setting.Langs, "Langs"},
		{[]string{"i18n", "NAMES"}, setting.Names, "Names"},
		{[]string{"indexer"}, setting.Indexer, "Indexer"},
		{[]string{"lfs"}, *setting.LFS.Storage, "LFS"},
		{[]string{"lfs_client"}, setting.LFSClient, "LFSClient"},
		{[]string{"log"}, setting.Log, "Log"},
		{[]string{"mailer"}, *setting.MailService, "MailService"},
		{[]string{"markdown"}, setting.Markdown, "Markdown"},
		{[]string{"markup", "MERMAID_MAX_SOURCE_CHARACTERS"}, setting.MermaidMaxSourceCharacters, "MermaidMaxSourceCharacters"},
		{[]string{"markup", "FILEPREVIEW_MAX_LINES"}, setting.FilePreviewMaxLines, "FilePreviewMaxLines"},
		{[]string{"metrics"}, setting.Metrics, "Metrics"},
		{[]string{"migrations"}, setting.Migrations, "Migrations"},
		{[]string{"mirror"}, setting.Mirror, "Mirror"},
		{[]string{"oauth2_client"}, setting.OAuth2Client, "OAuth2Client"},
		{[]string{"openid", "ENABLE_OPENID_SIGNIN"}, setting.Service.EnableOpenIDSignIn, ""},
		{[]string{"openid", "ENABLE_OPENID_SIGNUP"}, setting.Service.EnableOpenIDSignUp, ""},
		{[]string{"openid", "WHITELISTED_URIS"}, setting.Service.OpenIDWhitelist, ""},
		{[]string{"openid", "BLACKLISTED_URIS"}, setting.Service.OpenIDBlacklist, ""},
		{[]string{"other"}, setting.Other, "Other"},
		{[]string{"packages"}, setting.Packages, "Packages"},
		{[]string{"picture"}, setting.Avatar, "Avatar"},
		{[]string{"picture", "DISABLE_GRAVATAR"}, setting.DisableGravatar, "DisableGravatar"},
		{[]string{"picture", "ENABLE_FEDERATED_AVATAR"}, setting.EnableFederatedAvatar, "EnableFederatedAvatar"},
		{[]string{"picture", "GRAVATAR_SOURCE"}, setting.GravatarSource, "GravatarSource"},
		{[]string{"picture", "REPOSITORY_AVATAR_FALLBACK"}, setting.RepoAvatar.Fallback, "Fallback"},
		{[]string{"picture", "REPOSITORY_AVATAR_FALLBACK_IMAGE"}, setting.RepoAvatar.FallbackImage, "FallbackImage"},
		//TODO: *setting.RepoAvatar.Storage,
		{[]string{"project"}, setting.Project, "Project"},
		{[]string{"proxy"}, setting.Proxy, "Proxy"},
		{[]string{"queue"}, queueSettings, ""},
		{[]string{"quota"}, setting.Quota, "Quota"},
		{[]string{"repo-archive"}, *setting.RepoArchive.Storage, "RepoArchive"},
		{[]string{"repository"}, setting.Repository, "Repository"},
		{[]string{"repository", "pull-request"}, setting.Repository.PullRequest, ""},
		{[]string{"repository", "issue"}, setting.Repository.Issue, ""},
		{[]string{"repository", "release"}, setting.Repository.Release, ""},
		{[]string{"repository", "signing"}, setting.Repository.Signing, ""},
		{[]string{"security", "COOKIE_REMEMBER_NAME"}, setting.CookieRememberName, "CookieRememberName"},
		{[]string{"security", "REVERSE_PROXY_AUTHENTICATION_USER"}, setting.ReverseProxyAuthUser, "ReverseProxyAuthUser"},
		{[]string{"security", "REVERSE_PROXY_AUTHENTICATION_EMAIL"}, setting.ReverseProxyAuthEmail, "ReverseProxyAuthEmail"},
		{[]string{"security", "REVERSE_PROXY_AUTHENTICATION_FULL_NAME"}, setting.ReverseProxyAuthFullName, "ReverseProxyAuthFullName"},
		{[]string{"security", "REVERSE_PROXY_LIMIT"}, setting.ReverseProxyLimit, "ReverseProxyLimit"},
		{[]string{"security", "REVERSE_PROXY_TRUSTED_PROXIES"}, setting.ReverseProxyTrustedProxies, "ReverseProxyTrustedProxies"},
		{[]string{"security", "MIN_PASSWORD_LENGTH"}, setting.MinPasswordLength, "MinPasswordLength"},
		{[]string{"security", "IMPORT_LOCAL_PATHS"}, setting.ImportLocalPaths, "ImportLocalPaths"},
		{[]string{"security", "DISABLE_GIT_HOOKS"}, setting.DisableGitHooks, "DisableGitHooks"},
		{[]string{"security", "DISABLE_WEBHOOKS"}, setting.DisableWebhooks, "DisableWebhooks"},
		{[]string{"security", "ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET"}, setting.OnlyAllowPushIfGiteaEnvironmentSet, "OnlyAllowPushIfGiteaEnvironmentSet"},
		{[]string{"security", "PASSWORD_HASH_ALGO"}, setting.PasswordHashAlgo, "PasswordHashAlgo"},
		{[]string{"security", "CSRF_COOKIE_HTTP_ONLY"}, setting.CSRFCookieHTTPOnly, "CSRFCookieHTTPOnly"},
		{[]string{"security", "PASSWORD_CHECK_PWN"}, setting.PasswordCheckPwn, "PasswordCheckPwn"},
		{[]string{"security", "SUCCESSFUL_TOKENS_CACHE_SIZE"}, setting.SuccessfulTokensCacheSize, "SuccessfulTokensCacheSize"},
		//TODO security.INTERNAL_TOKEN, INTERNAL_TOKEN_URI
		{[]string{"security", "PASSWORD_COMPLEXITY"}, setting.PasswordComplexity, "PasswordComplexity"},
		{[]string{"security", "DISABLE_QUERY_AUTH_TOKEN"}, setting.DisableQueryAuthToken, "DisableQueryAuthToken"},
		//"repository.mimetype_mapping", setting.MimeTypeMap
		{[]string{"server"}, setting.SSH, "SSH"},
		{[]string{"server", "ROOT_URL"}, setting.AppURL, "AppURL"},
		{[]string{"server", "APP_DATA_PATH"}, setting.AppDataPath, "AppDataPath"},
		{[]string{"server", "LOCAL_ROOT_URL"}, setting.LocalURL, "LocalURL"},
		{[]string{"server", "PROTOCOL"}, setting.Protocol, "Protocol"},
		{[]string{"server", "USE_PROXY_PROTOCOL"}, setting.UseProxyProtocol, "UseProxyProtocol"},
		{[]string{"server", "PROXY_PROTOCOL_TLS_BRIDGING"}, setting.ProxyProtocolTLSBridging, "ProxyProtocolTLSBridging"},
		{[]string{"server", "PROXY_PROTOCOL_HEADER_TIMEOUT"}, setting.ProxyProtocolHeaderTimeout, "ProxyProtocolHeaderTimeout"},
		{[]string{"server", "PROXY_PROTOCOL_ACCEPT_UNKNOWN"}, setting.ProxyProtocolAcceptUnknown, "ProxyProtocolAcceptUnknown"},
		{[]string{"server", "DOMAIN"}, setting.Domain, "Domain"},
		{[]string{"server", "HTTP_ADDR"}, setting.HTTPAddr, "HTTPAddr"},
		{[]string{"server", "HTTP_PORT"}, setting.HTTPPort, "HTTPPort"},
		{[]string{"server", "LOCAL_USE_PROXY_PROTOCOL"}, setting.LocalUseProxyProtocol, "LocalUseProxyProtocol"},
		{[]string{"server", "REDIRECT_OTHER_PORT"}, setting.RedirectOtherPort, "RedirectOtherPort"},
		{[]string{"server", "REDIRECTOR_USE_PROXY_PROTOCOL"}, setting.RedirectorUseProxyProtocol, "RedirectorUseProxyProtocol"},
		{[]string{"server", "PORT_TO_REDIRECT"}, setting.PortToRedirect, "PortToRedirect"},
		{[]string{"server", "OFFLINE_MODE"}, setting.OfflineMode, "OfflineMode"},
		{[]string{"server", "CERT_FILE"}, setting.CertFile, "CertFile"},
		{[]string{"server", "KEY_FILE"}, setting.KeyFile, "KeyFile"},
		{[]string{"server", "STATIC_ROOT_PATH"}, setting.StaticRootPath, "StaticRootPath"},
		{[]string{"server", "STATIC_CACHE_TIME"}, setting.StaticCacheTime, "StaticCacheTime"},
		{[]string{"server", "ENABLE_GZIP"}, setting.EnableGzip, "EnableGzip"},
		//setting.LandingPageURL
		{[]string{"server", "UNIX_SOCKET_PERMISSION"}, setting.UnixSocketPermission, "UnixSocketPermission"},
		{[]string{"server", "ENABLE_PPROF"}, setting.EnablePprof, "EnablePprof"},
		{[]string{"server", "PPROF_DATA_PATH"}, setting.PprofDataPath, "PprofDataPath"},
		{[]string{"server", "ENABLE_ACME"}, setting.EnableAcme, "EnableAcme"},
		{[]string{"server", "ACME_ACCEPTTOS"}, setting.AcmeTOS, "AcmeTOS"},
		{[]string{"server", "ACME_DIRECTORY"}, setting.AcmeLiveDirectory, "AcmeLiveDirectory"},
		{[]string{"server", "ACME_EMAIL"}, setting.AcmeEmail, "AcmeEmail"},
		{[]string{"server", "ACME_URL"}, setting.AcmeURL, "AcmeURL"},
		{[]string{"server", "ACME_CA_ROOT"}, setting.AcmeCARoot, "AcmeCARoot"},
		{[]string{"server", "SSL_MIN_VERSION"}, setting.SSLMinimumVersion, "SSLMinimumVersion"},
		{[]string{"server", "SSL_MAX_VERSION"}, setting.SSLMaximumVersion, "SSLMaximumVersion"},
		{[]string{"server", "SSL_CURVE_PREFERENCES"}, setting.SSLCurvePreferences, "SSLCurvePreferences"},
		{[]string{"server", "SSL_CIPHER_SUITES"}, setting.SSLCipherSuites, "SSLCipherSuites"},
		{[]string{"server", "ALLOW_GRACEFUL_RESTARTS"}, setting.GracefulRestartable, "GracefulRestartable"},
		{[]string{"server", "GRACEFUL_HAMMER_TIME"}, setting.GracefulHammerTime, "GracefulHammerTime"},
		{[]string{"server", "STARTUP_TIMEOUT"}, setting.StartupTimeout, "StartupTimeout"},
		{[]string{"server", "PER_WRITE_TIMEOUT"}, setting.PerWriteTimeout, "PerWriteTimeout"},
		{[]string{"server", "PER_WRITE_PER_KB_TIMEOUT"}, setting.PerWritePerKbTimeout, "PerWritePerKbTimeout"},
		{[]string{"server", "STATIC_URL_PREFIX"}, setting.StaticURLPrefix, "StaticURLPrefix"},
		//setting.AbsoluteAssetURL
		//setting.ManifestData
		{[]string{"server", "LFS_START_SERVER"}, setting.LFS.StartServer, ""},
		{[]string{"server", "LFS_HTTP_AUTH_EXPIRY"}, setting.LFS.HTTPAuthExpiry, ""},
		{[]string{"server", "LFS_MAX_FILE_SIZE"}, setting.LFS.MaxFileSize, ""},
		{[]string{"server", "LFS_LOCKS_PAGING_NUM"}, setting.LFS.LocksPagingNum, ""},
		{[]string{"server", "LFS_MAX_BATCH_SIZE"}, setting.LFS.MaxBatchSize, ""},
		{[]string{"service"}, setting.Service, "Service"},
		{[]string{"service", "explore"}, setting.Service.Explore, ""},
		{[]string{"session"}, setting.SessionConfig, "SessionConfig"},
		{[]string{"session", "PROVIDER"}, setting.SessionConfig.OriginalProvider, ""},
		{[]string{"ssh", "minimum_key_sizes", "DSA"}, setting.SSH.MinimumKeySizes["dsa"], ""},
		{[]string{"ssh", "minimum_key_sizes", "ECDSA"}, setting.SSH.MinimumKeySizes["ecdsa"], ""},
		{[]string{"ssh", "minimum_key_sizes", "ECDSA-SK"}, setting.SSH.MinimumKeySizes["ecdsa-sk"], ""},
		{[]string{"ssh", "minimum_key_sizes", "ED25519"}, setting.SSH.MinimumKeySizes["ed25519"], ""},
		{[]string{"ssh", "minimum_key_sizes", "ED25519-SK"}, setting.SSH.MinimumKeySizes["ed25519-sk"], ""},
		{[]string{"ssh", "minimum_key_sizes", "RSA"}, setting.SSH.MinimumKeySizes["rsa"], ""},
		//TODO default storage
		{[]string{"storage", "actions_artifacts"}, *setting.Actions.ArtifactStorage, ""},
		{[]string{"storage", "actions_log"}, *setting.Actions.LogStorage, ""},
		{[]string{"storage", "attachments"}, *setting.Attachment.Storage, ""},
		{[]string{"storage", "packages"}, *setting.Packages.Storage, "Storage"},
		{[]string{"time", "DEFAULT_UI_LOCATION"}, setting.DefaultUILocation, "DefaultUILocation"},
		{[]string{"ui"}, setting.UI, "UI"},
		{[]string{"ui", "notification"}, setting.UI.Notification, ""},
		{[]string{"ui", "svg"}, setting.UI.SVG, ""},
		{[]string{"ui", "csv"}, setting.UI.CSV, ""},
		{[]string{"ui", "admin"}, setting.UI.Admin, ""},
		{[]string{"ui", "user"}, setting.UI.User, ""},
		{[]string{"ui", "meta"}, setting.UI.Meta, ""},
		{[]string{"webhook"}, setting.Webhook, "Webhook"},
	}
	setting_names_processed := []string{}
	for _, setting := range settings {
		if setting.Name != "" {
			setting_names_processed = append(setting_names_processed, setting.Name)
		}
		settingObj := setting.SettingObj
		key := strings.Join(setting.Tags, ".")
		tomlConfigObj, exists := getNestedValue(tomlConfig, setting.Tags...)
		if !exists {
			fmt.Printf("WARNING: Not found %s in options/setting/config.toml\n", key)
			continue
		}
		for i, _ := range(setting.Tags) {
			if t, exists := getNestedValue(tomlConfig, setting.Tags[:i + 1]...); exists {
				t.(map[string]any)["checked"] = true
			}
		}
		ignoredNamesObj, exists := ignoredNames[key]
		if !exists {
			ignoredNamesObj = []string{}
		}
		mapNamesToTagsObj, exists := mapNamesToTags[key]
		if !exists {
			mapNamesToTagsObj = map[string]string{}
		}
		check(key, settingObj, tomlConfigObj.(map[string]any), ignoredNamesObj, mapNamesToTagsObj.(map[string]string))
	}
	keysTomlConfig := getKeys(tomlConfig)
	sort.Slice(keysTomlConfig, func(i, j int) bool {
	    return strings.Join(keysTomlConfig[i], ".") < strings.Join(keysTomlConfig[j], ".")
	})
	for _, keys := range keysTomlConfig {
		lastKey := keys[len(keys) - 1]
		if lastKey == "default" || lastKey == "description" || lastKey == "heading" || lastKey == "checked"{
			continue
		}
		key := strings.Join(keys, ".")
		expected := false
		for _, ignoredKey := range ignoredKeys {
			if ignoredKey == key {
				expected = true
				continue
			}
		}
		if expected {
			continue
		}
		if checked, exists := getNestedValue(tomlConfig, append(keys, "checked")...); !exists || !checked.(bool) {
			fmt.Printf("WARNING: Not checked %s from options/setting/config.toml\n", strings.Join(keys, "."))
		}
	}
	for _, a := range setting_names {
		processed := false
		for _, b := range setting_names_processed {
			if a == b {
				processed = true
				continue
			}
		}
		if !processed {
			fmt.Printf("WaRNING: Not processed %s from modules/setting\n", a)
		}
	}
}

func check(key string, settingObj any, tomlConfigObj map[string]any, ignoredNamesObj []string, mapNamesToTagsObj map[string]string) {
	v := reflect.ValueOf(settingObj)
	t := v.Type()
	if t.Kind() != reflect.Struct {
		tomlValue, _ := tomlConfigObj["default"]
		// Warn if values are not the same
		if !equal(settingObj, tomlValue) {
			fmt.Printf("WARNING: Inconsistent values found for\n- code: %s = %v\n- toml: %s = %v\n", key, settingObj, key, tomlValue)
		}
		return
	}
	for i, field := range reflect.VisibleFields(t) {
		// Ignore structs
		if field.Type.Kind() == reflect.Struct {
			continue
		}
		// Retrieve variable information from modules/setting
		settingCodeName := field.Name
		settingCodeValue := v.Field(i).Interface()
		/*
		settingCodeType := field.Type
		fmt.Printf("--- code: %s = %s ; %s\n", settingCodeName, settingCodeValue, settingCodeType)
		*/
		// Ignore struct fields that are ignored above
		ignored := false
		for _, ignoredName := range ignoredNamesObj {
			if ignoredName == settingCodeName {
				ignored = true
				continue
			}
		}
		if ignored {
			continue
		}
		// Get tag for variable information in options/setting/config.toml
		settingIniTag := field.Tag.Get(`ini`)
		if settingIniTag == "" {
			settingIniTagMap, exists := mapNamesToTagsObj[settingCodeName]
			if exists {
				settingIniTag = settingIniTagMap
			} else {
				settingIniTag = toUpperSnakeCase(settingCodeName)
			}
		}
		// Ignore struct fields that have tag `ini:"-"` or are ignored above
		if settingIniTag == "-" {
            continue
        }
		settingTomlKey := key + "." + settingIniTag
		// Warn if setting is not found in options/setting/config.toml
		settingToml, exists := tomlConfigObj[settingIniTag]
		if !exists {
			fmt.Printf("WARNING: Not found %s in options/setting/config.toml\n- code: %s = %v\n", settingTomlKey, settingCodeName, settingCodeValue)
			continue
		}
		// Retrieve variable information from options/setting/config.toml
		settingTomlValue, _ := settingToml.(map[string]any)["default"]
		/*
		fmt.Printf("--- toml: %s = %s\n", settingTomlKey, settingTomlValue)
		*/
		settingToml.(map[string]any)["checked"] = true
		// Warn if values are not the same
		if !equal(settingCodeValue, settingTomlValue) {
			fmt.Printf("WARNING: Inconsistent values found for\n- code: %s = %#v\n- toml: %s = %#v\n", settingCodeName, settingCodeValue, settingTomlKey, settingTomlValue)
		}
	}
}

// Convert CamelCase to UPPER_SNAKE_CASE
func toUpperSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) && (i+1 < len(s) && unicode.IsLower(rune(s[i+1])) || unicode.IsLower(rune(s[i-1]))) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToUpper(r))
	}
	return string(result)
}

func equal(a, b interface{}) bool {
	return reflect.DeepEqual(a, b) || equalNotMirrored(a, b) || equalNotMirrored(b, a)
}

func equalNotMirrored(a, b interface{}) bool {
	if a == nil {
		if bStringArray, ok := b.([]string); ok {
			return len(bStringArray) == 0
		}
		if bContainerSet, ok := b.(container.Set[string]); ok {
			return len(bContainerSet) == 0
		}
		if bGlobArray, ok := b.([]setting.Glob); ok {
			return len(bGlobArray) == 0
		}
		return b == ""
	}
	// Mismatching types of int and int64
	if aInt, ok := a.(int); ok {
		if bInt64, ok := b.(int64); ok {
			return int64(aInt) == bInt64
		}
	}
	// Size values convert to bytes instead of MiB
	if aInt64, ok := a.(int64); ok {
		if bInt64, ok := b.(int64); ok {
			return aInt64 == 1 << 20 * bInt64
		}
	}
	// Size values
	if aUInt32, ok := a.(uint32); ok {
		if bInt64, ok := b.(int64); ok {
			return int64(aUInt32) == bInt64
		}
	}
	// Time duration values
	if aTime, ok := a.(time.Duration); ok {
		if bString, ok := b.(string); ok {
			if bTime, err := time.ParseDuration(bString); err == nil {
				return aTime == bTime
			}
		}
		if bInt64, ok := b.(int64); ok {
			bTime := time.Duration(bInt64) * time.Second
			return aTime == bTime
		}
	}
    if aString, ok := a.(string); ok {
		for from, to := range replaceValues {
			aString = strings.ReplaceAll(aString, from, to)
		}
		if aString == fmt.Sprintf("%v", b) {
			return true
		}
		/*// Storage type
		if bStorageType, ok := b.(setting.StorageType); ok {
			return aString == string(bStorageType)
		}*/
		// Log level
		if bLogLevel, ok := b.(log.Level); ok {
			return log.LevelFromString(aString) == bLogLevel
		}
		// OAuth2Username
		if bUsername, ok := b.(setting.OAuth2UsernameType); ok {
			return setting.OAuth2UsernameType(aString) == bUsername
		}
		// OAuth2AccountLinking
		if bAccountLinking, ok := b.(setting.OAuth2AccountLinkingType); ok {
			return setting.OAuth2AccountLinkingType(aString) == bAccountLinking
		}
		// Server scheme
		if bServerScheme, ok := b.(setting.Scheme); ok {
			return setting.Scheme(aString) == bServerScheme
		}
		// HTTP same site
		if bSameSite, ok := b.(http.SameSite); ok {
			switch strings.ToLower(aString) {
			case "none":
				return bSameSite == http.SameSiteNoneMode
			case "strict":
				return bSameSite == http.SameSiteStrictMode
			}
			return bSameSite == http.SameSiteLaxMode
		}
		// Relative path
		if bString, ok := b.(string); ok {
			bStringAbsolute := filepath.Join(setting.AppDataPath, bString)
			return aString == bStringAbsolute
		}
		aStringArray := strings.Split(aString, ",")
		if bContainerSet, ok := b.(container.Set[string]); ok {
			aContainerSet := container.SetOf(aStringArray...)
			bStringArray := bContainerSet.Values()
			return aContainerSet.IsSubset(bStringArray) && bContainerSet.IsSubset(aStringArray)
		}
		if bStringArray, ok := b.([]string); ok {
			if len(aStringArray) != len(bStringArray) {
				return false
			}
			for i := range aStringArray {
				if strings.TrimSpace(aStringArray[i]) != strings.TrimSpace(bStringArray[i]) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func getNestedValue(config map[string]any, keys ...string) (any, bool) {
	current := any(config)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func getKeys(m map[string]any, path ...string) [][]string {
	var keys [][]string
	for key, value := range m {
		currentPath := append(path, key)
		keys = append(keys, currentPath)
		if nestedMap, ok := value.(map[string]any); ok {
			keys = append(keys, getKeys(nestedMap, currentPath...)...)
		}
	}
	return keys
}
