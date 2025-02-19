---
title: 'Configuration Cheat Sheet'
license: 'Apache-2.0'
origin_url: 'https://github.com/go-gitea/gitea/blob/e865de1e9d65dc09797d165a51c8e705d2a86030/docs/content/administration/config-cheat-sheet.en-us.md'
edit_url: 'https://codeberg.org/forgejo/forgejo/src/branch/forgejo/options/config.toml'
---

This is a cheat sheet for the Forgejo configuration file. It contains the settings
that can be configured as well as their default values.

Any changes to the Forgejo configuration file should be made in `custom/conf/app.ini`
or any corresponding location. When installing from a distribution, this will
typically be found at `/etc/forgejo/app.ini`.

In the default values below, a value in the form `$XYZ` refers to an environment variable. See [environment-to-ini](https://codeberg.org/forgejo/forgejo/src/branch/forgejo/contrib/environment-to-ini) for information on how environment variables are translated to `app.ini` variables. Values in the form _`XxYyZz`_ refer to values listed as part of the default configuration. These notation forms will not work in your own `app.ini` file and are only listed here as documentation.

Any string in the format `%(X)s` is a feature powered by [ini](https://github.com/go-ini/ini/#recursive-values), for reading values recursively.

Values containing `#` or `;` must be quoted using `` ` `` or `"""`.

**Note:** A full restart is required for Forgejo configuration changes to take effect.

## Default configuration (non-`app.ini` configuration)

These values are environment-dependent but form the basis of a lot of values. They will be reported as part of the default configuration when running `forgejo help` or on start-up. The order they are emitted there is slightly different but we will list them here in the order they are set-up.

- <a name="AppPath" href="#AppPath">_`AppPath`_</a>: This is the absolute path of the running `forgejo` binary.

- <a name="AppWorkPath" href="#AppWorkPath">_`AppWorkPath`_</a>: This refers to "working path" of the `forgejo` binary. It is determined by using the first set thing in the following hierarchy:
  1. The `WORK_PATH` option in "app.ini" file
  2. The `--work-path` flag passed to the binary
  3. The environment variable `$FORGEJO_WORK_DIR`
  4. The environment variable `$GITEA_WORK_DIR`
  5. A built-in value set at build time (see building from source)
  6. Otherwise it defaults to the directory of the _`AppPath`_

  If any of the above are relative paths then they are made absolute against the directory of the _`AppPath`_.

- <a name="CustomPath" href="#CustomPath">_`CustomPath`_</a>: This is the base directory for custom templates and other options. It is determined by using the first set thing in the following hierarchy:
  1. The `--custom-path` flag passed to the binary
  2. The environment variable `$FORGEJO_CUSTOM`
  3. The environment variable `$GITEA_CUSTOM`
  4. A built-in value set at build time (see building from source)
  5. Otherwise it defaults to _`AppWorkPath`_`/custom`

  If any of the above are relative paths then they are made absolute against the directory of the _`AppWorkPath`_.

- <a name="CustomConf" href="#CustomConf">_`CustomConf`_</a>: This is the path to the `app.ini` file.
  1. The `--config` flag passed to the binary
  2. A built-in value set at build time (see building from source)
  3. Otherwise it defaults to _`CustomPath`_`/conf/app.ini`

  If any of the above are relative paths then they are made absolute against the directory of the _`CustomPath`_.

- <a name="StaticRootPath" href="#StaticRootPath">_`StaticRootPath`_</a>: This can be set as a built-in at build time, but will otherwise default to _`AppWorkPath`_.

## General settings

- <a name="APP_NAME" href="#APP_NAME">`APP_NAME`</a>:
  Application name that is shown in every page title:
  ```ini
  APP_NAME = Forgejo
  ```

- <a name="APP_SLOGAN" href="#APP_SLOGAN">`APP_SLOGAN`</a>:
  Slogan that is shown near the <a href="#APP_NAME">`APP_NAME`</a> in every page title:
  ```ini
  APP_SLOGAN = Beyond coding. We Forge.
  ```

- <a name="APP_DISPLAY_NAME_FORMAT" href="#APP_DISPLAY_NAME_FORMAT">`APP_DISPLAY_NAME_FORMAT`</a>:
  Format how the application display name presented in every page title. It is only used if <a href="#APP_SLOGAN">`APP_SLOGAN`</a> is set.:
  ```ini
  APP_DISPLAY_NAME_FORMAT = {APP_NAME}: {APP_SLOGAN}
  ```

- <a name="RUN_USER" href="#RUN_USER">`RUN_USER`</a>:
  Operating system user name that is running Forgejo. If it is omitted it will be automatically detected as the current user.:
  ```ini
  RUN_USER =
  ```

- <a name="RUN_MODE" href="#RUN_MODE">`RUN_MODE`</a>:
  Application run mode, which affects performance and debugging: `"dev"` or `"prod"`. Mode `"dev"` makes Forgejo easier to develop and debug, values other than `"dev"` are treated as `"prod"` which is for production use.:
  ```ini
  RUN_MODE = prod
  ```

- <a name="WORK_PATH" href="#WORK_PATH">`WORK_PATH`</a>:
  Path of the working directory. It sets the _<a href="#AppWorkPath">`AppWorkPath`</a>_ variable and may also be specified by the command line argument `--work-path` or an environment variable, `GITEA_WORK_DIR`, and defaults to the path of the Forgejo binary.:
  ```ini
  WORK_PATH =
  ```

## <a name="server" href="#server">Server</a>

```ini
[server]
```

- <a name="server.PROTOCOL" href="#server.PROTOCOL">`server.PROTOCOL`</a>:
  Protocol the server listens on. Must be one of `"http"`, `"https"`, `"http+unix"`, `"fcgi"` or `"fcgi+unix"`. Note: the value must be lowercase.:
  ```ini
  PROTOCOL = http
  ```

- <a name="server.USE_PROXY_PROTOCOL" href="#server.USE_PROXY_PROTOCOL">`server.USE_PROXY_PROTOCOL`</a>:
  Expect PROXY protocol headers on connections:
  ```ini
  USE_PROXY_PROTOCOL = false
  ```

- <a name="server.PROXY_PROTOCOL_TLS_BRIDGING" href="#server.PROXY_PROTOCOL_TLS_BRIDGING">`server.PROXY_PROTOCOL_TLS_BRIDGING`</a>:
  Use PROXY protocol in TLS Bridging mode:
  ```ini
  PROXY_PROTOCOL_TLS_BRIDGING = false
  ```

- <a name="server.PROXY_PROTOCOL_HEADER_TIMEOUT" href="#server.PROXY_PROTOCOL_HEADER_TIMEOUT">`server.PROXY_PROTOCOL_HEADER_TIMEOUT`</a>:
  Timeout to wait for PROXY protocol header (set to `0` to have no timeout):
  ```ini
  PROXY_PROTOCOL_HEADER_TIMEOUT = 5s
  ```

- <a name="server.PROXY_PROTOCOL_ACCEPT_UNKNOWN" href="#server.PROXY_PROTOCOL_ACCEPT_UNKNOWN">`server.PROXY_PROTOCOL_ACCEPT_UNKNOWN`</a>:
  Accept PROXY protocol headers with UNKNOWN type:
  ```ini
  PROXY_PROTOCOL_ACCEPT_UNKNOWN = false
  ```

- <a name="server.DOMAIN" href="#server.DOMAIN">`server.DOMAIN`</a>:
  Domain for the server:
  ```ini
  DOMAIN = localhost
  ```

- <a name="server.ROOT_URL" href="#server.ROOT_URL">`server.ROOT_URL`</a>:
  Root URL of the public URL. If specified it overwrites the automatically generated public URL, which is necessary for proxies and inside containers.:
  ```ini
  ROOT_URL = %(PROTOCOL)s://%(DOMAIN)s:%(HTTP_PORT)s/
  ```

- <a name="server.STATIC_URL_PREFIX" href="#server.STATIC_URL_PREFIX">`server.STATIC_URL_PREFIX`</a>:
  Prefix of the static URL. If omitted it will follow the prefix of <a href="#server.ROOT_URL">`ROOT_URL`</a>:
  ```ini
  STATIC_URL_PREFIX =
  ```

- <a name="server.HTTP_ADDR" href="#server.HTTP_ADDR">`server.HTTP_ADDR`</a>:
  Address to listen on. Either a IPv4/IPv6 address or the path to a unix socket.
  If <a href="#server.PROTOCOL">`PROTOCOL`</a> is set to `"http+unix"` or `"fcgi+unix"`, this should be the name of the Unix socket file to use.
  Relative paths will be made absolute against the _<a href="#AppWorkPath">`AppWorkPath`</a>_.:
  ```ini
  HTTP_ADDR = 0.0.0.0
  ```

- <a name="server.HTTP_PORT" href="#server.HTTP_PORT">`server.HTTP_PORT`</a>:
  HTTP port to listen on. It should be left empty when using a unix socket.:
  ```ini
  HTTP_PORT = 3000
  ```

- <a name="server.REDIRECT_OTHER_PORT" href="#server.REDIRECT_OTHER_PORT">`server.REDIRECT_OTHER_PORT`</a>:
  Redirect to other port. If `true` and <a href="#server.PROTOCOL">`PROTOCOL`</a> is `"https"` an HTTP server will be started on <a href="#server.PORT_TO_REDIRECT">`PORT_TO_REDIRECT`</a> and it will redirect plain, non-secure HTTP requests to the main <a href="#server.ROOT_URL">`ROOT_URL`</a>.:
  ```ini
  REDIRECT_OTHER_PORT = false
  ```

- <a name="server.PORT_TO_REDIRECT" href="#server.PORT_TO_REDIRECT">`server.PORT_TO_REDIRECT`</a>:
  Port to redirect if <a href="#server.REDIRECT_OTHER_PORT">`REDIRECT_OTHER_PORT`</a> is `true`:
  ```ini
  PORT_TO_REDIRECT = 80
  ```

- <a name="server.REDIRECTOR_USE_PROXY_PROTOCOL" href="#server.REDIRECTOR_USE_PROXY_PROTOCOL">`server.REDIRECTOR_USE_PROXY_PROTOCOL`</a>:
  Expect PROXY protocol header on connections to HTTPS redirector:
  ```ini
  REDIRECTOR_USE_PROXY_PROTOCOL = %(USE_PROXY_PROTOCOL)s
  ```

- <a name="server.SSL_MIN_VERSION" href="#server.SSL_MIN_VERSION">`server.SSL_MIN_VERSION`</a>:
  Minimum supported TLS version:
  ```ini
  SSL_MIN_VERSION = TLSv1.2
  ```

- <a name="server.SSL_MAX_VERSION" href="#server.SSL_MAX_VERSION">`server.SSL_MAX_VERSION`</a>:
  Maximum supported TLS version:
  ```ini
  SSL_MAX_VERSION =
  ```

- <a name="server.SSL_CURVE_PREFERENCES" href="#server.SSL_CURVE_PREFERENCES">`server.SSL_CURVE_PREFERENCES`</a>:
  Preferences for the SSL curve algorithms:
  ```ini
  SSL_CURVE_PREFERENCES = X25519,P256
  ```

- <a name="server.SSL_CIPHER_SUITES" href="#server.SSL_CIPHER_SUITES">`server.SSL_CIPHER_SUITES`</a>:
  SSL Cipher Suites. It defaults to `"ecdhe_ecdsa_with_aes_256_gcm_sha384,ecdhe_rsa_with_aes_256_gcm_sha384,ecdhe_ecdsa_with_aes_128_gcm_sha256,ecdhe_rsa_with_aes_128_gcm_sha256,ecdhe_ecdsa_with_chacha20_poly1305,ecdhe_rsa_with_chacha20_poly1305"` if AES is supported by hardware, otherwise `"chacha"` will be first.:
  ```ini
  SSL_CIPHER_SUITES =
  ```

- <a name="server.PER_WRITE_TIMEOUT" href="#server.PER_WRITE_TIMEOUT">`server.PER_WRITE_TIMEOUT`</a>:
  Timeout for any write connection. If `-1`, all timeouts are disabled. Formatted as strings such as `"30s"`.:
  ```ini
  PER_WRITE_TIMEOUT = 30s
  ```

- <a name="server.PER_WRITE_PER_KB_TIMEOUT" href="#server.PER_WRITE_PER_KB_TIMEOUT">`server.PER_WRITE_PER_KB_TIMEOUT`</a>:
  Timeout per kB written to any write connection. Formatted as strings such as `"30s"`.:
  ```ini
  PER_WRITE_PER_KB_TIMEOUT = 30s
  ```

- <a name="server.UNIX_SOCKET_PERMISSION" href="#server.UNIX_SOCKET_PERMISSION">`server.UNIX_SOCKET_PERMISSION`</a>:
  Permission for unix socket as octal mode (see `chmod --help`):
  ```ini
  UNIX_SOCKET_PERMISSION = 666
  ```

- <a name="server.LOCAL_ROOT_URL" href="#server.LOCAL_ROOT_URL">`server.LOCAL_ROOT_URL`</a>:
  Local (DMZ) URL for Forgejo workers (such as SSH update) accessing web service.
  In most cases you do not need to change the default value.
  Alter it only if your SSH server node is not the same as HTTP node.
  For different protocols, the default values are different.
  - If <a href="#server.PROTOCOL">`PROTOCOL`</a> is `"http+unix"`, the default value is `"http://unix/"`.
  - If <a href="#server.PROTOCOL">`PROTOCOL`</a> is `"fcgi"` or `"fcgi+unix"`, the default value is `"%(PROTOCOL)s://%(HTTP_ADDR)s:%(HTTP_PORT)s/"`.
  - If listen on `"0.0.0.0"`, the default value is `"%(PROTOCOL)s://localhost:%(HTTP_PORT)s/"`.
  - Otherwise the default value is `"%(PROTOCOL)s://%(HTTP_ADDR)s:%(HTTP_PORT)s/"`.:
  ```ini
  LOCAL_ROOT_URL =
  ```

- <a name="server.LOCAL_USE_PROXY_PROTOCOL" href="#server.LOCAL_USE_PROXY_PROTOCOL">`server.LOCAL_USE_PROXY_PROTOCOL`</a>:
  xpect PROXY protocol header when making local connections. If omitted, then it is the value of <a href="#server.USE_PROXY_PROTOCOL">`USE_PROXY_PROTOCOL`</a>.:
  ```ini
  LOCAL_USE_PROXY_PROTOCOL = %(USE_PROXY_PROTOCOL)s
  ```

- <a name="server.DISABLE_SSH" href="#server.DISABLE_SSH">`server.DISABLE_SSH`</a>:
  Disable SSH:
  ```ini
  DISABLE_SSH = false
  ```

- <a name="server.START_SSH_SERVER" href="#server.START_SSH_SERVER">`server.START_SSH_SERVER`</a>:
  Whether to use the builtin SSH server:
  ```ini
  START_SSH_SERVER = false
  ```

- <a name="server.SSH_SERVER_USE_PROXY_PROTOCOL" href="#server.SSH_SERVER_USE_PROXY_PROTOCOL">`server.SSH_SERVER_USE_PROXY_PROTOCOL`</a>:
  Expect PROXY protocol header on connections to the built-in SSH server:
  ```ini
  SSH_SERVER_USE_PROXY_PROTOCOL = false
  ```

- <a name="server.BUILTIN_SSH_SERVER_USER" href="#server.BUILTIN_SSH_SERVER_USER">`server.BUILTIN_SSH_SERVER_USER`</a>:
  Username to use for the builtin SSH server. If omitted, then it is the value of <a href="#RUN_USER">`RUN_USER`</a>.:
  ```ini
  BUILTIN_SSH_SERVER_USER = %(RUN_USER)s
  ```

- <a name="server.SSH_DOMAIN" href="#server.SSH_DOMAIN">`server.SSH_DOMAIN`</a>:
  Domain name to be exposed in clone URL:
  ```ini
  SSH_DOMAIN = %(DOMAIN)s
  ```

- <a name="server.SSH_USER" href="#server.SSH_USER">`server.SSH_USER`</a>:
  SSH username displayed in clone URLs:
  ```ini
  SSH_USER = %(BUILTIN_SSH_SERVER_USER)s
  ```

- <a name="server.SSH_LISTEN_HOST" href="#server.SSH_LISTEN_HOST">`server.SSH_LISTEN_HOST`</a>:
  Network interface the builtin SSH server should listen on:
  ```ini
  SSH_LISTEN_HOST =
  ```

- <a name="server.SSH_PORT" href="#server.SSH_PORT">`server.SSH_PORT`</a>:
  Port number to be exposed in clone URL:
  ```ini
  SSH_PORT = 22
  ```

- <a name="server.SSH_LISTEN_PORT" href="#server.SSH_LISTEN_PORT">`server.SSH_LISTEN_PORT`</a>:
  Port number the builtin SSH server should listen on:
  ```ini
  SSH_LISTEN_PORT = %(SSH_PORT)s
  ```

- <a name="server.SSH_ROOT_PATH" href="#server.SSH_ROOT_PATH">`server.SSH_ROOT_PATH`</a>:
  Root path of SSH directory, default is `"~/.ssh"`, but you have to use `"/home/git/.ssh"`.:
  ```ini
  SSH_ROOT_PATH =
  ```

- <a name="server.SSH_CREATE_AUTHORIZED_KEYS_FILE" href="#server.SSH_CREATE_AUTHORIZED_KEYS_FILE">`server.SSH_CREATE_AUTHORIZED_KEYS_FILE`</a>:
  Forgejo will create an `authorized_keys` file by default when it is not using the internal SSH server.
  If you intend to use the `AuthorizedKeysCommand` functionality then you should turn this off.:
  ```ini
  SSH_CREATE_AUTHORIZED_KEYS_FILE = true
  ```

- <a name="server.SSH_CREATE_AUTHORIZED_PRINCIPALS_FILE" href="#server.SSH_CREATE_AUTHORIZED_PRINCIPALS_FILE">`server.SSH_CREATE_AUTHORIZED_PRINCIPALS_FILE`</a>:
  Forgejo will create an `authorized_principals` file by default when it is not using the internal SSH server.
  If you intend to use the `AuthorizedPrincipalsCommand` functionality then you should turn this off.:
  ```ini
  SSH_CREATE_AUTHORIZED_PRINCIPALS_FILE = true
  ```

- <a name="server.SSH_SERVER_CIPHERS" href="#server.SSH_SERVER_CIPHERS">`server.SSH_SERVER_CIPHERS`</a>:
  For the built-in SSH server, choose the ciphers to support for SSH connections.
  For system SSH this setting has no effect.:
  ```ini
  SSH_SERVER_CIPHERS = chacha20-poly1305@openssh.com, aes128-ctr, aes192-ctr, aes256-ctr, aes128-gcm@openssh.com, aes256-gcm@openssh.com
  ```

- <a name="server.SSH_SERVER_KEY_EXCHANGES" href="#server.SSH_SERVER_KEY_EXCHANGES">`server.SSH_SERVER_KEY_EXCHANGES`</a>:
  For the built-in SSH server, choose the key exchange algorithms to support for SSH connections.
  For system SSH this setting has no effect.:
  ```ini
  SSH_SERVER_KEY_EXCHANGES = curve25519-sha256, ecdh-sha2-nistp256, ecdh-sha2-nistp384, ecdh-sha2-nistp521, diffie-hellman-group14-sha256, diffie-hellman-group14-sha1
  ```

- <a name="server.SSH_SERVER_MACS" href="#server.SSH_SERVER_MACS">`server.SSH_SERVER_MACS`</a>:
  For the built-in SSH server, choose the MACs to support for SSH connections.
  For system SSH this setting has no effect.:
  ```ini
  SSH_SERVER_MACS = hmac-sha2-256-etm@openssh.com, hmac-sha2-256, hmac-sha1
  ```

- <a name="server.SSH_SERVER_HOST_KEYS" href="#server.SSH_SERVER_HOST_KEYS">`server.SSH_SERVER_HOST_KEYS`</a>:
  For the built-in SSH server, choose the keypair to offer as the host key.
  The private key should be at `SSH_SERVER_HOST_KEY` and the public key at `SSH_SERVER_HOST_KEY`.pub.
  Relative paths are made absolute relative to the <a href="#server.APP_DATA_PATH">`APP_DATA_PATH`</a>.:
  ```ini
  SSH_SERVER_HOST_KEYS = ssh/gitea.rsa, ssh/gogs.rsa
  ```

- <a name="server.SSH_KEY_TEST_PATH" href="#server.SSH_KEY_TEST_PATH">`server.SSH_KEY_TEST_PATH`</a>:
  Path to the directory to create temporary files in when testing public keys using ssh-keygen, default is the system temporary directory.:
  ```ini
  SSH_KEY_TEST_PATH =
  ```

- <a name="server.SSH_KEYGEN_PATH" href="#server.SSH_KEYGEN_PATH">`server.SSH_KEYGEN_PATH`</a>:
  Path to the `ssh-keygen` binary to parse public SSH keys. The value is passed to the shell. By default, Forgejo does the parsing itself.:
  ```ini
  SSH_KEYGEN_PATH =
  ```

- <a name="server.SSH_AUTHORIZED_KEYS_BACKUP" href="#server.SSH_AUTHORIZED_KEYS_BACKUP">`server.SSH_AUTHORIZED_KEYS_BACKUP`</a>:
  Enable SSH authorized key backup when rewriting all keys:
  ```ini
  SSH_AUTHORIZED_KEYS_BACKUP = false
  ```

- <a name="server.SSH_AUTHORIZED_PRINCIPALS_ALLOW" href="#server.SSH_AUTHORIZED_PRINCIPALS_ALLOW">`server.SSH_AUTHORIZED_PRINCIPALS_ALLOW`</a>:
  Allowed SSH principals. Possible values are
  - empty: if SSH_TRUSTED_USER_CA_KEYS is empty this will default to off, otherwise will default to email, username.
  - `"off"`: Do not allow authorized principals
  - `"email"`: the principal must match the user's email
  - `"username"`: the principal must match the user's username
  - `"anything"`: there will be no checking on the content of the principal:
  ```ini
  SSH_AUTHORIZED_PRINCIPALS_ALLOW = email, username
  ```

- <a name="server.SSH_AUTHORIZED_PRINCIPALS_BACKUP" href="#server.SSH_AUTHORIZED_PRINCIPALS_BACKUP">`server.SSH_AUTHORIZED_PRINCIPALS_BACKUP`</a>:
  Enable SSH authorized principals backup when rewriting all keys:
  ```ini
  SSH_AUTHORIZED_PRINCIPALS_BACKUP = true
  ```

- <a name="server.SSH_TRUSTED_USER_CA_KEYS" href="#server.SSH_TRUSTED_USER_CA_KEYS">`server.SSH_TRUSTED_USER_CA_KEYS`</a>:
  Public keys of certificate authorities that are trusted to sign user certificates for authentication.
  Multiple keys should be comma separated, e.g. `"ssh-<algorithm> <key>"` or `"ssh-<algorithm> <key1>, ssh-<algorithm> <key2>"`.
  For more information see `TrustedUserCAKeys` in the `sshd_config` manpages.:
  ```ini
  SSH_TRUSTED_USER_CA_KEYS =
  ```

- <a name="server.SSH_TRUSTED_USER_CA_KEYS_FILENAME" href="#server.SSH_TRUSTED_USER_CA_KEYS_FILENAME">`server.SSH_TRUSTED_USER_CA_KEYS_FILENAME`</a>:
  Absolute path of the `TrustedUserCaKeys` file Forgejo will manage.
  By default this is <a href="#RUN_USER">`RUN_USER`</a>`/.ssh/gitea-trusted-user-ca-keys.pem`.
  If you're running your own ssh server and you want to use the Forgejo managed file you'll also need to modify your `sshd_config` to point to this file.
  The official docker image will automatically work without further configuration.:
  ```ini
  SSH_TRUSTED_USER_CA_KEYS_FILENAME =
  ```

- <a name="server.SSH_EXPOSE_ANONYMOUS" href="#server.SSH_EXPOSE_ANONYMOUS">`server.SSH_EXPOSE_ANONYMOUS`</a>:
  Enable exposure of SSH clone URL to anonymous visitors:
  ```ini
  SSH_EXPOSE_ANONYMOUS = false
  ```

- <a name="server.SSH_AUTHORIZED_KEYS_COMMAND_TEMPLATE" href="#server.SSH_AUTHORIZED_KEYS_COMMAND_TEMPLATE">`server.SSH_AUTHORIZED_KEYS_COMMAND_TEMPLATE`</a>:
  Command template for authorized keys entries:
  ```ini
  SSH_AUTHORIZED_KEYS_COMMAND_TEMPLATE = {{.AppPath}} --config={{.CustomConf}} serv key-{{.Key.ID}}
  ```

- <a name="server.SSH_PER_WRITE_TIMEOUT" href="#server.SSH_PER_WRITE_TIMEOUT">`server.SSH_PER_WRITE_TIMEOUT`</a>:
  Timeout for any write to ssh connections.
  If `-1`, all timeouts are disabled.
  Formatted as strings such as `"30s"`.
  If omitted, it is the value of <a href="#server.PER_WRITE_TIMEOUT">`PER_WRITE_TIMEOUT`</a>.:
  ```ini
  SSH_PER_WRITE_TIMEOUT =
  ```

- <a name="server.SSH_PER_WRITE_PER_KB_TIMEOUT" href="#server.SSH_PER_WRITE_PER_KB_TIMEOUT">`server.SSH_PER_WRITE_PER_KB_TIMEOUT`</a>:
  Timeout per Kb written to ssh connections.
  If `-1`, all timeouts are disabled.
  Formatted as strings such as `"30s"`.
  If omitted, it is the value of <a href="#server.PER_WRITE_PER_KB_TIMEOUT">`PER_WRITE_PER_KB_TIMEOUT`</a>.:
  ```ini
  SSH_PER_WRITE_PER_KB_TIMEOUT =
  ```

- <a name="server.MINIMUM_KEY_SIZE_CHECK" href="#server.MINIMUM_KEY_SIZE_CHECK">`server.MINIMUM_KEY_SIZE_CHECK`</a>:
  Whether to check minimum key size with corresponding type:
  ```ini
  MINIMUM_KEY_SIZE_CHECK = false
  ```

- <a name="server.OFFLINE_MODE" href="#server.OFFLINE_MODE">`server.OFFLINE_MODE`</a>:
  Disable CDN even in `"prod"` mode (<a href="#RUN_MODE">`RUN_MODE`</a>):
  ```ini
  OFFLINE_MODE = true
  ```

- <a name="server.ENABLE_ACME" href="#server.ENABLE_ACME">`server.ENABLE_ACME`</a>:
  TLS Settings: Either ACME or manual
  (Other common TLS configuration are found before):
  ```ini
  ENABLE_ACME = false
  ```

- <a name="server.ACME_URL" href="#server.ACME_URL">`server.ACME_URL`</a>:
  ACME automatic TLS settings
  ACME directory URL (e.g. LetsEncrypt's staging/testing URL: https://acme-staging-v02.api.letsencrypt.org/directory)
  Leave empty to default to LetsEncrypt's (production) URL:
  ```ini
  ACME_URL =
  ```

- <a name="server.ACME_ACCEPTTOS" href="#server.ACME_ACCEPTTOS">`server.ACME_ACCEPTTOS`</a>:
  Explicitly accept the ACME's TOS. The specific TOS cannot be retrieved at the moment.:
  ```ini
  ACME_ACCEPTTOS = false
  ```

- <a name="server.ACME_CA_ROOT" href="#server.ACME_CA_ROOT">`server.ACME_CA_ROOT`</a>:
  If the ACME CA is not in your system's CA trust chain, it can be manually added here:
  ```ini
  ACME_CA_ROOT =
  ```

- <a name="server.ACME_EMAIL" href="#server.ACME_EMAIL">`server.ACME_EMAIL`</a>:
  Email used for the ACME registration service
  Can be left blank to initialize at first run and use the cached value:
  ```ini
  ACME_EMAIL =
  ```

- <a name="server.ACME_DIRECTORY" href="#server.ACME_DIRECTORY">`server.ACME_DIRECTORY`</a>:
  ACME live directory (not to be confused with ACME directory URL: ACME_URL)
  (Refer to caddy's ACME manager https://github.com/caddyserver/certmagic):
  ```ini
  ACME_DIRECTORY = https
  ```

- <a name="server.CERT_FILE" href="#server.CERT_FILE">`server.CERT_FILE`</a>:
  Manual TLS settings: (Only applicable if ENABLE_ACME=false)
  Generate steps:
  $ ./forgejo cert -ca=true -duration=8760h0m0s -host=myhost.example.com
  Or from a .pfx file exported from the Windows certificate store (do
  not forget to export the private key):
  $ openssl pkcs12 -in cert.pfx -out cert.pem -nokeys
  $ openssl pkcs12 -in cert.pfx -out key.pem -nocerts -nodes
  Paths are relative to CUSTOM_PATH:
  ```ini
  CERT_FILE = https/cert.pem
  ```

- <a name="server.KEY_FILE" href="#server.KEY_FILE">`server.KEY_FILE`</a>:
  ```ini
  KEY_FILE = https/key.pem
  ```

- <a name="server.STATIC_ROOT_PATH" href="#server.STATIC_ROOT_PATH">`server.STATIC_ROOT_PATH`</a>:
  Root directory containing templates and static files.
  Defaults to the built-in value of _<a href="#StaticRootPath">`StaticRootPath`</a>_, which by default the path where Forgejo is executed.:
  ```ini
  STATIC_ROOT_PATH =
  ```

- <a name="server.APP_DATA_PATH" href="#server.APP_DATA_PATH">`server.APP_DATA_PATH`</a>:
  Default path for App data.
  Relative paths will be made absolute against the _<a href="#AppWorkPath">`AppWorkPath`</a>_.:
  ```ini
  APP_DATA_PATH = data
  ```

- <a name="server.ENABLE_GZIP" href="#server.ENABLE_GZIP">`server.ENABLE_GZIP`</a>:
  Enable gzip compression for runtime-generated content, static resources excluded:
  ```ini
  ENABLE_GZIP = false
  ```

- <a name="server.ENABLE_PPROF" href="#server.ENABLE_PPROF">`server.ENABLE_PPROF`</a>:
  Application profiling (memory and cpu)
  For "web" command it listens on localhost:6060
  For "serve" command it dumps to disk at PPROF_DATA_PATH as (cpuprofile|memprofile)_<username>_<temporary id>:
  ```ini
  ENABLE_PPROF = false
  ```

- <a name="server.PPROF_DATA_PATH" href="#server.PPROF_DATA_PATH">`server.PPROF_DATA_PATH`</a>:
  PProf data path, use an absolute path when you start Forgejo as service.'
  Relative paths will be made absolute against the _<a href="#AppWorkPath">`AppWorkPath`</a>_.:
  ```ini
  PPROF_DATA_PATH = data/tmp/pprof
  ```

- <a name="server.LANDING_PAGE" href="#server.LANDING_PAGE">`server.LANDING_PAGE`</a>:
  Landing page, can be "home", "explore", "organizations", "login", or any URL such as "/org/repo" or even "https://anotherwebsite.com"
  The "login" choice is not a security measure but just a UI flow change, use REQUIRE_SIGNIN_VIEW to force users to log in:
  ```ini
  LANDING_PAGE = home
  ```

- <a name="server.LFS_START_SERVER" href="#server.LFS_START_SERVER">`server.LFS_START_SERVER`</a>:
  Enables git-lfs support. true or false, default is false:
  ```ini
  LFS_START_SERVER = false
  ```

- <a name="server.LFS_JWT_SECRET" href="#server.LFS_JWT_SECRET">`server.LFS_JWT_SECRET`</a>:
  LFS authentication secret, change this yourself:
  ```ini
  LFS_JWT_SECRET =
  ```

- <a name="server.LFS_JWT_SECRET_URI" href="#server.LFS_JWT_SECRET_URI">`server.LFS_JWT_SECRET_URI`</a>:
  Alternative location to specify LFS authentication secret. You cannot specify both this and LFS_JWT_SECRET, and must pick one:
  ```ini
  LFS_JWT_SECRET_URI = file:/etc/gitea/lfs_jwt_secret
  ```

- <a name="server.LFS_HTTP_AUTH_EXPIRY" href="#server.LFS_HTTP_AUTH_EXPIRY">`server.LFS_HTTP_AUTH_EXPIRY`</a>:
  LFS authentication validity period (in time.Duration), pushes taking longer than this may fail:
  ```ini
  LFS_HTTP_AUTH_EXPIRY = 24h
  ```

- <a name="server.LFS_MAX_FILE_SIZE" href="#server.LFS_MAX_FILE_SIZE">`server.LFS_MAX_FILE_SIZE`</a>:
  Maximum allowed LFS file size in bytes (Set to 0 for no limit):
  ```ini
  LFS_MAX_FILE_SIZE = 0
  ```

- <a name="server.LFS_LOCKS_PAGING_NUM" href="#server.LFS_LOCKS_PAGING_NUM">`server.LFS_LOCKS_PAGING_NUM`</a>:
  Maximum number of locks returned per page:
  ```ini
  LFS_LOCKS_PAGING_NUM = 50
  ```

- <a name="server.LFS_MAX_BATCH_SIZE" href="#server.LFS_MAX_BATCH_SIZE">`server.LFS_MAX_BATCH_SIZE`</a>:
  When clients make lfs batch requests, reject them if there are more pointers than this number
  zero means 'unlimited':
  ```ini
  LFS_MAX_BATCH_SIZE = 0
  ```

- <a name="server.ALLOW_GRACEFUL_RESTARTS" href="#server.ALLOW_GRACEFUL_RESTARTS">`server.ALLOW_GRACEFUL_RESTARTS`</a>:
  Allow graceful restarts using SIGHUP to fork:
  ```ini
  ALLOW_GRACEFUL_RESTARTS = true
  ```

- <a name="server.GRACEFUL_HAMMER_TIME" href="#server.GRACEFUL_HAMMER_TIME">`server.GRACEFUL_HAMMER_TIME`</a>:
  After a restart the parent will finish ongoing requests before
  shutting down. Force shutdown if this process takes longer than this delay.
  set to a negative value to disable:
  ```ini
  GRACEFUL_HAMMER_TIME = 60s
  ```

- <a name="server.STARTUP_TIMEOUT" href="#server.STARTUP_TIMEOUT">`server.STARTUP_TIMEOUT`</a>:
  Allows the setting of a startup timeout and waithint for Windows as SVC service
  0 disables this:
  ```ini
  STARTUP_TIMEOUT = 0
  ```

- <a name="server.STATIC_CACHE_TIME" href="#server.STATIC_CACHE_TIME">`server.STATIC_CACHE_TIME`</a>:
  Static resources, includes resources on custom/, public/ and all uploaded avatars web browser cache time. Note that this cache is disabled when RUN_MODE is "dev". Default is 6h:
  ```ini
  STATIC_CACHE_TIME = 6h
  ```

## <a name="database" href="#database">Database</a>
The configuration options and their default values depend on the database to be used:
- SQLite configuration:
  ```ini
  DB_TYPE = sqlite3
  ;PATH = data/forgejo.db
  ;SQLITE_TIMEOUT = 500
  ;SQLITE_JOURNAL_MODE = ; defaults to sqlite database default (often DELETE), can be used to enable WAL mode. https://www.sqlite.org/pragma.html#pragma_journal_mode
  ```
- MySQL/MariaDB configuration
  ```ini
  DB_TYPE = mysql
  ;HOST = 127.0.0.1:3306 ; can use socket e.g. /var/run/mysqld/mysqld.sock
  ;NAME = gitea
  ;USER = root
  ;PASSWD = ;Use PASSWD = `your password` for quoting if you use special characters in the password.
  ;SSL_MODE = false ; either "false" (default), "true", or "skip-verify"
  ;CHARSET_COLLATION = ; Empty as default, Forgejo will try to find a case-sensitive collation. Don't change it unless you clearly know what you need.
  ```
- Postgres configuration
  ```ini
  DB_TYPE = postgres
  ;HOST = 127.0.0.1:5432 ; can use socket e.g. /var/run/postgresql/
  ;NAME = gitea
  ;USER = root
  ;PASSWD =
  ;SCHEMA =
  ;SSL_MODE=disable ;either "disable" (default), "require", or "verify-full"
  ```

```ini
[database]
```

- <a name="database.DB_TYPE" href="#database.DB_TYPE">`database.DB_TYPE`</a>:
  Database type, either `"sqlite3"`, `"mySQL"` or `"postgres"`.:
  ```ini
  DB_TYPE = sqlite3
  ```

- <a name="database.PATH" href="#database.PATH">`database.PATH`</a>:
  Path to the database, only used for `DB_TYPE = sqlite3`:
  ```ini
  PATH = data/forgejo.db
  ```

- <a name="database.SQLITE_TIMEOUT" href="#database.SQLITE_TIMEOUT">`database.SQLITE_TIMEOUT`</a>:
  Query timeout:
  ```ini
  SQLITE_TIMEOUT = 500
  ```

- <a name="database.SQLITE_JOURNAL_MODE" href="#database.SQLITE_JOURNAL_MODE">`database.SQLITE_JOURNAL_MODE`</a>:
  Journal mode, only used for `DB_TYPE = sqlite3`.:
  ```ini
  SQLITE_JOURNAL_MODE =
  ```

- <a name="database.HOST" href="#database.HOST">`database.HOST`</a>:
  Database host or socket.
  Ignored for `DB_TYPE = sqlite3`.
  Defaults to `127.0.0.1:3306` for `DB_TYPE = mysql` and `127.0.0.1:5432` for `DB_TYPE = postgres`.
  Can use socket, e.g. `/var/run/mysqld/mysqld.sock` or `/var/run/postgresql/`.:
  ```ini
  HOST =
  ```

- <a name="database.NAME" href="#database.NAME">`database.NAME`</a>:
  Database name. Ignored for `DB_TYPE = sqlite3`.:
  ```ini
  NAME = gitea
  ```

- <a name="database.USER" href="#database.USER">`database.USER`</a>:
  Database user. Ignored for `DB_TYPE = sqlite3`.:
  ```ini
  USER = root
  ```

- <a name="database.PASSWD" href="#database.PASSWD">`database.PASSWD`</a>:
  Database user's password.
  Use 
  ```ini
  PASSWD = `your password`
  ```
  for quoting if you use special characters in the password.':
  ```ini
  PASSWD =
  ```

- <a name="database.SSL_MODE" href="#database.SSL_MODE">`database.SSL_MODE`</a>:
  SSL mode.
  Ignored for `DB_TYPE = sqlite3`.
  For `DB_TYPE = mysql` either "false" (default), "true", or "skip-verify"'.
  For `DB_TYPE = postgres` either "disable" (default), "require", or "verify-full".:
  ```ini
  SSL_MODE =
  ```

- <a name="database.CHARSET_COLLATION" href="#database.CHARSET_COLLATION">`database.CHARSET_COLLATION`</a>:
  Charset collation, only used for `DB_TYPE = mysql`.
  Empty as default, Forgejo will try to find a case-sensitive collation. Don't change it unless you clearly know what you need.:
  ```ini
  CHARSET_COLLATION =
  ```

- <a name="database.SCHEMA" href="#database.SCHEMA">`database.SCHEMA`</a>:
  Database schema, only used for `DB_TYPE = postgres`:
  ```ini
  SCHEMA =
  ```

- <a name="database.ITERATE_BUFFER_SIZE" href="#database.ITERATE_BUFFER_SIZE">`database.ITERATE_BUFFER_SIZE`</a>:
  Size of iterate buffer:
  ```ini
  ITERATE_BUFFER_SIZE = 50
  ```

- <a name="database.LOG_SQL" href="#database.LOG_SQL">`database.LOG_SQL`</a>:
  Show the database generated SQL:
  ```ini
  LOG_SQL = false
  ```

- <a name="database.DB_RETRIES" href="#database.DB_RETRIES">`database.DB_RETRIES`</a>:
  Maximum number of database connect retries:
  ```ini
  DB_RETRIES = 10
  ```

- <a name="database.DB_RETRY_BACKOFF" href="#database.DB_RETRY_BACKOFF">`database.DB_RETRY_BACKOFF`</a>:
  Backoff time per database retry (time.Duration):
  ```ini
  DB_RETRY_BACKOFF = 3s
  ```

- <a name="database.MAX_IDLE_CONNS" href="#database.MAX_IDLE_CONNS">`database.MAX_IDLE_CONNS`</a>:
  Max idle database connections on connection pool:
  ```ini
  MAX_IDLE_CONNS = 2
  ```

- <a name="database.CONN_MAX_LIFETIME" href="#database.CONN_MAX_LIFETIME">`database.CONN_MAX_LIFETIME`</a>:
  Database connection max life time, default is `0` or `3s` mysql (See #6804 & #7071 for reasoning).:
  ```ini
  CONN_MAX_LIFETIME = 3s
  ```

- <a name="database.CONN_MAX_IDLETIME" href="#database.CONN_MAX_IDLETIME">`database.CONN_MAX_IDLETIME`</a>:
  Database connection max idle time. `0` prevents closing due to idle time.:
  ```ini
  CONN_MAX_IDLETIME = 0
  ```

- <a name="database.MAX_OPEN_CONNS" href="#database.MAX_OPEN_CONNS">`database.MAX_OPEN_CONNS`</a>:
  Database maximum number of open connections, default is `100` which is the lowest default from Postgres (MariaDB and MySQL default to `151`). Ensure you only increase the value if you configured your database server accordingly:
  ```ini
  MAX_OPEN_CONNS = 100
  ```

- <a name="database.AUTO_MIGRATION" href="#database.AUTO_MIGRATION">`database.AUTO_MIGRATION`</a>:
  Whether execute database models migrations automatically:
  ```ini
  AUTO_MIGRATION = true
  ```

- <a name="database.SLOW_QUERY_TRESHOLD" href="#database.SLOW_QUERY_TRESHOLD">`database.SLOW_QUERY_TRESHOLD`</a>:
  Threshold value (in seconds) beyond which query execution time is logged as a warning in the xorm logger:
  ```ini
  SLOW_QUERY_TRESHOLD = 5s
  ```

## <a name="security" href="#security">Security</a>

```ini
[security]
```

- <a name="security.INSTALL_LOCK" href="#security.INSTALL_LOCK">`security.INSTALL_LOCK`</a>:
  Whether the installer is disabled (set to true to disable the installer):
  ```ini
  INSTALL_LOCK = false
  ```

- <a name="security.SECRET_KEY" href="#security.SECRET_KEY">`security.SECRET_KEY`</a>:
  Global secret key that will be used
  This key is VERY IMPORTANT. If you lose it, the data encrypted by it (like 2FA secret) can't be decrypted anymore.:
  ```ini
  SECRET_KEY =
  ```

- <a name="security.SECRET_KEY_URI" href="#security.SECRET_KEY_URI">`security.SECRET_KEY_URI`</a>:
  Alternative location to specify secret key, instead of this file; you cannot specify both this and SECRET_KEY, and must pick one
  This key is VERY IMPORTANT. If you lose it, the data encrypted by it (like 2FA secret) can't be decrypted anymore.:
  ```ini
  SECRET_KEY_URI = file:/etc/gitea/secret_key
  ```

- <a name="security.INTERNAL_TOKEN" href="#security.INTERNAL_TOKEN">`security.INTERNAL_TOKEN`</a>:
  Secret used to validate communication within Forgejo binary:
  ```ini
  INTERNAL_TOKEN =
  ```

- <a name="security.INTERNAL_TOKEN_URI" href="#security.INTERNAL_TOKEN_URI">`security.INTERNAL_TOKEN_URI`</a>:
  Alternative location to specify internal token, instead of this file; you cannot specify both this and INTERNAL_TOKEN, and must pick one:
  ```ini
  INTERNAL_TOKEN_URI = file:/etc/gitea/internal_token
  ```

- <a name="security.LOGIN_REMEMBER_DAYS" href="#security.LOGIN_REMEMBER_DAYS">`security.LOGIN_REMEMBER_DAYS`</a>:
  How long to remember that a user is logged in before requiring relogin (in days):
  ```ini
  LOGIN_REMEMBER_DAYS = 31
  ```

- <a name="security.COOKIE_REMEMBER_NAME" href="#security.COOKIE_REMEMBER_NAME">`security.COOKIE_REMEMBER_NAME`</a>:
  Name of cookie used to store authentication information:
  ```ini
  COOKIE_REMEMBER_NAME = gitea_incredible
  ```

- <a name="security.REVERSE_PROXY_AUTHENTICATION_USER" href="#security.REVERSE_PROXY_AUTHENTICATION_USER">`security.REVERSE_PROXY_AUTHENTICATION_USER`</a>:
  Reverse proxy authentication header name of user name, email, and full name:
  ```ini
  REVERSE_PROXY_AUTHENTICATION_USER = X-WEBAUTH-USER
  ```

- <a name="security.REVERSE_PROXY_AUTHENTICATION_EMAIL" href="#security.REVERSE_PROXY_AUTHENTICATION_EMAIL">`security.REVERSE_PROXY_AUTHENTICATION_EMAIL`</a>:
  ```ini
  REVERSE_PROXY_AUTHENTICATION_EMAIL = X-WEBAUTH-EMAIL
  ```

- <a name="security.REVERSE_PROXY_AUTHENTICATION_FULL_NAME" href="#security.REVERSE_PROXY_AUTHENTICATION_FULL_NAME">`security.REVERSE_PROXY_AUTHENTICATION_FULL_NAME`</a>:
  ```ini
  REVERSE_PROXY_AUTHENTICATION_FULL_NAME = X-WEBAUTH-FULLNAME
  ```

- <a name="security.REVERSE_PROXY_LIMIT" href="#security.REVERSE_PROXY_LIMIT">`security.REVERSE_PROXY_LIMIT`</a>:
  Interpret X-Forwarded-For header or the X-Real-IP header and set this as the remote IP for the request:
  ```ini
  REVERSE_PROXY_LIMIT = 1
  ```

- <a name="security.REVERSE_PROXY_TRUSTED_PROXIES" href="#security.REVERSE_PROXY_TRUSTED_PROXIES">`security.REVERSE_PROXY_TRUSTED_PROXIES`</a>:
  List of IP addresses and networks separated by comma of trusted proxy servers. Use `*` to trust all:
  ```ini
  REVERSE_PROXY_TRUSTED_PROXIES = 127.0.0.0/8,::1/128
  ```

- <a name="security.MIN_PASSWORD_LENGTH" href="#security.MIN_PASSWORD_LENGTH">`security.MIN_PASSWORD_LENGTH`</a>:
  The minimum password length for new Users:
  ```ini
  MIN_PASSWORD_LENGTH = 8
  ```

- <a name="security.IMPORT_LOCAL_PATHS" href="#security.IMPORT_LOCAL_PATHS">`security.IMPORT_LOCAL_PATHS`</a>:
  Set to true to allow users to import local server paths:
  ```ini
  IMPORT_LOCAL_PATHS = false
  ```

- <a name="security.DISABLE_GIT_HOOKS" href="#security.DISABLE_GIT_HOOKS">`security.DISABLE_GIT_HOOKS`</a>:
  Set to false to allow users with git hook privileges to create custom git hooks.
  Custom git hooks can be used to perform arbitrary code execution on the host operating system.
  This enables the users to access and modify this config file and the Forgejo database and interrupt the Forgejo service.
  By modifying the Forgejo database, users can gain Forgejo administrator privileges.
  It also enables them to access other resources available to the user on the operating system that is running the Forgejo instance and perform arbitrary actions in the name of the Forgejo OS user.
  WARNING: This maybe harmful to you website or your operating system.
  WARNING: Setting this to true does not change existing hooks in git repos; adjust it before if necessary:
  ```ini
  DISABLE_GIT_HOOKS = true
  ```

- <a name="security.DISABLE_WEBHOOKS" href="#security.DISABLE_WEBHOOKS">`security.DISABLE_WEBHOOKS`</a>:
  Set to true to disable webhooks feature:
  ```ini
  DISABLE_WEBHOOKS = false
  ```

- <a name="security.ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET" href="#security.ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET">`security.ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET`</a>:
  Set to false to allow pushes to Forgejo repositories despite having an incomplete environment - NOT RECOMMENDED:
  ```ini
  ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET = true
  ```

- <a name="security.PASSWORD_COMPLEXITY" href="#security.PASSWORD_COMPLEXITY">`security.PASSWORD_COMPLEXITY`</a>:
  Comma separated list of character classes required to pass minimum complexity.
  If left empty or no valid values are specified, the default is off (no checking)
  Classes include "lower,upper,digit,spec":
  ```ini
  PASSWORD_COMPLEXITY = off
  ```

- <a name="security.PASSWORD_HASH_ALGO" href="#security.PASSWORD_HASH_ALGO">`security.PASSWORD_HASH_ALGO`</a>:
  Password Hash algorithm, either "argon2", "pbkdf2"/"pbkdf2_v2", "pbkdf2_hi", "scrypt" or "bcrypt":
  ```ini
  PASSWORD_HASH_ALGO = pbkdf2_hi
  ```

- <a name="security.CSRF_COOKIE_HTTP_ONLY" href="#security.CSRF_COOKIE_HTTP_ONLY">`security.CSRF_COOKIE_HTTP_ONLY`</a>:
  Set false to allow JavaScript to read CSRF cookie:
  ```ini
  CSRF_COOKIE_HTTP_ONLY = true
  ```

- <a name="security.PASSWORD_CHECK_PWN" href="#security.PASSWORD_CHECK_PWN">`security.PASSWORD_CHECK_PWN`</a>:
  Validate against https://haveibeenpwned.com/Passwords to see if a password has been exposed:
  ```ini
  PASSWORD_CHECK_PWN = false
  ```

- <a name="security.SUCCESSFUL_TOKENS_CACHE_SIZE" href="#security.SUCCESSFUL_TOKENS_CACHE_SIZE">`security.SUCCESSFUL_TOKENS_CACHE_SIZE`</a>:
  Cache successful token hashes. API tokens are stored in the DB as pbkdf2 hashes however, this means that there is a potentially significant hashing load when there are multiple API operations.
  This cache will store the successfully hashed tokens in a LRU cache as a balance between performance and security:
  ```ini
  SUCCESSFUL_TOKENS_CACHE_SIZE = 20
  ```

- <a name="security.DISABLE_QUERY_AUTH_TOKEN" href="#security.DISABLE_QUERY_AUTH_TOKEN">`security.DISABLE_QUERY_AUTH_TOKEN`</a>:
  Reject API tokens sent in URL query string (Accept Header-based API tokens only). This avoids security vulnerabilities
  stemming from cached/logged plain-text API tokens.
  In future releases, this will become the default behavior:
  ```ini
  DISABLE_QUERY_AUTH_TOKEN = false
  ```

## <a name="camo" href="#camo">Camo</a>

```ini
[camo]
```

- <a name="camo.ENABLED" href="#camo.ENABLED">`camo.ENABLED`</a>:
  At the moment we only support images
  if the camo is enabled:
  ```ini
  ENABLED = false
  ```

- <a name="camo.SERVER_URL" href="#camo.SERVER_URL">`camo.SERVER_URL`</a>:
  url to a camo image proxy, it **is required** if camo is enabled:
  ```ini
  SERVER_URL =
  ```

- <a name="camo.HMAC_KEY" href="#camo.HMAC_KEY">`camo.HMAC_KEY`</a>:
  HMAC to encode urls with, it **is required** if camo is enabled:
  ```ini
  HMAC_KEY =
  ```

- <a name="camo.ALWAYS" href="#camo.ALWAYS">`camo.ALWAYS`</a>:
  Set to true to use camo for https too lese only non https urls are proxyed
  ALLWAYS is deprecated and will be removed in the future:
  ```ini
  ALWAYS = false
  ```

## <a name="oauth2" href="#oauth2">Oauth2</a>

```ini
[oauth2]
```

- <a name="oauth2.ENABLED" href="#oauth2.ENABLED">`oauth2.ENABLED`</a>:
  Enables OAuth2 provider:
  ```ini
  ENABLED = true
  ```

- <a name="oauth2.JWT_SIGNING_ALGORITHM" href="#oauth2.JWT_SIGNING_ALGORITHM">`oauth2.JWT_SIGNING_ALGORITHM`</a>:
  Algorithm used to sign OAuth2 tokens. Valid values: HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512, EdDSA:
  ```ini
  JWT_SIGNING_ALGORITHM = RS256
  ```

- <a name="oauth2.JWT_SIGNING_PRIVATE_KEY_FILE" href="#oauth2.JWT_SIGNING_PRIVATE_KEY_FILE">`oauth2.JWT_SIGNING_PRIVATE_KEY_FILE`</a>:
  Private key file path used to sign OAuth2 tokens. The path is relative to APP_DATA_PATH.
  This setting is only needed if JWT_SIGNING_ALGORITHM is set to RS256, RS384, RS512, ES256, ES384 or ES512.
  The file must contain a RSA or ECDSA private key in the PKCS8 format. If no key exists a 4096 bit key will be created for you.:
  ```ini
  JWT_SIGNING_PRIVATE_KEY_FILE = jwt/private.pem
  ```

- <a name="oauth2.JWT_SECRET" href="#oauth2.JWT_SECRET">`oauth2.JWT_SECRET`</a>:
  OAuth2 authentication secret for access and refresh tokens, change this yourself to a unique string. CLI generate option is helpful in this case. https://docs.gitea.io/en-us/command-line/#generate
  This setting is only needed if JWT_SIGNING_ALGORITHM is set to HS256, HS384 or HS512.:
  ```ini
  JWT_SECRET =
  ```

- <a name="oauth2.JWT_SECRET_URI" href="#oauth2.JWT_SECRET_URI">`oauth2.JWT_SECRET_URI`</a>:
  Alternative location to specify OAuth2 authentication secret. You cannot specify both this and JWT_SECRET, and must pick one.:
  ```ini
  JWT_SECRET_URI = file:/etc/gitea/oauth2_jwt_secret
  ```

- <a name="oauth2.ACCESS_TOKEN_EXPIRATION_TIME" href="#oauth2.ACCESS_TOKEN_EXPIRATION_TIME">`oauth2.ACCESS_TOKEN_EXPIRATION_TIME`</a>:
  Lifetime of an OAuth2 access token in seconds:
  ```ini
  ACCESS_TOKEN_EXPIRATION_TIME = 3600
  ```

- <a name="oauth2.REFRESH_TOKEN_EXPIRATION_TIME" href="#oauth2.REFRESH_TOKEN_EXPIRATION_TIME">`oauth2.REFRESH_TOKEN_EXPIRATION_TIME`</a>:
  Lifetime of an OAuth2 refresh token in hours:
  ```ini
  REFRESH_TOKEN_EXPIRATION_TIME = 730
  ```

- <a name="oauth2.INVALIDATE_REFRESH_TOKENS" href="#oauth2.INVALIDATE_REFRESH_TOKENS">`oauth2.INVALIDATE_REFRESH_TOKENS`</a>:
  Check if refresh token got already used:
  ```ini
  INVALIDATE_REFRESH_TOKENS = false
  ```

- <a name="oauth2.MAX_TOKEN_LENGTH" href="#oauth2.MAX_TOKEN_LENGTH">`oauth2.MAX_TOKEN_LENGTH`</a>:
  Maximum length of oauth2 token/cookie stored on server:
  ```ini
  MAX_TOKEN_LENGTH = 32767
  ```

- <a name="oauth2.DEFAULT_APPLICATIONS" href="#oauth2.DEFAULT_APPLICATIONS">`oauth2.DEFAULT_APPLICATIONS`</a>:
  Pre-register OAuth2 applications for some universally useful services
  * https://github.com/hickford/git-credential-oauth
  * https://github.com/git-ecosystem/git-credential-manager
  * https://gitea.com/gitea/tea:
  ```ini
  DEFAULT_APPLICATIONS = git-credential-oauth, git-credential-manager, tea
  ```

## <a name="log" href="#log">Log</a>

```ini
[log]
```

- <a name="log.ROOT_PATH" href="#log.ROOT_PATH">`log.ROOT_PATH`</a>:
  Root path for the log files - defaults to %(GITEA_WORK_DIR)/log:
  ```ini
  ROOT_PATH =
  ```

- <a name="log.MODE" href="#log.MODE">`log.MODE`</a>:
  Main Logger
  Either "console", "file" or "conn", default is "console"
  Use comma to separate multiple modes, e.g. "console, file":
  ```ini
  MODE = console
  ```

- <a name="log.LEVEL" href="#log.LEVEL">`log.LEVEL`</a>:
  Either "Trace", "Debug", "Info", "Warn", "Error" or "None", default is "Info":
  ```ini
  LEVEL = Info
  ```

- <a name="log.STACKTRACE_LEVEL" href="#log.STACKTRACE_LEVEL">`log.STACKTRACE_LEVEL`</a>:
  Print Stacktrace with logs (rarely helpful, do not set) Either "Trace", "Debug", "Info", "Warn", "Error", default is "None":
  ```ini
  STACKTRACE_LEVEL = None
  ```

- <a name="log.BUFFER_LEN" href="#log.BUFFER_LEN">`log.BUFFER_LEN`</a>:
  Buffer length of the channel, keep it as it is if you don't know what it is:
  ```ini
  BUFFER_LEN = 10000
  ```

- <a name="log.ENABLE_SSH_LOG" href="#log.ENABLE_SSH_LOG">`log.ENABLE_SSH_LOG`</a>:
  Collect SSH logs (Creates log from ssh git request):
  ```ini
  ENABLE_SSH_LOG = false
  ```

- <a name="log.REQUEST_ID_HEADERS" href="#log.REQUEST_ID_HEADERS">`log.REQUEST_ID_HEADERS`</a>:
  Access Logger (Creates log in NCSA common log format)
  Print request id which parsed from request headers in access log, when access log is enabled.
  E.g:
  * In request header:         `X-Request-ID: test-id-123`
  * Configuration in app.ini:  `REQUEST_ID_HEADERS = X-Request-ID`
  * Print in log:              `127.0.0.1:58384 - - [14/Feb/2023:16:33:51 +0800] "test-id-123"`
  
  If you configure more than one in the .ini file, it will match in the order of configuration,
  and the first match will be finally printed in the log.
  E.g:
  * In request header:         `X-Trace-ID: trace-id-1q2w3e4r`
  * Configuration in app.ini:  `REQUEST_ID_HEADERS = X-Request-ID, X-Trace-ID, X-Req-ID`
  * Print in log:              `127.0.0.1:58384 - - [14/Feb/2023:16:33:51 +0800] "trace-id-1q2w3e4r"`:
  ```ini
  REQUEST_ID_HEADERS =
  ```

- <a name="log.ACCESS_LOG_TEMPLATE" href="#log.ACCESS_LOG_TEMPLATE">`log.ACCESS_LOG_TEMPLATE`</a>:
  Sets the template used to create the access log:
  ```ini
  ACCESS_LOG_TEMPLATE = {{.Ctx.RemoteHost}} - {{.Identity}} {{.Start.Format "[02/Jan/2006:15:04:05 -0700]" }} "{{.Ctx.Req.Method}} {{.Ctx.Req.URL.RequestURI}} {{.Ctx.Req.Proto}}" {{.ResponseWriter.Status}} {{.ResponseWriter.Size}} "{{.Ctx.Req.Referer}}" "{{.Ctx.Req.UserAgent}}"
  ```

### <a name="log.logger" href="#log.logger">Log logger</a>
Sub logger modes, a single comma means use default MODE above, empty means disable it

```ini
[log.logger]
```

- <a name="log.logger.access.MODE" href="#log.logger.access.MODE">`log.logger.access.MODE`</a>:
  ```ini
  access.MODE =
  ```

- <a name="log.logger.router.MODE" href="#log.logger.router.MODE">`log.logger.router.MODE`</a>:
  ```ini
  router.MODE = ,
  ```

- <a name="log.logger.xorm.MODE" href="#log.logger.xorm.MODE">`log.logger.xorm.MODE`</a>:
  ```ini
  xorm.MODE = ,
  ```

### <a name="log.%(WriterMode)" href="#log.%(WriterMode)">Log %(writermode)</a>
Log modes (aka log writers)

```ini
[log.%(WriterMode)]
```

- <a name="log.%(WriterMode).MODE" href="#log.%(WriterMode).MODE">`log.%(WriterMode).MODE`</a>:
  console/file/conn/...:
  ```ini
  MODE =
  ```

- <a name="log.%(WriterMode).LEVEL" href="#log.%(WriterMode).LEVEL">`log.%(WriterMode).LEVEL`</a>:
  ```ini
  LEVEL =
  ```

- <a name="log.%(WriterMode).FLAGS" href="#log.%(WriterMode).FLAGS">`log.%(WriterMode).FLAGS`</a>:
  stdflags or journald:
  ```ini
  FLAGS =
  ```

- <a name="log.%(WriterMode).EXPRESSION" href="#log.%(WriterMode).EXPRESSION">`log.%(WriterMode).EXPRESSION`</a>:
  ```ini
  EXPRESSION =
  ```

- <a name="log.%(WriterMode).PREFIX" href="#log.%(WriterMode).PREFIX">`log.%(WriterMode).PREFIX`</a>:
  ```ini
  PREFIX =
  ```

- <a name="log.%(WriterMode).COLORIZE" href="#log.%(WriterMode).COLORIZE">`log.%(WriterMode).COLORIZE`</a>:
  ```ini
  COLORIZE = false
  ```

- <a name="log.console.STDERR" href="#log.console.STDERR">`log.console.STDERR`</a>:
  ```ini
  STDERR = false
  ```

### <a name="log.file" href="#log.file">Log file</a>

```ini
[log.file]
```

- <a name="log.file.FILE_NAME" href="#log.file.FILE_NAME">`log.file.FILE_NAME`</a>:
  Set the file_name for the logger. If this is a relative path this will be relative to ROOT_PATH:
  ```ini
  FILE_NAME =
  ```

- <a name="log.file.LOG_ROTATE" href="#log.file.LOG_ROTATE">`log.file.LOG_ROTATE`</a>:
  This enables automated log rotate(switch of following options), default is true:
  ```ini
  LOG_ROTATE = true
  ```

- <a name="log.file.MAX_SIZE_SHIFT" href="#log.file.MAX_SIZE_SHIFT">`log.file.MAX_SIZE_SHIFT`</a>:
  Max size shift of a single file, default is 28 means 1 << 28, 256MB:
  ```ini
  MAX_SIZE_SHIFT = 28
  ```

- <a name="log.file.DAILY_ROTATE" href="#log.file.DAILY_ROTATE">`log.file.DAILY_ROTATE`</a>:
  Segment log daily, default is true:
  ```ini
  DAILY_ROTATE = true
  ```

- <a name="log.file.MAX_DAYS" href="#log.file.MAX_DAYS">`log.file.MAX_DAYS`</a>:
  delete the log file after n days, default is 7:
  ```ini
  MAX_DAYS = 7
  ```

- <a name="log.file.COMPRESS" href="#log.file.COMPRESS">`log.file.COMPRESS`</a>:
  compress logs with gzip:
  ```ini
  COMPRESS = true
  ```

- <a name="log.file.COMPRESSION_LEVEL" href="#log.file.COMPRESSION_LEVEL">`log.file.COMPRESSION_LEVEL`</a>:
  compression level see godoc for compress/gzip:
  ```ini
  COMPRESSION_LEVEL = -1
  ```

### <a name="log.conn" href="#log.conn">Log conn</a>

```ini
[log.conn]
```

- <a name="log.conn.RECONNECT_ON_MSG" href="#log.conn.RECONNECT_ON_MSG">`log.conn.RECONNECT_ON_MSG`</a>:
  Reconnect host for every single message, default is false:
  ```ini
  RECONNECT_ON_MSG = false
  ```

- <a name="log.conn.RECONNECT" href="#log.conn.RECONNECT">`log.conn.RECONNECT`</a>:
  Try to reconnect when connection is lost, default is false:
  ```ini
  RECONNECT = false
  ```

- <a name="log.conn.PROTOCOL" href="#log.conn.PROTOCOL">`log.conn.PROTOCOL`</a>:
  Either "tcp", "unix" or "udp", default is "tcp":
  ```ini
  PROTOCOL = tcp
  ```

- <a name="log.conn.ADDR" href="#log.conn.ADDR">`log.conn.ADDR`</a>:
  Host address:
  ```ini
  ADDR =
  ```

## <a name="git" href="#git">Git</a>

```ini
[git]
```

- <a name="git.PATH" href="#git.PATH">`git.PATH`</a>:
  The path of git executable. If empty, Forgejo searches through the PATH environment.:
  ```ini
  PATH =
  ```

- <a name="git.HOME_PATH" href="#git.HOME_PATH">`git.HOME_PATH`</a>:
  The HOME directory for Git:
  ```ini
  HOME_PATH = %(APP_DATA_PATH)s/home
  ```

- <a name="git.DISABLE_DIFF_HIGHLIGHT" href="#git.DISABLE_DIFF_HIGHLIGHT">`git.DISABLE_DIFF_HIGHLIGHT`</a>:
  Disables highlight of added and removed changes:
  ```ini
  DISABLE_DIFF_HIGHLIGHT = false
  ```

- <a name="git.MAX_GIT_DIFF_LINES" href="#git.MAX_GIT_DIFF_LINES">`git.MAX_GIT_DIFF_LINES`</a>:
  Max number of lines allowed in a single file in diff view:
  ```ini
  MAX_GIT_DIFF_LINES = 1000
  ```

- <a name="git.MAX_GIT_DIFF_LINE_CHARACTERS" href="#git.MAX_GIT_DIFF_LINE_CHARACTERS">`git.MAX_GIT_DIFF_LINE_CHARACTERS`</a>:
  Max number of allowed characters in a line in diff view:
  ```ini
  MAX_GIT_DIFF_LINE_CHARACTERS = 5000
  ```

- <a name="git.MAX_GIT_DIFF_FILES" href="#git.MAX_GIT_DIFF_FILES">`git.MAX_GIT_DIFF_FILES`</a>:
  Max number of files shown in diff view:
  ```ini
  MAX_GIT_DIFF_FILES = 100
  ```

- <a name="git.COMMITS_RANGE_SIZE" href="#git.COMMITS_RANGE_SIZE">`git.COMMITS_RANGE_SIZE`</a>:
  Set the default commits range size:
  ```ini
  COMMITS_RANGE_SIZE = 50
  ```

- <a name="git.BRANCHES_RANGE_SIZE" href="#git.BRANCHES_RANGE_SIZE">`git.BRANCHES_RANGE_SIZE`</a>:
  Set the default branches range size:
  ```ini
  BRANCHES_RANGE_SIZE = 20
  ```

- <a name="git.VERBOSE_PUSH" href="#git.VERBOSE_PUSH">`git.VERBOSE_PUSH`</a>:
  Print out verbose infos on push to stdout:
  ```ini
  VERBOSE_PUSH = true
  ```

- <a name="git.VERBOSE_PUSH_DELAY" href="#git.VERBOSE_PUSH_DELAY">`git.VERBOSE_PUSH_DELAY`</a>:
  Delay before verbose push infos are printed to stdout:
  ```ini
  VERBOSE_PUSH_DELAY = 5s
  ```

- <a name="git.GC_ARGS" href="#git.GC_ARGS">`git.GC_ARGS`</a>:
  Arguments for command `git gc`, e.g. `--aggressive --auto`
  see more on <https://git-scm.com/docs/git-gc/>.:
  ```ini
  GC_ARGS =
  ```

- <a name="git.ENABLE_AUTO_GIT_WIRE_PROTOCOL" href="#git.ENABLE_AUTO_GIT_WIRE_PROTOCOL">`git.ENABLE_AUTO_GIT_WIRE_PROTOCOL`</a>:
  Whether to use git wire protocol version 2 when git version >= 2.18, default is true, set to false when you always want git wire protocol version 1.
  To enable this for Git over SSH when using a OpenSSH server, add `AcceptEnv GIT_PROTOCOL` to your `sshd_config` file.:
  ```ini
  ENABLE_AUTO_GIT_WIRE_PROTOCOL = true
  ```

- <a name="git.PULL_REQUEST_PUSH_MESSAGE" href="#git.PULL_REQUEST_PUSH_MESSAGE">`git.PULL_REQUEST_PUSH_MESSAGE`</a>:
  Respond to pushes to a non-default branch with a URL for creating a Pull Request (if the repository has them enabled):
  ```ini
  PULL_REQUEST_PUSH_MESSAGE = true
  ```

- <a name="git.LARGE_OBJECT_THRESHOLD" href="#git.LARGE_OBJECT_THRESHOLD">`git.LARGE_OBJECT_THRESHOLD`</a>:
  (Go-Git only) Don't cache objects greater than this in memory. (Set to 0 to disable.):
  ```ini
  LARGE_OBJECT_THRESHOLD = 1048576
  ```

- <a name="git.DISABLE_CORE_PROTECT_NTFS" href="#git.DISABLE_CORE_PROTECT_NTFS">`git.DISABLE_CORE_PROTECT_NTFS`</a>:
  Set to true to forcibly set core.protectNTFS=false:
  ```ini
  DISABLE_CORE_PROTECT_NTFS = false
  ```

- <a name="git.DISABLE_PARTIAL_CLONE" href="#git.DISABLE_PARTIAL_CLONE">`git.DISABLE_PARTIAL_CLONE`</a>:
  Disable the usage of using partial clones for git:
  ```ini
  DISABLE_PARTIAL_CLONE = false
  ```

### <a name="git.timeout" href="#git.timeout">Git timeout</a>
Git Operation timeout in seconds

```ini
[git.timeout]
```

- <a name="git.timeout.DEFAULT" href="#git.timeout.DEFAULT">`git.timeout.DEFAULT`</a>:
  ```ini
  DEFAULT = 360
  ```

- <a name="git.timeout.MIGRATE" href="#git.timeout.MIGRATE">`git.timeout.MIGRATE`</a>:
  ```ini
  MIGRATE = 600
  ```

- <a name="git.timeout.MIRROR" href="#git.timeout.MIRROR">`git.timeout.MIRROR`</a>:
  ```ini
  MIRROR = 300
  ```

- <a name="git.timeout.CLONE" href="#git.timeout.CLONE">`git.timeout.CLONE`</a>:
  ```ini
  CLONE = 300
  ```

- <a name="git.timeout.PULL" href="#git.timeout.PULL">`git.timeout.PULL`</a>:
  ```ini
  PULL = 300
  ```

- <a name="git.timeout.GC" href="#git.timeout.GC">`git.timeout.GC`</a>:
  ```ini
  GC = 60
  ```

- <a name="git.timeout.GREP" href="#git.timeout.GREP">`git.timeout.GREP`</a>:
  ```ini
  GREP = 2
  ```

### <a name="git.config" href="#git.config">Git config</a>
Git config options.
This section only does "set" config, a removed config key from this section won't be removed from git config automatically. The format is `some.configKey = value`.

```ini
[git.config]
```

- <a name="git.config.diff.algorithm" href="#git.config.diff.algorithm">`git.config.diff.algorithm`</a>:
  ```ini
  diff.algorithm = histogram
  ```

- <a name="git.config.core.logAllRefUpdates" href="#git.config.core.logAllRefUpdates">`git.config.core.logAllRefUpdates`</a>:
  ```ini
  core.logAllRefUpdates = true
  ```

- <a name="git.config.gc.reflogExpire" href="#git.config.gc.reflogExpire">`git.config.gc.reflogExpire`</a>:
  ```ini
  gc.reflogExpire = 90
  ```

## <a name="service" href="#service">Service</a>

```ini
[service]
```

- <a name="service.ACTIVE_CODE_LIVE_MINUTES" href="#service.ACTIVE_CODE_LIVE_MINUTES">`service.ACTIVE_CODE_LIVE_MINUTES`</a>:
  Time limit to confirm account/email registration:
  ```ini
  ACTIVE_CODE_LIVE_MINUTES = 180
  ```

- <a name="service.RESET_PASSWD_CODE_LIVE_MINUTES" href="#service.RESET_PASSWD_CODE_LIVE_MINUTES">`service.RESET_PASSWD_CODE_LIVE_MINUTES`</a>:
  Time limit to perform the reset of a forgotten password:
  ```ini
  RESET_PASSWD_CODE_LIVE_MINUTES = 180
  ```

- <a name="service.REGISTER_EMAIL_CONFIRM" href="#service.REGISTER_EMAIL_CONFIRM">`service.REGISTER_EMAIL_CONFIRM`</a>:
  Whether a new user needs to confirm their email when registering:
  ```ini
  REGISTER_EMAIL_CONFIRM = false
  ```

- <a name="service.REGISTER_MANUAL_CONFIRM" href="#service.REGISTER_MANUAL_CONFIRM">`service.REGISTER_MANUAL_CONFIRM`</a>:
  Whether a new user needs to be confirmed manually after registration. (Requires <a href="#service.REGISTER_EMAIL_CONFIRM">`REGISTER_EMAIL_CONFIRM`</a> to be disabled.):
  ```ini
  REGISTER_MANUAL_CONFIRM = false
  ```

- <a name="service.EMAIL_DOMAIN_ALLOWLIST" href="#service.EMAIL_DOMAIN_ALLOWLIST">`service.EMAIL_DOMAIN_ALLOWLIST`</a>:
  List of domain names that are allowed to be used to register on a Forgejo instance, wildcard is supported
  eg: gitea.io,example.com,*.mydomain.com:
  ```ini
  EMAIL_DOMAIN_ALLOWLIST =
  ```

- <a name="service.EMAIL_DOMAIN_BLOCKLIST" href="#service.EMAIL_DOMAIN_BLOCKLIST">`service.EMAIL_DOMAIN_BLOCKLIST`</a>:
  Comma-separated list of domain names that are not allowed to be used to register on a Forgejo instance, wildcard is supported:
  ```ini
  EMAIL_DOMAIN_BLOCKLIST =
  ```

- <a name="service.DISABLE_REGISTRATION" href="#service.DISABLE_REGISTRATION">`service.DISABLE_REGISTRATION`</a>:
  Disallow registration, only allow admins to create accounts:
  ```ini
  DISABLE_REGISTRATION = false
  ```

- <a name="service.ALLOW_ONLY_INTERNAL_REGISTRATION" href="#service.ALLOW_ONLY_INTERNAL_REGISTRATION">`service.ALLOW_ONLY_INTERNAL_REGISTRATION`</a>:
  Allow registration only using Forgejo itself, it works only when DISABLE_REGISTRATION is false:
  ```ini
  ALLOW_ONLY_INTERNAL_REGISTRATION = false
  ```

- <a name="service.ALLOW_ONLY_EXTERNAL_REGISTRATION" href="#service.ALLOW_ONLY_EXTERNAL_REGISTRATION">`service.ALLOW_ONLY_EXTERNAL_REGISTRATION`</a>:
  Allow registration only using third-party services, it works only when DISABLE_REGISTRATION is false:
  ```ini
  ALLOW_ONLY_EXTERNAL_REGISTRATION = false
  ```

- <a name="service.REQUIRE_SIGNIN_VIEW" href="#service.REQUIRE_SIGNIN_VIEW">`service.REQUIRE_SIGNIN_VIEW`</a>:
  User must sign in to view anything:
  ```ini
  REQUIRE_SIGNIN_VIEW = false
  ```

- <a name="service.ENABLE_NOTIFY_MAIL" href="#service.ENABLE_NOTIFY_MAIL">`service.ENABLE_NOTIFY_MAIL`</a>:
  Mail notification:
  ```ini
  ENABLE_NOTIFY_MAIL = false
  ```

- <a name="service.ENABLE_BASIC_AUTHENTICATION" href="#service.ENABLE_BASIC_AUTHENTICATION">`service.ENABLE_BASIC_AUTHENTICATION`</a>:
  This setting enables Forgejo to be signed in with HTTP BASIC Authentication using the user's password.
  If you set this to false you will not be able to access the tokens endpoints on the API with your password.
  Please note that setting this to false will not disable OAuth Basic or Basic authentication using a token.:
  ```ini
  ENABLE_BASIC_AUTHENTICATION = true
  ```

- <a name="service.ENABLE_REVERSE_PROXY_AUTHENTICATION" href="#service.ENABLE_REVERSE_PROXY_AUTHENTICATION">`service.ENABLE_REVERSE_PROXY_AUTHENTICATION`</a>:
  More detail: https://github.com/gogits/gogs/issues/165:
  ```ini
  ENABLE_REVERSE_PROXY_AUTHENTICATION = false
  ```

- <a name="service.ENABLE_REVERSE_PROXY_AUTHENTICATION_API" href="#service.ENABLE_REVERSE_PROXY_AUTHENTICATION_API">`service.ENABLE_REVERSE_PROXY_AUTHENTICATION_API`</a>:
  Enable this to allow reverse proxy authentication for API requests, the reverse proxy is responsible for ensuring that no CSRF is possible.:
  ```ini
  ENABLE_REVERSE_PROXY_AUTHENTICATION_API = false
  ```

- <a name="service.ENABLE_REVERSE_PROXY_AUTO_REGISTRATION" href="#service.ENABLE_REVERSE_PROXY_AUTO_REGISTRATION">`service.ENABLE_REVERSE_PROXY_AUTO_REGISTRATION`</a>:
  ```ini
  ENABLE_REVERSE_PROXY_AUTO_REGISTRATION = false
  ```

- <a name="service.ENABLE_REVERSE_PROXY_EMAIL" href="#service.ENABLE_REVERSE_PROXY_EMAIL">`service.ENABLE_REVERSE_PROXY_EMAIL`</a>:
  ```ini
  ENABLE_REVERSE_PROXY_EMAIL = false
  ```

- <a name="service.ENABLE_REVERSE_PROXY_FULL_NAME" href="#service.ENABLE_REVERSE_PROXY_FULL_NAME">`service.ENABLE_REVERSE_PROXY_FULL_NAME`</a>:
  ```ini
  ENABLE_REVERSE_PROXY_FULL_NAME = false
  ```

- <a name="service.ENABLE_CAPTCHA" href="#service.ENABLE_CAPTCHA">`service.ENABLE_CAPTCHA`</a>:
  Enable captcha validation for registration:
  ```ini
  ENABLE_CAPTCHA = false
  ```

- <a name="service.REQUIRE_CAPTCHA_FOR_LOGIN" href="#service.REQUIRE_CAPTCHA_FOR_LOGIN">`service.REQUIRE_CAPTCHA_FOR_LOGIN`</a>:
  Enable this to require captcha validation for login:
  ```ini
  REQUIRE_CAPTCHA_FOR_LOGIN = false
  ```

- <a name="service.REQUIRE_EXTERNAL_REGISTRATION_CAPTCHA" href="#service.REQUIRE_EXTERNAL_REGISTRATION_CAPTCHA">`service.REQUIRE_EXTERNAL_REGISTRATION_CAPTCHA`</a>:
  Requires captcha for external registrations:
  ```ini
  REQUIRE_EXTERNAL_REGISTRATION_CAPTCHA = false
  ```

- <a name="service.REQUIRE_EXTERNAL_REGISTRATION_PASSWORD" href="#service.REQUIRE_EXTERNAL_REGISTRATION_PASSWORD">`service.REQUIRE_EXTERNAL_REGISTRATION_PASSWORD`</a>:
  Requires a password for external registrations:
  ```ini
  REQUIRE_EXTERNAL_REGISTRATION_PASSWORD = false
  ```

- <a name="service.CAPTCHA_TYPE" href="#service.CAPTCHA_TYPE">`service.CAPTCHA_TYPE`</a>:
  Type of captcha you want to use. Options: image, recaptcha, hcaptcha, mcaptcha, cfturnstile.:
  ```ini
  CAPTCHA_TYPE = image
  ```

- <a name="service.RECAPTCHA_URL" href="#service.RECAPTCHA_URL">`service.RECAPTCHA_URL`</a>:
  Change this to use recaptcha.net or other recaptcha service:
  ```ini
  RECAPTCHA_URL = https://www.google.com/recaptcha/
  ```

- <a name="service.RECAPTCHA_SECRET" href="#service.RECAPTCHA_SECRET">`service.RECAPTCHA_SECRET`</a>:
  Enable recaptcha to use Google's recaptcha service
  Go to https://www.google.com/recaptcha/admin to sign up for a key:
  ```ini
  RECAPTCHA_SECRET =
  ```

- <a name="service.RECAPTCHA_SITEKEY" href="#service.RECAPTCHA_SITEKEY">`service.RECAPTCHA_SITEKEY`</a>:
  ```ini
  RECAPTCHA_SITEKEY =
  ```

- <a name="service.HCAPTCHA_SECRET" href="#service.HCAPTCHA_SECRET">`service.HCAPTCHA_SECRET`</a>:
  For hCaptcha, create an account at https://accounts.hcaptcha.com/login to get your keys:
  ```ini
  HCAPTCHA_SECRET =
  ```

- <a name="service.HCAPTCHA_SITEKEY" href="#service.HCAPTCHA_SITEKEY">`service.HCAPTCHA_SITEKEY`</a>:
  ```ini
  HCAPTCHA_SITEKEY =
  ```

- <a name="service.MCAPTCHA_URL" href="#service.MCAPTCHA_URL">`service.MCAPTCHA_URL`</a>:
  Change this to use demo.mcaptcha.org or your self-hosted mcaptcha.org instance:
  ```ini
  MCAPTCHA_URL = https://demo.mcaptcha.org
  ```

- <a name="service.MCAPTCHA_SECRET" href="#service.MCAPTCHA_SECRET">`service.MCAPTCHA_SECRET`</a>:
  Go to your configured mCaptcha instance and register a sitekey
  and use your account's secret:
  ```ini
  MCAPTCHA_SECRET =
  ```

- <a name="service.MCAPTCHA_SITEKEY" href="#service.MCAPTCHA_SITEKEY">`service.MCAPTCHA_SITEKEY`</a>:
  ```ini
  MCAPTCHA_SITEKEY =
  ```

- <a name="service.CF_TURNSTILE_SITEKEY" href="#service.CF_TURNSTILE_SITEKEY">`service.CF_TURNSTILE_SITEKEY`</a>:
  Go to https://dash.cloudflare.com/?to=/:account/turnstile to sign up for a key:
  ```ini
  CF_TURNSTILE_SITEKEY =
  ```

- <a name="service.CF_TURNSTILE_SECRET" href="#service.CF_TURNSTILE_SECRET">`service.CF_TURNSTILE_SECRET`</a>:
  ```ini
  CF_TURNSTILE_SECRET =
  ```

- <a name="service.DEFAULT_KEEP_EMAIL_PRIVATE" href="#service.DEFAULT_KEEP_EMAIL_PRIVATE">`service.DEFAULT_KEEP_EMAIL_PRIVATE`</a>:
  Default value for KeepEmailPrivate
  Each new user will get the value of this setting copied into their profile:
  ```ini
  DEFAULT_KEEP_EMAIL_PRIVATE = false
  ```

- <a name="service.DEFAULT_ALLOW_CREATE_ORGANIZATION" href="#service.DEFAULT_ALLOW_CREATE_ORGANIZATION">`service.DEFAULT_ALLOW_CREATE_ORGANIZATION`</a>:
  Default value for AllowCreateOrganization
  Every new user will have rights set to create organizations depending on this setting:
  ```ini
  DEFAULT_ALLOW_CREATE_ORGANIZATION = true
  ```

- <a name="service.DEFAULT_USER_IS_RESTRICTED" href="#service.DEFAULT_USER_IS_RESTRICTED">`service.DEFAULT_USER_IS_RESTRICTED`</a>:
  Default value for IsRestricted
  Every new user will have restricted permissions depending on this setting:
  ```ini
  DEFAULT_USER_IS_RESTRICTED = false
  ```

- <a name="service.ALLOW_DOTS_IN_USERNAMES" href="#service.ALLOW_DOTS_IN_USERNAMES">`service.ALLOW_DOTS_IN_USERNAMES`</a>:
  Users will be able to use dots when choosing their username. Disabling this is
  helpful if your usersare having issues with e.g. RSS feeds or advanced third-party
  extensions that use strange regex patterns:
  ```ini
  ALLOW_DOTS_IN_USERNAMES = true
  ```

- <a name="service.DEFAULT_USER_VISIBILITY" href="#service.DEFAULT_USER_VISIBILITY">`service.DEFAULT_USER_VISIBILITY`</a>:
  Either "public", "limited" or "private", default is "public"
  Limited is for users visible only to signed users
  Private is for users visible only to members of their organizations
  Public is for users visible for everyone:
  ```ini
  DEFAULT_USER_VISIBILITY = public
  ```

- <a name="service.ALLOWED_USER_VISIBILITY_MODES" href="#service.ALLOWED_USER_VISIBILITY_MODES">`service.ALLOWED_USER_VISIBILITY_MODES`</a>:
  Set which visibility modes a user can have:
  ```ini
  ALLOWED_USER_VISIBILITY_MODES = public,limited,private
  ```

- <a name="service.DEFAULT_ORG_VISIBILITY" href="#service.DEFAULT_ORG_VISIBILITY">`service.DEFAULT_ORG_VISIBILITY`</a>:
  Either "public", "limited" or "private", default is "public"
  Limited is for organizations visible only to signed users
  Private is for organizations visible only to members of the organization
  Public is for organizations visible to everyone:
  ```ini
  DEFAULT_ORG_VISIBILITY = public
  ```

- <a name="service.DEFAULT_ORG_MEMBER_VISIBLE" href="#service.DEFAULT_ORG_MEMBER_VISIBLE">`service.DEFAULT_ORG_MEMBER_VISIBLE`</a>:
  Default value for DefaultOrgMemberVisible
  True will make the membership of the users visible when added to the organisation:
  ```ini
  DEFAULT_ORG_MEMBER_VISIBLE = false
  ```

- <a name="service.DEFAULT_ENABLE_DEPENDENCIES" href="#service.DEFAULT_ENABLE_DEPENDENCIES">`service.DEFAULT_ENABLE_DEPENDENCIES`</a>:
  Default value for EnableDependencies
  Repositories will use dependencies by default depending on this setting:
  ```ini
  DEFAULT_ENABLE_DEPENDENCIES = true
  ```

- <a name="service.ALLOW_CROSS_REPOSITORY_DEPENDENCIES" href="#service.ALLOW_CROSS_REPOSITORY_DEPENDENCIES">`service.ALLOW_CROSS_REPOSITORY_DEPENDENCIES`</a>:
  Dependencies can be added from any repository where the user is granted access or only from the current repository depending on this setting:
  ```ini
  ALLOW_CROSS_REPOSITORY_DEPENDENCIES = true
  ```

- <a name="service.USER_LOCATION_MAP_URL" href="#service.USER_LOCATION_MAP_URL">`service.USER_LOCATION_MAP_URL`</a>:
  Default map service. No external API support has been included. A service has to allow
  searching using URL parameters, the location will be appended to the URL as escaped query parameter.
  Some example values are:
  - OpenStreetMap: https://www.openstreetmap.org/search?query=
  - Google Maps: https://www.google.com/maps/place/
  - MapQuest: https://www.mapquest.com/search/
  - Bing Maps: https://www.bing.com/maps?where1=:
  ```ini
  USER_LOCATION_MAP_URL = https://www.openstreetmap.org/search?query=
  ```

- <a name="service.ENABLE_USER_HEATMAP" href="#service.ENABLE_USER_HEATMAP">`service.ENABLE_USER_HEATMAP`</a>:
  Enable heatmap on users profiles:
  ```ini
  ENABLE_USER_HEATMAP = true
  ```

- <a name="service.ENABLE_TIMETRACKING" href="#service.ENABLE_TIMETRACKING">`service.ENABLE_TIMETRACKING`</a>:
  Enable Timetracking:
  ```ini
  ENABLE_TIMETRACKING = true
  ```

- <a name="service.DEFAULT_ENABLE_TIMETRACKING" href="#service.DEFAULT_ENABLE_TIMETRACKING">`service.DEFAULT_ENABLE_TIMETRACKING`</a>:
  Default value for EnableTimetracking
  Repositories will use timetracking by default depending on this setting:
  ```ini
  DEFAULT_ENABLE_TIMETRACKING = true
  ```

- <a name="service.DEFAULT_ALLOW_ONLY_CONTRIBUTORS_TO_TRACK_TIME" href="#service.DEFAULT_ALLOW_ONLY_CONTRIBUTORS_TO_TRACK_TIME">`service.DEFAULT_ALLOW_ONLY_CONTRIBUTORS_TO_TRACK_TIME`</a>:
  Default value for AllowOnlyContributorsToTrackTime
  Only users with write permissions can track time if this is true:
  ```ini
  DEFAULT_ALLOW_ONLY_CONTRIBUTORS_TO_TRACK_TIME = true
  ```

- <a name="service.NO_REPLY_ADDRESS" href="#service.NO_REPLY_ADDRESS">`service.NO_REPLY_ADDRESS`</a>:
  Value for the domain part of the user's email address in the git log if user
  has set KeepEmailPrivate to true. The user's email will be replaced with a
  concatenation of the user name in lower case, "@" and <a href="#service.NO_REPLY_ADDRESS">`NO_REPLY_ADDRESS`</a>.:
  ```ini
  NO_REPLY_ADDRESS = noreply.%(server.DOMAIN)s
  ```

- <a name="service.SHOW_REGISTRATION_BUTTON" href="#service.SHOW_REGISTRATION_BUTTON">`service.SHOW_REGISTRATION_BUTTON`</a>:
  Show Registration button:
  ```ini
  SHOW_REGISTRATION_BUTTON = true
  ```

- <a name="service.ENABLE_INTERNAL_SIGNIN" href="#service.ENABLE_INTERNAL_SIGNIN">`service.ENABLE_INTERNAL_SIGNIN`</a>:
  Whether to allow internal signin:
  ```ini
  ENABLE_INTERNAL_SIGNIN = true
  ```

- <a name="service.SHOW_MILESTONES_DASHBOARD_PAGE" href="#service.SHOW_MILESTONES_DASHBOARD_PAGE">`service.SHOW_MILESTONES_DASHBOARD_PAGE`</a>:
  Show milestones dashboard page - a view of all the user's milestones:
  ```ini
  SHOW_MILESTONES_DASHBOARD_PAGE = true
  ```

- <a name="service.AUTO_WATCH_NEW_REPOS" href="#service.AUTO_WATCH_NEW_REPOS">`service.AUTO_WATCH_NEW_REPOS`</a>:
  Default value for AutoWatchNewRepos
  When adding a repo to a team or creating a new repo all team members will watch the
  repo automatically if enabled:
  ```ini
  AUTO_WATCH_NEW_REPOS = true
  ```

- <a name="service.AUTO_WATCH_ON_CHANGES" href="#service.AUTO_WATCH_ON_CHANGES">`service.AUTO_WATCH_ON_CHANGES`</a>:
  Default value for AutoWatchOnChanges
  Make the user watch a repository When they commit for the first time:
  ```ini
  AUTO_WATCH_ON_CHANGES = false
  ```

- <a name="service.USER_DELETE_WITH_COMMENTS_MAX_TIME" href="#service.USER_DELETE_WITH_COMMENTS_MAX_TIME">`service.USER_DELETE_WITH_COMMENTS_MAX_TIME`</a>:
  Minimum amount of time a user must exist before comments are kept when the user is deleted:
  ```ini
  USER_DELETE_WITH_COMMENTS_MAX_TIME = 0
  ```

- <a name="service.VALID_SITE_URL_SCHEMES" href="#service.VALID_SITE_URL_SCHEMES">`service.VALID_SITE_URL_SCHEMES`</a>:
  Valid site url schemes for user profiles:
  ```ini
  VALID_SITE_URL_SCHEMES = http,https
  ```

### <a name="service.explore" href="#service.explore">Service explore</a>

```ini
[service.explore]
```

- <a name="service.explore.REQUIRE_SIGNIN_VIEW" href="#service.explore.REQUIRE_SIGNIN_VIEW">`service.explore.REQUIRE_SIGNIN_VIEW`</a>:
  Only allow signed in users to view the explore pages:
  ```ini
  REQUIRE_SIGNIN_VIEW = false
  ```

- <a name="service.explore.DISABLE_USERS_PAGE" href="#service.explore.DISABLE_USERS_PAGE">`service.explore.DISABLE_USERS_PAGE`</a>:
  Disable the users explore page:
  ```ini
  DISABLE_USERS_PAGE = false
  ```

- <a name="service.explore.DISABLE_ORGANIZATIONS_PAGE" href="#service.explore.DISABLE_ORGANIZATIONS_PAGE">`service.explore.DISABLE_ORGANIZATIONS_PAGE`</a>:
  Disable the organizations explore page:
  ```ini
  DISABLE_ORGANIZATIONS_PAGE = false
  ```

- <a name="service.explore.DISABLE_CODE_PAGE" href="#service.explore.DISABLE_CODE_PAGE">`service.explore.DISABLE_CODE_PAGE`</a>:
  Disable the code explore page:
  ```ini
  DISABLE_CODE_PAGE = false
  ```

## <a name="badges" href="#badges">Badges</a>

```ini
[badges]
```

- <a name="badges.ENABLED" href="#badges.ENABLED">`badges.ENABLED`</a>:
  Enable repository badges (via shields.io or a similar generator):
  ```ini
  ENABLED = true
  ```

- <a name="badges.GENERATOR_URL_TEMPLATE" href="#badges.GENERATOR_URL_TEMPLATE">`badges.GENERATOR_URL_TEMPLATE`</a>:
  Template for the badge generator:
  ```ini
  GENERATOR_URL_TEMPLATE = https://img.shields.io/badge/{{.label}}-{{.text}}-{{.color}}
  ```

## <a name="repository" href="#repository">Repository</a>

```ini
[repository]
```

- <a name="repository.ROOT" href="#repository.ROOT">`repository.ROOT`</a>:
  Root path for storing all repository data. By default, it is set to `%(APP_DATA_PATH)s/gitea-repositories`.
  Relative paths will be made absolute against the _<a href="#AppWorkPath">`AppWorkPath`</a>_.:
  ```ini
  ROOT =
  ```

- <a name="repository.SCRIPT_TYPE" href="#repository.SCRIPT_TYPE">`repository.SCRIPT_TYPE`</a>:
  The script type this server supports. Usually this is `bash`, but some users report that only `sh` is available.:
  ```ini
  SCRIPT_TYPE = bash
  ```

- <a name="repository.DETECTED_CHARSETS_ORDER" href="#repository.DETECTED_CHARSETS_ORDER">`repository.DETECTED_CHARSETS_ORDER`</a>:
  Tie-break order for detected charsets.
  If the charsets have equal confidence, tie-breaking will be done by order in this list
  with charsets earlier in the list chosen in preference to those later.
  Adding "defaults" will place the unused charsets at that position.:
  ```ini
  DETECTED_CHARSETS_ORDER = UTF-8, UTF-16BE, UTF-16LE, UTF-32BE, UTF-32LE, ISO-8859, windows-1252, ISO-8859, windows-1250, ISO-8859, ISO-8859, ISO-8859, windows-1253, ISO-8859, windows-1255, ISO-8859, windows-1251, windows-1256, KOI8-R, ISO-8859, windows-1254, Shift_JIS, GB18030, EUC-JP, EUC-KR, Big5, ISO-2022, ISO-2022, ISO-2022, IBM424_rtl, IBM424_ltr, IBM420_rtl, IBM420_ltr
  ```

- <a name="repository.ANSI_CHARSET" href="#repository.ANSI_CHARSET">`repository.ANSI_CHARSET`</a>:
  Default ANSI charset to override non-UTF-8 charsets to:
  ```ini
  ANSI_CHARSET =
  ```

- <a name="repository.FORCE_PRIVATE" href="#repository.FORCE_PRIVATE">`repository.FORCE_PRIVATE`</a>:
  Force every new repository to be private:
  ```ini
  FORCE_PRIVATE = false
  ```

- <a name="repository.DEFAULT_PRIVATE" href="#repository.DEFAULT_PRIVATE">`repository.DEFAULT_PRIVATE`</a>:
  Default privacy setting when creating a new repository, allowed values: last, private, public. Default is last which means the last setting used:
  ```ini
  DEFAULT_PRIVATE = last
  ```

- <a name="repository.DEFAULT_PUSH_CREATE_PRIVATE" href="#repository.DEFAULT_PUSH_CREATE_PRIVATE">`repository.DEFAULT_PUSH_CREATE_PRIVATE`</a>:
  Default private when using push-to-create:
  ```ini
  DEFAULT_PUSH_CREATE_PRIVATE = true
  ```

- <a name="repository.MAX_CREATION_LIMIT" href="#repository.MAX_CREATION_LIMIT">`repository.MAX_CREATION_LIMIT`</a>:
  Global limit of repositories per user, applied at creation time. -1 means no limit:
  ```ini
  MAX_CREATION_LIMIT = -1
  ```

- <a name="repository.PREFERRED_LICENSES" href="#repository.PREFERRED_LICENSES">`repository.PREFERRED_LICENSES`</a>:
  Preferred Licenses to place at the top of the List
  The name here must match the filename in options/license or custom/options/license:
  ```ini
  PREFERRED_LICENSES = Apache-2.0,MIT
  ```

- <a name="repository.DISABLE_HTTP_GIT" href="#repository.DISABLE_HTTP_GIT">`repository.DISABLE_HTTP_GIT`</a>:
  Disable the ability to interact with repositories using the HTTP protocol:
  ```ini
  DISABLE_HTTP_GIT = false
  ```

- <a name="repository.ACCESS_CONTROL_ALLOW_ORIGIN" href="#repository.ACCESS_CONTROL_ALLOW_ORIGIN">`repository.ACCESS_CONTROL_ALLOW_ORIGIN`</a>:
  Value for Access-Control-Allow-Origin header, default is not to present.
  WARNING: This may be harmful to your website if you do not give it a right value.:
  ```ini
  ACCESS_CONTROL_ALLOW_ORIGIN =
  ```

- <a name="repository.USE_COMPAT_SSH_URI" href="#repository.USE_COMPAT_SSH_URI">`repository.USE_COMPAT_SSH_URI`</a>:
  Force ssh:// clone url instead of scp-style uri when default SSH port is used:
  ```ini
  USE_COMPAT_SSH_URI = true
  ```

- <a name="repository.GO_GET_CLONE_URL_PROTOCOL" href="#repository.GO_GET_CLONE_URL_PROTOCOL">`repository.GO_GET_CLONE_URL_PROTOCOL`</a>:
  Value for the "go get" request returns the repository url as https or ssh, default is https:
  ```ini
  GO_GET_CLONE_URL_PROTOCOL = https
  ```

- <a name="repository.DEFAULT_CLOSE_ISSUES_VIA_COMMITS_IN_ANY_BRANCH" href="#repository.DEFAULT_CLOSE_ISSUES_VIA_COMMITS_IN_ANY_BRANCH">`repository.DEFAULT_CLOSE_ISSUES_VIA_COMMITS_IN_ANY_BRANCH`</a>:
  Close issues as long as a commit on any branch marks it as fixed:
  ```ini
  DEFAULT_CLOSE_ISSUES_VIA_COMMITS_IN_ANY_BRANCH = false
  ```

- <a name="repository.ENABLE_PUSH_CREATE_USER" href="#repository.ENABLE_PUSH_CREATE_USER">`repository.ENABLE_PUSH_CREATE_USER`</a>:
  Allow users to push local repositories to Forgejo and have them automatically created for a user or an org:
  ```ini
  ENABLE_PUSH_CREATE_USER = false
  ```

- <a name="repository.ENABLE_PUSH_CREATE_ORG" href="#repository.ENABLE_PUSH_CREATE_ORG">`repository.ENABLE_PUSH_CREATE_ORG`</a>:
  ```ini
  ENABLE_PUSH_CREATE_ORG = false
  ```

- <a name="repository.DISABLED_REPO_UNITS" href="#repository.DISABLED_REPO_UNITS">`repository.DISABLED_REPO_UNITS`</a>:
  Comma separated list of globally disabled repo units. Allowed values: repo.issues, repo.ext_issues, repo.pulls, repo.wiki, repo.ext_wiki, repo.projects, repo.packages, repo.actions.:
  ```ini
  DISABLED_REPO_UNITS =
  ```

- <a name="repository.DEFAULT_REPO_UNITS" href="#repository.DEFAULT_REPO_UNITS">`repository.DEFAULT_REPO_UNITS`</a>:
  Comma separated list of default new repo units. Allowed values: repo.code, repo.releases, repo.issues, repo.pulls, repo.wiki, repo.projects, repo.packages, repo.actions.
  Note: Code and Releases can currently not be deactivated. If you specify default repo units you should still list them for future compatibility.
  External wiki and issue tracker can't be enabled by default as it requires additional settings.
  Disabled repo units will not be added to new repositories regardless if it is in the default list.:
  ```ini
  DEFAULT_REPO_UNITS = repo.code,repo.releases,repo.issues,repo.pulls,repo.wiki,repo.projects,repo.packages,repo.actions
  ```

- <a name="repository.DEFAULT_FORK_REPO_UNITS" href="#repository.DEFAULT_FORK_REPO_UNITS">`repository.DEFAULT_FORK_REPO_UNITS`</a>:
  Comma separated list of default forked repo units.
  The set of allowed values and rules are the same as <a href="#repository.DEFAULT_REPO_UNITS">`DEFAULT_REPO_UNITS`</a>:
  ```ini
  DEFAULT_FORK_REPO_UNITS = repo.code,repo.pulls
  ```

- <a name="repository.PREFIX_ARCHIVE_FILES" href="#repository.PREFIX_ARCHIVE_FILES">`repository.PREFIX_ARCHIVE_FILES`</a>:
  Prefix archive files by placing them in a directory named after the repository:
  ```ini
  PREFIX_ARCHIVE_FILES = true
  ```

- <a name="repository.DISABLE_MIGRATIONS" href="#repository.DISABLE_MIGRATIONS">`repository.DISABLE_MIGRATIONS`</a>:
  Disable migrating feature:
  ```ini
  DISABLE_MIGRATIONS = false
  ```

- <a name="repository.DISABLE_STARS" href="#repository.DISABLE_STARS">`repository.DISABLE_STARS`</a>:
  Disable stars feature:
  ```ini
  DISABLE_STARS = false
  ```

- <a name="repository.DISABLE_FORKS" href="#repository.DISABLE_FORKS">`repository.DISABLE_FORKS`</a>:
  Disable repository forking:
  ```ini
  DISABLE_FORKS = false
  ```

- <a name="repository.DEFAULT_BRANCH" href="#repository.DEFAULT_BRANCH">`repository.DEFAULT_BRANCH`</a>:
  The default branch name of new repositories:
  ```ini
  DEFAULT_BRANCH = main
  ```

- <a name="repository.ALLOW_ADOPTION_OF_UNADOPTED_REPOSITORIES" href="#repository.ALLOW_ADOPTION_OF_UNADOPTED_REPOSITORIES">`repository.ALLOW_ADOPTION_OF_UNADOPTED_REPOSITORIES`</a>:
  Allow adoption of unadopted repositories:
  ```ini
  ALLOW_ADOPTION_OF_UNADOPTED_REPOSITORIES = false
  ```

- <a name="repository.ALLOW_DELETION_OF_UNADOPTED_REPOSITORIES" href="#repository.ALLOW_DELETION_OF_UNADOPTED_REPOSITORIES">`repository.ALLOW_DELETION_OF_UNADOPTED_REPOSITORIES`</a>:
  Allow deletion of unadopted repositories:
  ```ini
  ALLOW_DELETION_OF_UNADOPTED_REPOSITORIES = false
  ```

- <a name="repository.DISABLE_DOWNLOAD_SOURCE_ARCHIVES" href="#repository.DISABLE_DOWNLOAD_SOURCE_ARCHIVES">`repository.DISABLE_DOWNLOAD_SOURCE_ARCHIVES`</a>:
  Don't allow download source archive files from UI:
  ```ini
  DISABLE_DOWNLOAD_SOURCE_ARCHIVES = false
  ```

- <a name="repository.ALLOW_FORK_WITHOUT_MAXIMUM_LIMIT" href="#repository.ALLOW_FORK_WITHOUT_MAXIMUM_LIMIT">`repository.ALLOW_FORK_WITHOUT_MAXIMUM_LIMIT`</a>:
  Allow fork repositories without maximum number limit:
  ```ini
  ALLOW_FORK_WITHOUT_MAXIMUM_LIMIT = true
  ```

- <a name="repository.editor.LINE_WRAP_EXTENSIONS" href="#repository.editor.LINE_WRAP_EXTENSIONS">`repository.editor.LINE_WRAP_EXTENSIONS`</a>:
  List of file extensions for which lines should be wrapped in the Monaco editor
  Separate extensions with a comma. To line wrap files without an extension, just put a comma:
  ```ini
  LINE_WRAP_EXTENSIONS = .txt,.md,.markdown,.mdown,.mkd,.livemd,
  ```

- <a name="repository.local.LOCAL_COPY_PATH" href="#repository.local.LOCAL_COPY_PATH">`repository.local.LOCAL_COPY_PATH`</a>:
  Path for local repository copy. Defaults to `tmp/local-repo` (content gets deleted on Forgejo restart):
  ```ini
  LOCAL_COPY_PATH = tmp/local-repo
  ```

### <a name="repository.upload" href="#repository.upload">Repository upload</a>

```ini
[repository.upload]
```

- <a name="repository.upload.ENABLED" href="#repository.upload.ENABLED">`repository.upload.ENABLED`</a>:
  Whether repository file uploads are enabled. Defaults to `true`:
  ```ini
  ENABLED = true
  ```

- <a name="repository.upload.TEMP_PATH" href="#repository.upload.TEMP_PATH">`repository.upload.TEMP_PATH`</a>:
  Path for uploads. Defaults to `data/tmp/uploads` (content gets deleted on gitea restart):
  ```ini
  TEMP_PATH = data/tmp/uploads
  ```

- <a name="repository.upload.ALLOWED_TYPES" href="#repository.upload.ALLOWED_TYPES">`repository.upload.ALLOWED_TYPES`</a>:
  Comma-separated list of allowed file extensions (`.zip`), mime types (`text/plain`) or wildcard type (`image/*`, `audio/*`, `video/*`). Empty value or `*/*` allows all types.:
  ```ini
  ALLOWED_TYPES =
  ```

- <a name="repository.upload.FILE_MAX_SIZE" href="#repository.upload.FILE_MAX_SIZE">`repository.upload.FILE_MAX_SIZE`</a>:
  Max size of each file in megabytes. Defaults to 50MB:
  ```ini
  FILE_MAX_SIZE = 50
  ```

- <a name="repository.upload.MAX_FILES" href="#repository.upload.MAX_FILES">`repository.upload.MAX_FILES`</a>:
  Max number of files per upload. Defaults to 5:
  ```ini
  MAX_FILES = 5
  ```

### <a name="repository.pull-request" href="#repository.pull-request">Repository pull request</a>

```ini
[repository.pull-request]
```

- <a name="repository.pull-request.WORK_IN_PROGRESS_PREFIXES" href="#repository.pull-request.WORK_IN_PROGRESS_PREFIXES">`repository.pull-request.WORK_IN_PROGRESS_PREFIXES`</a>:
  List of prefixes used in Pull Request title to mark them as Work In Progress (matched in a case-insensitive manner):
  ```ini
  WORK_IN_PROGRESS_PREFIXES = WIP:,[WIP]
  ```

- <a name="repository.pull-request.CLOSE_KEYWORDS" href="#repository.pull-request.CLOSE_KEYWORDS">`repository.pull-request.CLOSE_KEYWORDS`</a>:
  List of keywords used in Pull Request comments to automatically close a related issue:
  ```ini
  CLOSE_KEYWORDS = close,closes,closed,fix,fixes,fixed,resolve,resolves,resolved
  ```

- <a name="repository.pull-request.REOPEN_KEYWORDS" href="#repository.pull-request.REOPEN_KEYWORDS">`repository.pull-request.REOPEN_KEYWORDS`</a>:
  List of keywords used in Pull Request comments to automatically reopen a related issue:
  ```ini
  REOPEN_KEYWORDS = reopen,reopens,reopened
  ```

- <a name="repository.pull-request.DEFAULT_MERGE_STYLE" href="#repository.pull-request.DEFAULT_MERGE_STYLE">`repository.pull-request.DEFAULT_MERGE_STYLE`</a>:
  Set default merge style for repository creating, valid options: merge, rebase, rebase-merge, squash, fast-forward-only:
  ```ini
  DEFAULT_MERGE_STYLE = merge
  ```

- <a name="repository.pull-request.DEFAULT_MERGE_MESSAGE_COMMITS_LIMIT" href="#repository.pull-request.DEFAULT_MERGE_MESSAGE_COMMITS_LIMIT">`repository.pull-request.DEFAULT_MERGE_MESSAGE_COMMITS_LIMIT`</a>:
  In the default merge message for squash commits include at most this many commits:
  ```ini
  DEFAULT_MERGE_MESSAGE_COMMITS_LIMIT = 50
  ```

- <a name="repository.pull-request.DEFAULT_MERGE_MESSAGE_SIZE" href="#repository.pull-request.DEFAULT_MERGE_MESSAGE_SIZE">`repository.pull-request.DEFAULT_MERGE_MESSAGE_SIZE`</a>:
  In the default merge message for squash commits limit the size of the commit messages to this:
  ```ini
  DEFAULT_MERGE_MESSAGE_SIZE = 5120
  ```

- <a name="repository.pull-request.DEFAULT_MERGE_MESSAGE_ALL_AUTHORS" href="#repository.pull-request.DEFAULT_MERGE_MESSAGE_ALL_AUTHORS">`repository.pull-request.DEFAULT_MERGE_MESSAGE_ALL_AUTHORS`</a>:
  In the default merge message for squash commits walk all commits to include all authors in the Co-authored-by otherwise just use those in the limited list:
  ```ini
  DEFAULT_MERGE_MESSAGE_ALL_AUTHORS = false
  ```

- <a name="repository.pull-request.DEFAULT_MERGE_MESSAGE_MAX_APPROVERS" href="#repository.pull-request.DEFAULT_MERGE_MESSAGE_MAX_APPROVERS">`repository.pull-request.DEFAULT_MERGE_MESSAGE_MAX_APPROVERS`</a>:
  In default merge messages limit the number of approvers listed as Reviewed-by: to this many:
  ```ini
  DEFAULT_MERGE_MESSAGE_MAX_APPROVERS = 10
  ```

- <a name="repository.pull-request.DEFAULT_MERGE_MESSAGE_OFFICIAL_APPROVERS_ONLY" href="#repository.pull-request.DEFAULT_MERGE_MESSAGE_OFFICIAL_APPROVERS_ONLY">`repository.pull-request.DEFAULT_MERGE_MESSAGE_OFFICIAL_APPROVERS_ONLY`</a>:
  In default merge messages only include approvers who are official:
  ```ini
  DEFAULT_MERGE_MESSAGE_OFFICIAL_APPROVERS_ONLY = true
  ```

- <a name="repository.pull-request.POPULATE_SQUASH_COMMENT_WITH_COMMIT_MESSAGES" href="#repository.pull-request.POPULATE_SQUASH_COMMENT_WITH_COMMIT_MESSAGES">`repository.pull-request.POPULATE_SQUASH_COMMENT_WITH_COMMIT_MESSAGES`</a>:
  If an squash commit's comment should be populated with the commit messages of the squashed commits:
  ```ini
  POPULATE_SQUASH_COMMENT_WITH_COMMIT_MESSAGES = false
  ```

- <a name="repository.pull-request.ADD_CO_COMMITTER_TRAILERS" href="#repository.pull-request.ADD_CO_COMMITTER_TRAILERS">`repository.pull-request.ADD_CO_COMMITTER_TRAILERS`</a>:
  Add co-authored-by and co-committed-by trailers if committer does not match author:
  ```ini
  ADD_CO_COMMITTER_TRAILERS = true
  ```

- <a name="repository.pull-request.TEST_CONFLICTING_PATCHES_WITH_GIT_APPLY" href="#repository.pull-request.TEST_CONFLICTING_PATCHES_WITH_GIT_APPLY">`repository.pull-request.TEST_CONFLICTING_PATCHES_WITH_GIT_APPLY`</a>:
  In addition to testing patches using the three-way merge method, re-test conflicting patches with git apply:
  ```ini
  TEST_CONFLICTING_PATCHES_WITH_GIT_APPLY = false
  ```

- <a name="repository.pull-request.RETARGET_CHILDREN_ON_MERGE" href="#repository.pull-request.RETARGET_CHILDREN_ON_MERGE">`repository.pull-request.RETARGET_CHILDREN_ON_MERGE`</a>:
  Retarget child pull requests to the parent pull request branch target on merge of parent pull request. It only works on merged PRs where the head and base branch target the same repo.:
  ```ini
  RETARGET_CHILDREN_ON_MERGE = true
  ```

### <a name="repository.issue" href="#repository.issue">Repository issue</a>

```ini
[repository.issue]
```

- <a name="repository.issue.LOCK_REASONS" href="#repository.issue.LOCK_REASONS">`repository.issue.LOCK_REASONS`</a>:
  List of reasons why a Pull Request or Issue can be locked:
  ```ini
  LOCK_REASONS = Too heated,Off-topic,Spam,Resolved
  ```

- <a name="repository.issue.MAX_PINNED" href="#repository.issue.MAX_PINNED">`repository.issue.MAX_PINNED`</a>:
  Maximum number of pinned Issues per repo
  Set to 0 to disable pinning Issues:
  ```ini
  MAX_PINNED = 3
  ```

### <a name="repository.release" href="#repository.release">Repository release</a>

```ini
[repository.release]
```

- <a name="repository.release.ALLOWED_TYPES" href="#repository.release.ALLOWED_TYPES">`repository.release.ALLOWED_TYPES`</a>:
  Comma-separated list of allowed file extensions (`.zip`), mime types (`text/plain`) or wildcard type (`image/*`, `audio/*`, `video/*`). Empty value or `*/*` allows all types.:
  ```ini
  ALLOWED_TYPES =
  ```

- <a name="repository.release.DEFAULT_PAGING_NUM" href="#repository.release.DEFAULT_PAGING_NUM">`repository.release.DEFAULT_PAGING_NUM`</a>:
  ```ini
  DEFAULT_PAGING_NUM = 10
  ```

### <a name="repository.signing" href="#repository.signing">Repository signing</a>

```ini
[repository.signing]
```

- <a name="repository.signing.SIGNING_KEY" href="#repository.signing.SIGNING_KEY">`repository.signing.SIGNING_KEY`</a>:
  GPG key to use to sign commits, Defaults to the default - that is the value of `git config --get user.signingkey`
  Run in the context of the <a href="#RUN_USER">`RUN_USER`</a>.
  Switch to none to stop signing completely.:
  ```ini
  SIGNING_KEY = default
  ```

- <a name="repository.signing.SIGNING_NAME" href="#repository.signing.SIGNING_NAME">`repository.signing.SIGNING_NAME`</a>:
  If a SIGNING_KEY ID is provided and is not set to default, use the provided Name and Email address as the signer.
  These should match a publicized name and email address for the key
  (When SIGNING_KEY is default these are set to
  the results of `git config --get user.name` and `git config --get user.email`, respectively and can only be overridden
  by setting the SIGNING_KEY ID to the correct ID.).:
  ```ini
  SIGNING_NAME =
  ```

- <a name="repository.signing.SIGNING_EMAIL" href="#repository.signing.SIGNING_EMAIL">`repository.signing.SIGNING_EMAIL`</a>:
  ```ini
  SIGNING_EMAIL =
  ```

- <a name="repository.signing.DEFAULT_TRUST_MODEL" href="#repository.signing.DEFAULT_TRUST_MODEL">`repository.signing.DEFAULT_TRUST_MODEL`</a>:
  Sets the default trust model for repositories. Options are: collaborator, committer, collaboratorcommitter.:
  ```ini
  DEFAULT_TRUST_MODEL = collaborator
  ```

- <a name="repository.signing.INITIAL_COMMIT" href="#repository.signing.INITIAL_COMMIT">`repository.signing.INITIAL_COMMIT`</a>:
  Determines when Forgejo should sign the initial commit when creating a repository
  Either:
  - never
  - pubkey: only sign if the user has a pubkey
  - twofa: only sign if the user has logged in with twofa
  - always
  options other than none and always can be combined as comma separated list.:
  ```ini
  INITIAL_COMMIT = always
  ```

- <a name="repository.signing.CRUD_ACTIONS" href="#repository.signing.CRUD_ACTIONS">`repository.signing.CRUD_ACTIONS`</a>:
  Determines when to sign for CRUD actions
  - as above for <a href="#repository.signing.INITIAL_COMMIT">`INITIAL_COMMIT`</a> or
  - parentsigned: requires that the parent commit is signed.:
  ```ini
  CRUD_ACTIONS = pubkey, twofa, parentsigned
  ```

- <a name="repository.signing.WIKI" href="#repository.signing.WIKI">`repository.signing.WIKI`</a>:
  Determines when to sign Wiki commits
  - as above for <a href="#repository.signing.INITIAL_COMMIT">`INITIAL_COMMIT`</a>.:
  ```ini
  WIKI = never
  ```

- <a name="repository.signing.MERGES" href="#repository.signing.MERGES">`repository.signing.MERGES`</a>:
  Determines when to sign on merges:
  - as above for <a href="#repository.signing.INITIAL_COMMIT">`INITIAL_COMMIT`</a> or
  - `basesigned`: require that the parent of commit on the base repo is signed,
  - `commitssigned`: require that all the commits in the head branch are signed,
  - `approved`: only sign when merging an approved pr to a protected branch.:
  ```ini
  MERGES = pubkey, twofa, basesigned, commitssigned
  ```

- <a name="repository.mimetype_mapping" href="#repository.mimetype_mapping">`repository.mimetype_mapping`</a>:
  ```ini
  mimetype_mapping =
  ```

## <a name="project" href="#project">Project</a>

```ini
[project]
```

- <a name="project.PROJECT_BOARD_BASIC_KANBAN_TYPE" href="#project.PROJECT_BOARD_BASIC_KANBAN_TYPE">`project.PROJECT_BOARD_BASIC_KANBAN_TYPE`</a>:
  Default templates for project boards:
  ```ini
  PROJECT_BOARD_BASIC_KANBAN_TYPE = To Do, In Progress, Done
  ```

- <a name="project.PROJECT_BOARD_BUG_TRIAGE_TYPE" href="#project.PROJECT_BOARD_BUG_TRIAGE_TYPE">`project.PROJECT_BOARD_BUG_TRIAGE_TYPE`</a>:
  ```ini
  PROJECT_BOARD_BUG_TRIAGE_TYPE = Needs Triage, High Priority, Low Priority, Closed
  ```

## <a name="cors" href="#cors">CORS</a>

```ini
[cors]
```

- <a name="cors.ENABLED" href="#cors.ENABLED">`cors.ENABLED`</a>:
  More information about CORS can be found here: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#The_HTTP_response_headers
  enable cors headers (disabled by default):
  ```ini
  ENABLED = false
  ```

- <a name="cors.ALLOW_DOMAIN" href="#cors.ALLOW_DOMAIN">`cors.ALLOW_DOMAIN`</a>:
  list of requesting origins that are allowed, eg: "https://*.example.com":
  ```ini
  ALLOW_DOMAIN = *
  ```

- <a name="cors.METHODS" href="#cors.METHODS">`cors.METHODS`</a>:
  list of methods allowed to request:
  ```ini
  METHODS = GET,HEAD,POST,PUT,PATCH,DELETE,OPTIONS
  ```

- <a name="cors.MAX_AGE" href="#cors.MAX_AGE">`cors.MAX_AGE`</a>:
  max time to cache response:
  ```ini
  MAX_AGE = 10m
  ```

- <a name="cors.ALLOW_CREDENTIALS" href="#cors.ALLOW_CREDENTIALS">`cors.ALLOW_CREDENTIALS`</a>:
  allow request with credentials:
  ```ini
  ALLOW_CREDENTIALS = false
  ```

- <a name="cors.HEADERS" href="#cors.HEADERS">`cors.HEADERS`</a>:
  headers to permit:
  ```ini
  HEADERS = Content-Type,User-Agent
  ```

- <a name="cors.X_FRAME_OPTIONS" href="#cors.X_FRAME_OPTIONS">`cors.X_FRAME_OPTIONS`</a>:
  set X-FRAME-OPTIONS header:
  ```ini
  X_FRAME_OPTIONS = SAMEORIGIN
  ```

## <a name="ui" href="#ui">UI</a>

```ini
[ui]
```

- <a name="ui.EXPLORE_PAGING_NUM" href="#ui.EXPLORE_PAGING_NUM">`ui.EXPLORE_PAGING_NUM`</a>:
  Number of repositories that are displayed on one explore page:
  ```ini
  EXPLORE_PAGING_NUM = 20
  ```

- <a name="ui.ISSUE_PAGING_NUM" href="#ui.ISSUE_PAGING_NUM">`ui.ISSUE_PAGING_NUM`</a>:
  Number of issues that are displayed on one page:
  ```ini
  ISSUE_PAGING_NUM = 20
  ```

- <a name="ui.REPO_SEARCH_PAGING_NUM" href="#ui.REPO_SEARCH_PAGING_NUM">`ui.REPO_SEARCH_PAGING_NUM`</a>:
  Number of repositories that are displayed on one page when searching:
  ```ini
  REPO_SEARCH_PAGING_NUM = 20
  ```

- <a name="ui.MEMBERS_PAGING_NUM" href="#ui.MEMBERS_PAGING_NUM">`ui.MEMBERS_PAGING_NUM`</a>:
  Number of members that are displayed on one page:
  ```ini
  MEMBERS_PAGING_NUM = 20
  ```

- <a name="ui.FEED_MAX_COMMIT_NUM" href="#ui.FEED_MAX_COMMIT_NUM">`ui.FEED_MAX_COMMIT_NUM`</a>:
  Number of maximum commits displayed in one activity feed:
  ```ini
  FEED_MAX_COMMIT_NUM = 5
  ```

- <a name="ui.FEED_PAGING_NUM" href="#ui.FEED_PAGING_NUM">`ui.FEED_PAGING_NUM`</a>:
  Number of items that are displayed in home feed:
  ```ini
  FEED_PAGING_NUM = 20
  ```

- <a name="ui.SITEMAP_PAGING_NUM" href="#ui.SITEMAP_PAGING_NUM">`ui.SITEMAP_PAGING_NUM`</a>:
  Number of items that are displayed in a single subsitemap:
  ```ini
  SITEMAP_PAGING_NUM = 20
  ```

- <a name="ui.PACKAGES_PAGING_NUM" href="#ui.PACKAGES_PAGING_NUM">`ui.PACKAGES_PAGING_NUM`</a>:
  Number of packages that are displayed on one page:
  ```ini
  PACKAGES_PAGING_NUM = 20
  ```

- <a name="ui.GRAPH_MAX_COMMIT_NUM" href="#ui.GRAPH_MAX_COMMIT_NUM">`ui.GRAPH_MAX_COMMIT_NUM`</a>:
  Number of maximum commits displayed in commit graph:
  ```ini
  GRAPH_MAX_COMMIT_NUM = 100
  ```

- <a name="ui.CODE_COMMENT_LINES" href="#ui.CODE_COMMENT_LINES">`ui.CODE_COMMENT_LINES`</a>:
  Number of line of codes shown for a code comment:
  ```ini
  CODE_COMMENT_LINES = 4
  ```

- <a name="ui.MAX_DISPLAY_FILE_SIZE" href="#ui.MAX_DISPLAY_FILE_SIZE">`ui.MAX_DISPLAY_FILE_SIZE`</a>:
  Max size of files to be displayed (default is 8MiB):
  ```ini
  MAX_DISPLAY_FILE_SIZE = 8388608
  ```

- <a name="ui.AMBIGUOUS_UNICODE_DETECTION" href="#ui.AMBIGUOUS_UNICODE_DETECTION">`ui.AMBIGUOUS_UNICODE_DETECTION`</a>:
  Detect ambiguous unicode characters in file contents and show warnings on the UI:
  ```ini
  AMBIGUOUS_UNICODE_DETECTION = true
  ```

- <a name="ui.SHOW_USER_EMAIL" href="#ui.SHOW_USER_EMAIL">`ui.SHOW_USER_EMAIL`</a>:
  Whether the email of the user should be shown in the Explore Users page:
  ```ini
  SHOW_USER_EMAIL = true
  ```

- <a name="ui.DEFAULT_THEME" href="#ui.DEFAULT_THEME">`ui.DEFAULT_THEME`</a>:
  Set the default theme for the Forgejo install:
  ```ini
  DEFAULT_THEME = forgejo-auto
  ```

- <a name="ui.THEMES" href="#ui.THEMES">`ui.THEMES`</a>:
  All available themes. Allow users select personalized themes regardless of the value of <a href="#ui.DEFAULT_THEME">`DEFAULT_THEME`</a>.
  By default available:
  - forgejo-auto, forgejo-light, forgejo-dark
  - gitea-auto, gitea-light, gitea-dark
  - forgejo-auto-deuteranopia-protanopia, forgejo-light-deuteranopia-protanopia, forgejo-dark-deuteranopia-protanopia
  - forgejo-auto-tritanopia, forgejo-light-tritanopia, forgejo-dark-tritanopia:
  ```ini
  THEMES = gitea-auto,gitea-light,gitea-dark
  ```

- <a name="ui.REACTIONS" href="#ui.REACTIONS">`ui.REACTIONS`</a>:
  All available reactions users can choose on issues/prs and comments.
  Values can be emoji alias (:smile:) or a unicode emoji.
  For custom reactions, add a tightly cropped square image to public/assets/img/emoji/reaction_name.png:
  ```ini
  REACTIONS = +1, -1, laugh, hooray, confused, heart, rocket, eyes
  ```

- <a name="ui.REACTION_MAX_USER_NUM" href="#ui.REACTION_MAX_USER_NUM">`ui.REACTION_MAX_USER_NUM`</a>:
  Change the number of users that are displayed in reactions tooltip (triggered by mouse hover):
  ```ini
  REACTION_MAX_USER_NUM = 10
  ```

- <a name="ui.CUSTOM_EMOJIS" href="#ui.CUSTOM_EMOJIS">`ui.CUSTOM_EMOJIS`</a>:
  Additional Emojis not defined in the utf8 standard
  By default we support gitea (:gitea:), to add more copy them to public/assets/img/emoji/emoji_name.png and add it to this config.
  Dont mistake it for Reactions:
  ```ini
  CUSTOM_EMOJIS = gitea, codeberg, gitlab, git, github, gogs, forgejo
  ```

- <a name="ui.DEFAULT_SHOW_FULL_NAME" href="#ui.DEFAULT_SHOW_FULL_NAME">`ui.DEFAULT_SHOW_FULL_NAME`</a>:
  Whether the full name of the users should be shown where possible. If the full name isn't set, the username will be used.:
  ```ini
  DEFAULT_SHOW_FULL_NAME = false
  ```

- <a name="ui.SEARCH_REPO_DESCRIPTION" href="#ui.SEARCH_REPO_DESCRIPTION">`ui.SEARCH_REPO_DESCRIPTION`</a>:
  Whether to search within description at repository search on explore page:
  ```ini
  SEARCH_REPO_DESCRIPTION = true
  ```

- <a name="ui.ONLY_SHOW_RELEVANT_REPOS" href="#ui.ONLY_SHOW_RELEVANT_REPOS">`ui.ONLY_SHOW_RELEVANT_REPOS`</a>:
  Whether to only show relevant repos on the explore page when no keyword is specified and default sorting is used.
  A repo is considered irrelevant if it's a fork or if it has no metadata (no description, no icon, no topic).:
  ```ini
  ONLY_SHOW_RELEVANT_REPOS = false
  ```

- <a name="ui.EXPLORE_PAGING_DEFAULT_SORT" href="#ui.EXPLORE_PAGING_DEFAULT_SORT">`ui.EXPLORE_PAGING_DEFAULT_SORT`</a>:
  Change the sort type of the explore pages.
  Default is "recentupdate", but you also have "alphabetically", "reverselastlogin", "newest", "oldest":
  ```ini
  EXPLORE_PAGING_DEFAULT_SORT = recentupdate
  ```

- <a name="ui.PREFERRED_TIMESTAMP_TENSE" href="#ui.PREFERRED_TIMESTAMP_TENSE">`ui.PREFERRED_TIMESTAMP_TENSE`</a>:
  The tense all timestamps should be rendered in. Possible values are `absolute` time (i.e. 1970-01-01, 11:59) and `mixed`.
  `mixed` means most timestamps are rendered in relative time (i.e. 2 days ago):
  ```ini
  PREFERRED_TIMESTAMP_TENSE = mixed
  ```

### <a name="ui.admin" href="#ui.admin">UI admin</a>

```ini
[ui.admin]
```

- <a name="ui.admin.USER_PAGING_NUM" href="#ui.admin.USER_PAGING_NUM">`ui.admin.USER_PAGING_NUM`</a>:
  Number of users that are displayed on one page:
  ```ini
  USER_PAGING_NUM = 50
  ```

- <a name="ui.admin.REPO_PAGING_NUM" href="#ui.admin.REPO_PAGING_NUM">`ui.admin.REPO_PAGING_NUM`</a>:
  Number of repos that are displayed on one page:
  ```ini
  REPO_PAGING_NUM = 50
  ```

- <a name="ui.admin.NOTICE_PAGING_NUM" href="#ui.admin.NOTICE_PAGING_NUM">`ui.admin.NOTICE_PAGING_NUM`</a>:
  Number of notices that are displayed on one page:
  ```ini
  NOTICE_PAGING_NUM = 25
  ```

- <a name="ui.admin.ORG_PAGING_NUM" href="#ui.admin.ORG_PAGING_NUM">`ui.admin.ORG_PAGING_NUM`</a>:
  Number of organizations that are displayed on one page:
  ```ini
  ORG_PAGING_NUM = 50
  ```

### <a name="ui.user" href="#ui.user">UI user</a>

```ini
[ui.user]
```

- <a name="ui.user.REPO_PAGING_NUM" href="#ui.user.REPO_PAGING_NUM">`ui.user.REPO_PAGING_NUM`</a>:
  Number of repos that are displayed on one page:
  ```ini
  REPO_PAGING_NUM = 15
  ```

### <a name="ui.meta" href="#ui.meta">UI metadata</a>

```ini
[ui.meta]
```

- <a name="ui.meta.AUTHOR" href="#ui.meta.AUTHOR">`ui.meta.AUTHOR`</a>:
  ```ini
  AUTHOR = Forgejo – Beyond coding. We forge.
  ```

- <a name="ui.meta.DESCRIPTION" href="#ui.meta.DESCRIPTION">`ui.meta.DESCRIPTION`</a>:
  ```ini
  DESCRIPTION = Forgejo is a self-hosted lightweight software forge. Easy to install and low maintenance, it just does the job.
  ```

- <a name="ui.meta.KEYWORDS" href="#ui.meta.KEYWORDS">`ui.meta.KEYWORDS`</a>:
  ```ini
  KEYWORDS = git,forge,forgejo
  ```

### <a name="ui.notification" href="#ui.notification">UI notification</a>

```ini
[ui.notification]
```

- <a name="ui.notification.MIN_TIMEOUT" href="#ui.notification.MIN_TIMEOUT">`ui.notification.MIN_TIMEOUT`</a>:
  Control how often the notification endpoint is polled to update the notification
  The timeout will increase to MAX_TIMEOUT in TIMEOUT_STEPs if the notification count is unchanged
  Set MIN_TIMEOUT to -1 to turn off:
  ```ini
  MIN_TIMEOUT = 10s
  ```

- <a name="ui.notification.MAX_TIMEOUT" href="#ui.notification.MAX_TIMEOUT">`ui.notification.MAX_TIMEOUT`</a>:
  ```ini
  MAX_TIMEOUT = 60s
  ```

- <a name="ui.notification.TIMEOUT_STEP" href="#ui.notification.TIMEOUT_STEP">`ui.notification.TIMEOUT_STEP`</a>:
  ```ini
  TIMEOUT_STEP = 10s
  ```

- <a name="ui.notification.EVENT_SOURCE_UPDATE_TIME" href="#ui.notification.EVENT_SOURCE_UPDATE_TIME">`ui.notification.EVENT_SOURCE_UPDATE_TIME`</a>:
  This setting determines how often the db is queried to get the latest notification counts.
  If the browser client supports EventSource and SharedWorker, a SharedWorker will be used in preference to polling notification. Set to -1 to disable the EventSource:
  ```ini
  EVENT_SOURCE_UPDATE_TIME = 10s
  ```

### <a name="ui.svg" href="#ui.svg">UI SVG images</a>

```ini
[ui.svg]
```

- <a name="ui.svg.ENABLE_RENDER" href="#ui.svg.ENABLE_RENDER">`ui.svg.ENABLE_RENDER`</a>:
  Whether to render SVG files as images.
  If SVG rendering is disabled, SVG files are displayed as text and cannot be embedded in markdown files as images.:
  ```ini
  ENABLE_RENDER = true
  ```

### <a name="ui.csv" href="#ui.csv">UI CSV files</a>

```ini
[ui.csv]
```

- <a name="ui.csv.MAX_FILE_SIZE" href="#ui.csv.MAX_FILE_SIZE">`ui.csv.MAX_FILE_SIZE`</a>:
  Maximum allowed file size in bytes to render CSV files as table. (Set to 0 for no limit):
  ```ini
  MAX_FILE_SIZE = 524288
  ```

- <a name="ui.csv.MAX_ROWS" href="#ui.csv.MAX_ROWS">`ui.csv.MAX_ROWS`</a>:
  Maximum allowed rows to render CSV files. Set to 0 for no limit.:
  ```ini
  MAX_ROWS = 2500
  ```

## <a name="markdown" href="#markdown">Markdown</a>

```ini
[markdown]
```

- <a name="markdown.ENABLE_HARD_LINE_BREAK_IN_COMMENTS" href="#markdown.ENABLE_HARD_LINE_BREAK_IN_COMMENTS">`markdown.ENABLE_HARD_LINE_BREAK_IN_COMMENTS`</a>:
  Render soft line breaks as hard line breaks, which means a single newline
  character between paragraphs will cause a line break and adding trailing
  whitespace to paragraphs is not necessary to force a line break.
  Render soft line breaks as hard line breaks for comments.:
  ```ini
  ENABLE_HARD_LINE_BREAK_IN_COMMENTS = true
  ```

- <a name="markdown.ENABLE_HARD_LINE_BREAK_IN_DOCUMENTS" href="#markdown.ENABLE_HARD_LINE_BREAK_IN_DOCUMENTS">`markdown.ENABLE_HARD_LINE_BREAK_IN_DOCUMENTS`</a>:
  Render soft line breaks as hard line breaks for markdown documents:
  ```ini
  ENABLE_HARD_LINE_BREAK_IN_DOCUMENTS = false
  ```

- <a name="markdown.CUSTOM_URL_SCHEMES" href="#markdown.CUSTOM_URL_SCHEMES">`markdown.CUSTOM_URL_SCHEMES`</a>:
  Comma separated list of custom URL-Schemes that are allowed as links when rendering Markdown
  for example git,magnet,ftp (more at https://en.wikipedia.org/wiki/List_of_URI_schemes)
  URLs starting with http and https are always displayed, whatever is put in this entry.
  If this entry is empty, all URL schemes are allowed.:
  ```ini
  CUSTOM_URL_SCHEMES =
  ```

- <a name="markdown.FILE_EXTENSIONS" href="#markdown.FILE_EXTENSIONS">`markdown.FILE_EXTENSIONS`</a>:
  List of file extensions that should be rendered/edited as Markdown
  Separate the extensions with a comma. To render files without any extension as markdown, just put a comma:
  ```ini
  FILE_EXTENSIONS = .md,.markdown,.mdown,.mkd,.livemd
  ```

- <a name="markdown.ENABLE_MATH" href="#markdown.ENABLE_MATH">`markdown.ENABLE_MATH`</a>:
  Enables math inline and block detection:
  ```ini
  ENABLE_MATH = true
  ```

## <a name="ssh.minimum_key_sizes" href="#ssh.minimum_key_sizes">SSH minimum key sizes</a>
Define allowed algorithms and their minimum key length (use -1 to disable a type).

```ini
[ssh.minimum_key_sizes]
```

- <a name="ssh.minimum_key_sizes.ED25519" href="#ssh.minimum_key_sizes.ED25519">`ssh.minimum_key_sizes.ED25519`</a>:
  Minimum ED25519 key size:
  ```ini
  ED25519 = 256
  ```

- <a name="ssh.minimum_key_sizes.ECDSA" href="#ssh.minimum_key_sizes.ECDSA">`ssh.minimum_key_sizes.ECDSA`</a>:
  Minimum ECDSA key size:
  ```ini
  ECDSA = 256
  ```

- <a name="ssh.minimum_key_sizes.RSA" href="#ssh.minimum_key_sizes.RSA">`ssh.minimum_key_sizes.RSA`</a>:
  Minimum RSA key size. The default value of `3071` ensures that an otherwise valid 3072 bit RSA key is allowed, because it may be reported as having 3071 bit length:
  ```ini
  RSA = 3071
  ```

- <a name="ssh.minimum_key_sizes.DSA" href="#ssh.minimum_key_sizes.DSA">`ssh.minimum_key_sizes.DSA`</a>:
  Minimum DSA key size. Set to 1024 to switch on.:
  ```ini
  DSA = -1
  ```

## <a name="indexer" href="#indexer">Indexer</a>

```ini
[indexer]
```

- <a name="indexer.ISSUE_INDEXER_TYPE" href="#indexer.ISSUE_INDEXER_TYPE">`indexer.ISSUE_INDEXER_TYPE`</a>:
  Issue Indexer settings
  Issue indexer type, currently support: bleve, db, elasticsearch or meilisearch default is bleve:
  ```ini
  ISSUE_INDEXER_TYPE = bleve
  ```

- <a name="indexer.ISSUE_INDEXER_PATH" href="#indexer.ISSUE_INDEXER_PATH">`indexer.ISSUE_INDEXER_PATH`</a>:
  Issue indexer storage path, available when <a href="#indexer.ISSUE_INDEXER_TYPE">`ISSUE_INDEXER_TYPE`</a> is bleve'
  Relative paths will be made absolute against the _<a href="#AppWorkPath">`AppWorkPath`</a>_.:
  ```ini
  ISSUE_INDEXER_PATH = indexers/issues.bleve
  ```

- <a name="indexer.ISSUE_INDEXER_CONN_STR" href="#indexer.ISSUE_INDEXER_CONN_STR">`indexer.ISSUE_INDEXER_CONN_STR`</a>:
  Issue indexer connection string, available when <a href="#indexer.ISSUE_INDEXER_TYPE">`ISSUE_INDEXER_TYPE`</a> is elasticsearch (e.g. `http://elastic:password@localhost:9200`) or meilisearch (e.g. `http://:apikey@localhost:7700`):
  ```ini
  ISSUE_INDEXER_CONN_STR =
  ```

- <a name="indexer.ISSUE_INDEXER_NAME" href="#indexer.ISSUE_INDEXER_NAME">`indexer.ISSUE_INDEXER_NAME`</a>:
  Issue indexer name, available when <a href="#indexer.ISSUE_INDEXER_TYPE">`ISSUE_INDEXER_TYPE`</a> is elasticsearch or meilisearch:
  ```ini
  ISSUE_INDEXER_NAME = gitea_issues
  ```

- <a name="indexer.STARTUP_TIMEOUT" href="#indexer.STARTUP_TIMEOUT">`indexer.STARTUP_TIMEOUT`</a>:
  Timeout the indexer if it takes longer than this to start.
  Set to -1 to disable timeout:
  ```ini
  STARTUP_TIMEOUT = 30s
  ```

- <a name="indexer.REPO_INDEXER_ENABLED" href="#indexer.REPO_INDEXER_ENABLED">`indexer.REPO_INDEXER_ENABLED`</a>:
  Repository Indexer settings
  repo indexer by default disabled, since it uses a lot of disk space:
  ```ini
  REPO_INDEXER_ENABLED = false
  ```

- <a name="indexer.REPO_INDEXER_REPO_TYPES" href="#indexer.REPO_INDEXER_REPO_TYPES">`indexer.REPO_INDEXER_REPO_TYPES`</a>:
  repo indexer units, the items to index, could be `sources`, `forks`, `mirrors`, `templates` or any combination of them separated by a comma.
  If empty then it defaults to `sources` only, as if you'd like to disable fully please see <a href="#indexer.REPO_INDEXER_ENABLED">`REPO_INDEXER_ENABLED`</a>.:
  ```ini
  REPO_INDEXER_REPO_TYPES = sources,forks,mirrors,templates
  ```

- <a name="indexer.REPO_INDEXER_TYPE" href="#indexer.REPO_INDEXER_TYPE">`indexer.REPO_INDEXER_TYPE`</a>:
  Code search engine type, could be `bleve` or `elasticsearch`:
  ```ini
  REPO_INDEXER_TYPE = bleve
  ```

- <a name="indexer.REPO_INDEXER_PATH" href="#indexer.REPO_INDEXER_PATH">`indexer.REPO_INDEXER_PATH`</a>:
  Index file used for code search. available when <a href="#indexer.REPO_INDEXER_TYPE">`REPO_INDEXER_TYPE`</a> is bleve:
  ```ini
  REPO_INDEXER_PATH = indexers/repos.bleve
  ```

- <a name="indexer.REPO_INDEXER_CONN_STR" href="#indexer.REPO_INDEXER_CONN_STR">`indexer.REPO_INDEXER_CONN_STR`</a>:
  Code indexer connection string, available when <a href="#indexer.REPO_INDEXER_TYPE">`REPO_INDEXER_TYPE`</a> is elasticsearch. i.e. `http://elastic:password@localhost:9200`:
  ```ini
  REPO_INDEXER_CONN_STR =
  ```

- <a name="indexer.REPO_INDEXER_NAME" href="#indexer.REPO_INDEXER_NAME">`indexer.REPO_INDEXER_NAME`</a>:
  Code indexer name, available when <a href="#indexer.REPO_INDEXER_TYPE">`REPO_INDEXER_TYPE`</a> is elasticsearch:
  ```ini
  REPO_INDEXER_NAME = gitea_codes
  ```

- <a name="indexer.REPO_INDEXER_INCLUDE" href="#indexer.REPO_INDEXER_INCLUDE">`indexer.REPO_INDEXER_INCLUDE`</a>:
  A comma separated list of glob patterns (see <https://github.com/gobwas/glob>) to include
  in the index; default is empty:
  ```ini
  REPO_INDEXER_INCLUDE =
  ```

- <a name="indexer.REPO_INDEXER_EXCLUDE" href="#indexer.REPO_INDEXER_EXCLUDE">`indexer.REPO_INDEXER_EXCLUDE`</a>:
  A comma separated list of glob patterns to exclude from the index; ; default is empty:
  ```ini
  REPO_INDEXER_EXCLUDE =
  ```

- <a name="indexer.REPO_INDEXER_EXCLUDE_VENDORED" href="#indexer.REPO_INDEXER_EXCLUDE_VENDORED">`indexer.REPO_INDEXER_EXCLUDE_VENDORED`</a>:
  If vendored files should be excluded.
  See <https://github.com/go-enry/go-enry> for more details which files are considered to be vendored:
  ```ini
  REPO_INDEXER_EXCLUDE_VENDORED = true
  ```

- <a name="indexer.MAX_FILE_SIZE" href="#indexer.MAX_FILE_SIZE">`indexer.MAX_FILE_SIZE`</a>:
  The maximum filesize to include for indexing:
  ```ini
  MAX_FILE_SIZE = 1048576
  ```

## <a name="queue" href="#queue">Queue</a>

```ini
[queue]
```

- <a name="queue.TYPE" href="#queue.TYPE">`queue.TYPE`</a>:
  Specific queues can be individually configured with `[queue.name]`. <a href="#queue">`[queue]`</a> provides defaults
  (`[queue.issue_indexer]` is special due to the old configuration described above)
  General queue type, currently support: `persistable-channel`, `channel`, `level`, `redis`, `dummy`.:
  ```ini
  TYPE = persistable-channel
  ```

- <a name="queue.DATADIR" href="#queue.DATADIR">`queue.DATADIR`</a>:
  Path to the directory for storing persistable queues and level queues.
  Individual queues will default to `queues/common` meaning the queue is shared.
  Relative paths will be made absolute against `%(APP_DATA_PATH)s`.:
  ```ini
  DATADIR = queues/
  ```

- <a name="queue.LENGTH" href="#queue.LENGTH">`queue.LENGTH`</a>:
  Default queue length before a channel queue will block:
  ```ini
  LENGTH = 100000
  ```

- <a name="queue.BATCH_LENGTH" href="#queue.BATCH_LENGTH">`queue.BATCH_LENGTH`</a>:
  Batch size to send for batched queues:
  ```ini
  BATCH_LENGTH = 20
  ```

- <a name="queue.CONN_STR" href="#queue.CONN_STR">`queue.CONN_STR`</a>:
  Connection string for redis queues this will store the redis (or Redis cluster) connection string.
  When <a href="#queue.TYPE">`TYPE`</a> is `persistable-channel`, this provides a directory for the underlying leveldb
  or additional options of the form `leveldb://path/to/db?option=value&....`, and will override <a href="#queue.DATADIR">`DATADIR`</a>:
  ```ini
  CONN_STR = redis://127.0.0.1:6379/0
  ```

- <a name="queue.QUEUE_NAME" href="#queue.QUEUE_NAME">`queue.QUEUE_NAME`</a>:
  Suffix of the default redis/disk queue name.
  Specific queues can be overridden within in their `[queue.name]` sections.:
  ```ini
  QUEUE_NAME = _queue
  ```

- <a name="queue.SET_NAME" href="#queue.SET_NAME">`queue.SET_NAME`</a>:
  Suffix of the default redis/disk unique queue set name.
  Specific queues can be overridden within in their `[queue.name]` sections.:
  ```ini
  SET_NAME = _unique
  ```

- <a name="queue.MAX_WORKERS" href="#queue.MAX_WORKERS">`queue.MAX_WORKERS`</a>:
  Maximum number of worker go-routines for the queue. Defaults to half the number of CPUs, clipped to between 1 and 10.:
  ```ini
  MAX_WORKERS =
  ```

## <a name="admin" href="#admin">Admin</a>

```ini
[admin]
```

- <a name="admin.DISABLE_REGULAR_ORG_CREATION" href="#admin.DISABLE_REGULAR_ORG_CREATION">`admin.DISABLE_REGULAR_ORG_CREATION`</a>:
  Disallow regular (non-admin) users from creating organizations:
  ```ini
  DISABLE_REGULAR_ORG_CREATION = false
  ```

- <a name="admin.DEFAULT_EMAIL_NOTIFICATIONS" href="#admin.DEFAULT_EMAIL_NOTIFICATIONS">`admin.DEFAULT_EMAIL_NOTIFICATIONS`</a>:
  Default configuration for email notifications for users (user configurable). Options: `enabled`, `onmention`, `disabled`.:
  ```ini
  DEFAULT_EMAIL_NOTIFICATIONS = enabled
  ```

- <a name="admin.SEND_NOTIFICATION_EMAIL_ON_NEW_USER" href="#admin.SEND_NOTIFICATION_EMAIL_ON_NEW_USER">`admin.SEND_NOTIFICATION_EMAIL_ON_NEW_USER`</a>:
  Send an email to all admins when a new user signs up to inform the admins about this act:
  ```ini
  SEND_NOTIFICATION_EMAIL_ON_NEW_USER = false
  ```

- <a name="admin.USER_DISABLED_FEATURES" href="#admin.USER_DISABLED_FEATURES">`admin.USER_DISABLED_FEATURES`</a>:
  Comma-separated list of disabled features for users.
  Features that can be disabled are:
  - `deletion`: a user cannot delete their own account
  - `manage_ssh_keys`: a user cannot configure ssh keys
  - `manage_gpg_keys`: a user cannot configure gpg keys:
  ```ini
  USER_DISABLED_FEATURES =
  ```

- <a name="admin.EXTERNAL_USER_DISABLE_FEATURES" href="#admin.EXTERNAL_USER_DISABLE_FEATURES">`admin.EXTERNAL_USER_DISABLE_FEATURES`</a>:
  Comma-separated list of disabled features ONLY if the user has an external login type (eg. LDAP, Oauth, etc.).
  This setting is independent from <a href="#admin.USER_DISABLED_FEATURES">`USER_DISABLED_FEATURES`</a> and supplements its behavior.
  Features that can be disabled are:
  - `deletion`: a user cannot delete their own account
  - `manage_ssh_keys`: a user cannot configure ssh keys
  - `manage_gpg_keys`: a user cannot configure gpg keys:
  ```ini
  EXTERNAL_USER_DISABLE_FEATURES =
  ```

## <a name="openid" href="#openid">Openid</a>

```ini
[openid]
```

- <a name="openid.ENABLE_OPENID_SIGNIN" href="#openid.ENABLE_OPENID_SIGNIN">`openid.ENABLE_OPENID_SIGNIN`</a>:
  Whether to allow signin in via OpenID.
  OpenID is an open, standard and decentralized authentication protocol.
  Your identity is the address of a webpage you provide, which describes
  how to prove you are in control of that page.
  For more info: <https://en.wikipedia.org/wiki/OpenID>
  Current implementation supports OpenID-2.0
  Tested to work providers at the time of writing:
  - Any GNUSocial node (your.hostname.tld/username)
  - Any SimpleID provider (<http://simpleid.koinic.net>)
  - <http://openid.org.cn/>
  - <openid.stackexchange.com>
  - <login.launchpad.net>
  - <username>.livejournal.com:
  ```ini
  ENABLE_OPENID_SIGNIN = true
  ```

- <a name="openid.ENABLE_OPENID_SIGNUP" href="#openid.ENABLE_OPENID_SIGNUP">`openid.ENABLE_OPENID_SIGNUP`</a>:
  Whether to allow registering via OpenID.
  Do not include to rely on the <a href="#service.DISABLE_REGISTRATION">`DISABLE_REGISTRATION`</a> setting.:
  ```ini
  ENABLE_OPENID_SIGNUP = true
  ```

- <a name="openid.WHITELISTED_URIS" href="#openid.WHITELISTED_URIS">`openid.WHITELISTED_URIS`</a>:
  Allowed URI patterns (POSIX regexp).
  Space separated.
  Only these would be allowed if non-blank.
  Example value: `trusted.domain.org trusted.domain.net`:
  ```ini
  WHITELISTED_URIS =
  ```

- <a name="openid.BLACKLISTED_URIS" href="#openid.BLACKLISTED_URIS">`openid.BLACKLISTED_URIS`</a>:
  Forbidden URI patterns (POSIX regexp).
  Space separated.
  Only used if <a href="#openid.WHITELISTED_URIS">`WHITELISTED_URIS`</a> is blank.
  Example value: `loadaverage.org/badguy stackexchange.com/.*spammer`:
  ```ini
  BLACKLISTED_URIS =
  ```

## <a name="oauth2_client" href="#oauth2_client">Oauth2 client</a>

```ini
[oauth2_client]
```

- <a name="oauth2_client.REGISTER_EMAIL_CONFIRM" href="#oauth2_client.REGISTER_EMAIL_CONFIRM">`oauth2_client.REGISTER_EMAIL_CONFIRM`</a>:
  Whether a new auto registered oauth2 user needs to confirm their email.
  Do not include to use the REGISTER_EMAIL_CONFIRM setting from the <a href="#service">`[service]`</a> section.:
  ```ini
  REGISTER_EMAIL_CONFIRM =
  ```

- <a name="oauth2_client.OPENID_CONNECT_SCOPES" href="#oauth2_client.OPENID_CONNECT_SCOPES">`oauth2_client.OPENID_CONNECT_SCOPES`</a>:
  Scopes for the openid connect oauth2 provider (separated by space, the openid scope is implicitly added).
  Typical values are profile and email.
  For more information about the possible values see <https://openid.net/specs/openid-connect-core-1_0.html#ScopeClaims>.:
  ```ini
  OPENID_CONNECT_SCOPES =
  ```

- <a name="oauth2_client.ENABLE_AUTO_REGISTRATION" href="#oauth2_client.ENABLE_AUTO_REGISTRATION">`oauth2_client.ENABLE_AUTO_REGISTRATION`</a>:
  Automatically create user accounts for new oauth2 users:
  ```ini
  ENABLE_AUTO_REGISTRATION = false
  ```

- <a name="oauth2_client.USERNAME" href="#oauth2_client.USERNAME">`oauth2_client.USERNAME`</a>:
  The source of the username for new oauth2 accounts:
  - `userid`:  use the userid / sub attribute
  - `nickname`: use the nickname attribute
  - <a href="#email">`email`</a>: use the username part of the email attribute
  
  Note: `nickname` and <a href="#email">`email`</a> options will normalize input strings using the following criteria:
  - diacritics are removed
  - the characters in the set `['´\x60]` are removed
  - the characters in the set `[\s~+]` are replaced with `-`:
  ```ini
  USERNAME = nickname
  ```

- <a name="oauth2_client.UPDATE_AVATAR" href="#oauth2_client.UPDATE_AVATAR">`oauth2_client.UPDATE_AVATAR`</a>:
  Update avatar if available from oauth2 provider.
  Update will be performed on each login:
  ```ini
  UPDATE_AVATAR = false
  ```

- <a name="oauth2_client.ACCOUNT_LINKING" href="#oauth2_client.ACCOUNT_LINKING">`oauth2_client.ACCOUNT_LINKING`</a>:
  How to handle if an account / email already exists:
  disabled = show an error
  login = show an account linking login
  auto = link directly with the account:
  ```ini
  ACCOUNT_LINKING = login
  ```

## <a name="webhook" href="#webhook">Webhook</a>

```ini
[webhook]
```

- <a name="webhook.QUEUE_LENGTH" href="#webhook.QUEUE_LENGTH">`webhook.QUEUE_LENGTH`</a>:
  Hook task queue length, increase if webhook shooting starts hanging:
  ```ini
  QUEUE_LENGTH = 1000
  ```

- <a name="webhook.DELIVER_TIMEOUT" href="#webhook.DELIVER_TIMEOUT">`webhook.DELIVER_TIMEOUT`</a>:
  Deliver timeout in seconds:
  ```ini
  DELIVER_TIMEOUT = 5
  ```

- <a name="webhook.ALLOWED_HOST_LIST" href="#webhook.ALLOWED_HOST_LIST">`webhook.ALLOWED_HOST_LIST`</a>:
  Webhook can only call allowed hosts for security reasons. Comma separated list, eg: `external, 192.168.1.0/24, *.mydomain.com`
  Built-in:
  - `loopback`: for localhost,
  - `private`: for LAN/intranet,
  - `external`: for public hosts on internet,
  - `*`: for all hosts
  
  CIDR list: `1.2.3.0/8, 2001:db8::/32`,
  Wildcard hosts: `*.mydomain.com, 192.168.100.*`.:
  ```ini
  ALLOWED_HOST_LIST = external
  ```

- <a name="webhook.SKIP_TLS_VERIFY" href="#webhook.SKIP_TLS_VERIFY">`webhook.SKIP_TLS_VERIFY`</a>:
  Allow insecure certification:
  ```ini
  SKIP_TLS_VERIFY = false
  ```

- <a name="webhook.PAGING_NUM" href="#webhook.PAGING_NUM">`webhook.PAGING_NUM`</a>:
  Number of history information in each page:
  ```ini
  PAGING_NUM = 10
  ```

- <a name="webhook.PROXY_URL" href="#webhook.PROXY_URL">`webhook.PROXY_URL`</a>:
  Proxy server URL, support `http://`, `https//`, `socks://`, blank will follow environment http_proxy/https_proxy.:
  ```ini
  PROXY_URL =
  ```

- <a name="webhook.PROXY_HOSTS" href="#webhook.PROXY_HOSTS">`webhook.PROXY_HOSTS`</a>:
  Comma separated list of host names requiring proxy. Glob patterns (`*`) are accepted; use `**` to match all hosts.:
  ```ini
  PROXY_HOSTS =
  ```

## <a name="mailer" href="#mailer">Mailer</a>

```ini
[mailer]
```

- <a name="mailer.ENABLED" href="#mailer.ENABLED">`mailer.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="mailer.SEND_BUFFER_LEN" href="#mailer.SEND_BUFFER_LEN">`mailer.SEND_BUFFER_LEN`</a>:
  Buffer length of channel, keep it as it is if you don't know what it is:
  ```ini
  SEND_BUFFER_LEN = 100
  ```

- <a name="mailer.SUBJECT_PREFIX" href="#mailer.SUBJECT_PREFIX">`mailer.SUBJECT_PREFIX`</a>:
  Prefix displayed before subject in mail:
  ```ini
  SUBJECT_PREFIX =
  ```

- <a name="mailer.PROTOCOL" href="#mailer.PROTOCOL">`mailer.PROTOCOL`</a>:
  Mail server protocol. One of `smtp`, `smtps`, `smtp+starttls`, `smtp+unix`, `sendmail`, `dummy`.
  - `sendmail`: use the operating system's `sendmail` command instead of SMTP. This is common on Linux systems.
  - `dummy`: send email messages to the log as a testing phase.
  If your provider does not explicitly say which protocol it uses but does provide a port,
  you can set <a href="#mailer.SMTP_PORT">`SMTP_PORT`</a> instead and this will be inferred.:
  ```ini
  PROTOCOL =
  ```

- <a name="mailer.SMTP_ADDR" href="#mailer.SMTP_ADDR">`mailer.SMTP_ADDR`</a>:
  Mail server address, e.g. `smtp.gmail.com`.
  For `smtp+unix`, this should be a path to a unix socket instead.:
  ```ini
  SMTP_ADDR =
  ```

- <a name="mailer.SMTP_PORT" href="#mailer.SMTP_PORT">`mailer.SMTP_PORT`</a>:
  Mail server port. Common ports are:
  - 25:  insecure SMTP
  - 465: SMTP Secure
  - 587: StartTLS
  If no protocol is specified, it will be inferred by this setting.:
  ```ini
  SMTP_PORT =
  ```

- <a name="mailer.ENABLE_HELO" href="#mailer.ENABLE_HELO">`mailer.ENABLE_HELO`</a>:
  Enable HELO operation. Defaults to true:
  ```ini
  ENABLE_HELO = true
  ```

- <a name="mailer.HELO_HOSTNAME" href="#mailer.HELO_HOSTNAME">`mailer.HELO_HOSTNAME`</a>:
  Custom hostname for HELO operation.
  If no value is provided, one is retrieved from system.:
  ```ini
  HELO_HOSTNAME =
  ```

- <a name="mailer.FORCE_TRUST_SERVER_CERT" href="#mailer.FORCE_TRUST_SERVER_CERT">`mailer.FORCE_TRUST_SERVER_CERT`</a>:
  If set to `true`, completely ignores server certificate validation errors.
  This option is unsafe.
  Consider adding the certificate to the system trust store instead.:
  ```ini
  FORCE_TRUST_SERVER_CERT = false
  ```

- <a name="mailer.USE_CLIENT_CERT" href="#mailer.USE_CLIENT_CERT">`mailer.USE_CLIENT_CERT`</a>:
  Use client certificate in connection:
  ```ini
  USE_CLIENT_CERT = false
  ```

- <a name="mailer.CLIENT_CERT_FILE" href="#mailer.CLIENT_CERT_FILE">`mailer.CLIENT_CERT_FILE`</a>:
  ```ini
  CLIENT_CERT_FILE = custom/mailer/cert.pem
  ```

- <a name="mailer.CLIENT_KEY_FILE" href="#mailer.CLIENT_KEY_FILE">`mailer.CLIENT_KEY_FILE`</a>:
  ```ini
  CLIENT_KEY_FILE = custom/mailer/key.pem
  ```

- <a name="mailer.FROM" href="#mailer.FROM">`mailer.FROM`</a>:
  Mail from address, RFC 5322. This can be just an email address, or the `"Name" <email@example.com>` format.:
  ```ini
  FROM =
  ```

- <a name="mailer.ENVELOPE_FROM" href="#mailer.ENVELOPE_FROM">`mailer.ENVELOPE_FROM`</a>:
  Sometimes it is helpful to use a different address on the envelope.
  Set this to use <a href="#mailer.ENVELOPE_FROM">`ENVELOPE_FROM`</a> as the from on the envelope.
  Set to `<>` to send an empty address.:
  ```ini
  ENVELOPE_FROM =
  ```

- <a name="mailer.FROM_DISPLAY_NAME_FORMAT" href="#mailer.FROM_DISPLAY_NAME_FORMAT">`mailer.FROM_DISPLAY_NAME_FORMAT`</a>:
  If Forgejo sends mails on behave of users, it will just use the name also displayed in the WebUI.
  If you want e.g. `Mister X (by ExampleCom) <forgejo@example.com>`, set it to `{{ .DisplayName }} (by {{ .AppName }})`.
  Available Variables: `.DisplayName`, `.AppName` and `.Domain`.:
  ```ini
  FROM_DISPLAY_NAME_FORMAT = {{ .DisplayName }}
  ```

- <a name="mailer.USER" href="#mailer.USER">`mailer.USER`</a>:
  Mailer user name, if required by provider:
  ```ini
  USER =
  ```

- <a name="mailer.PASSWD" href="#mailer.PASSWD">`mailer.PASSWD`</a>:
  Mailer password, if required by provider. Use `` PASSWD = `your password` `` for quoting if you use special characters in the password.:
  ```ini
  PASSWD =
  ```

- <a name="mailer.SEND_AS_PLAIN_TEXT" href="#mailer.SEND_AS_PLAIN_TEXT">`mailer.SEND_AS_PLAIN_TEXT`</a>:
  Send mails only in plain text, without HTML alternative:
  ```ini
  SEND_AS_PLAIN_TEXT = false
  ```

- <a name="mailer.SENDMAIL_PATH" href="#mailer.SENDMAIL_PATH">`mailer.SENDMAIL_PATH`</a>:
  Specify an alternative sendmail binary:
  ```ini
  SENDMAIL_PATH = sendmail
  ```

- <a name="mailer.SENDMAIL_ARGS" href="#mailer.SENDMAIL_ARGS">`mailer.SENDMAIL_ARGS`</a>:
  Specify any extra sendmail arguments
  WARNING: if your sendmail program interprets options you should set this to "--" or terminate these args with "--".:
  ```ini
  SENDMAIL_ARGS =
  ```

- <a name="mailer.SENDMAIL_TIMEOUT" href="#mailer.SENDMAIL_TIMEOUT">`mailer.SENDMAIL_TIMEOUT`</a>:
  Timeout for Sendmail:
  ```ini
  SENDMAIL_TIMEOUT = 5m
  ```

- <a name="mailer.SENDMAIL_CONVERT_CRLF" href="#mailer.SENDMAIL_CONVERT_CRLF">`mailer.SENDMAIL_CONVERT_CRLF`</a>:
  convert \r\n to \n for Sendmail:
  ```ini
  SENDMAIL_CONVERT_CRLF = true
  ```

### <a name="mailer.override_header" href="#mailer.override_header">Mailer override header</a>

```ini
[mailer.override_header]
```

- <a name="mailer.override_header.Reply-To" href="#mailer.override_header.Reply-To">`mailer.override_header.Reply-To`</a>:
  Reply-To mail addresses.
  Multiple addresses may be specified, separated by comma, e.g. `test@example.com, test2@example.com`.
  This is empty by default, use it only if you know what you need it for.:
  ```ini
  Reply-To =
  ```

- <a name="mailer.override_header.Content-Type" href="#mailer.override_header.Content-Type">`mailer.override_header.Content-Type`</a>:
  ```ini
  Content-Type = `text/html; charset=utf-8`
  ```

- <a name="mailer.override_header.In-Reply-To" href="#mailer.override_header.In-Reply-To">`mailer.override_header.In-Reply-To`</a>:
  ```ini
  In-Reply-To =
  ```

## <a name="email.incoming" href="#email.incoming">Incoming email</a>

```ini
[email.incoming]
```

- <a name="email.incoming.ENABLED" href="#email.incoming.ENABLED">`email.incoming.ENABLED`</a>:
  Enable handling of incoming emails:
  ```ini
  ENABLED = false
  ```

- <a name="email.incoming.REPLY_TO_ADDRESS" href="#email.incoming.REPLY_TO_ADDRESS">`email.incoming.REPLY_TO_ADDRESS`</a>:
  The email address including the %{token} placeholder that will be replaced per user/action.
  Example: incoming+%{token}@example.com
  The placeholder must appear in the user part of the address (before the @):
  ```ini
  REPLY_TO_ADDRESS =
  ```

- <a name="email.incoming.HOST" href="#email.incoming.HOST">`email.incoming.HOST`</a>:
  IMAP server host:
  ```ini
  HOST =
  ```

- <a name="email.incoming.PORT" href="#email.incoming.PORT">`email.incoming.PORT`</a>:
  IMAP server port:
  ```ini
  PORT =
  ```

- <a name="email.incoming.USERNAME" href="#email.incoming.USERNAME">`email.incoming.USERNAME`</a>:
  Username of the receiving account:
  ```ini
  USERNAME =
  ```

- <a name="email.incoming.PASSWORD" href="#email.incoming.PASSWORD">`email.incoming.PASSWORD`</a>:
  Password of the receiving account:
  ```ini
  PASSWORD =
  ```

- <a name="email.incoming.USE_TLS" href="#email.incoming.USE_TLS">`email.incoming.USE_TLS`</a>:
  Whether the IMAP server uses TLS:
  ```ini
  USE_TLS = false
  ```

- <a name="email.incoming.SKIP_TLS_VERIFY" href="#email.incoming.SKIP_TLS_VERIFY">`email.incoming.SKIP_TLS_VERIFY`</a>:
  If set to true, completely ignores server certificate validation errors. This option is unsafe:
  ```ini
  SKIP_TLS_VERIFY = true
  ```

- <a name="email.incoming.MAILBOX" href="#email.incoming.MAILBOX">`email.incoming.MAILBOX`</a>:
  The mailbox name where incoming mail will end up:
  ```ini
  MAILBOX = INBOX
  ```

- <a name="email.incoming.DELETE_HANDLED_MESSAGE" href="#email.incoming.DELETE_HANDLED_MESSAGE">`email.incoming.DELETE_HANDLED_MESSAGE`</a>:
  Whether handled messages should be deleted from the mailbox:
  ```ini
  DELETE_HANDLED_MESSAGE = true
  ```

- <a name="email.incoming.MAXIMUM_MESSAGE_SIZE" href="#email.incoming.MAXIMUM_MESSAGE_SIZE">`email.incoming.MAXIMUM_MESSAGE_SIZE`</a>:
  Maximum size of a message to handle. Bigger messages are ignored. Set to 0 to allow every size:
  ```ini
  MAXIMUM_MESSAGE_SIZE = 10485760
  ```

## <a name="cache" href="#cache">Cache</a>

```ini
[cache]
```

- <a name="cache.ADAPTER" href="#cache.ADAPTER">`cache.ADAPTER`</a>:
  Either "memory", "redis", "memcache", or "twoqueue". default is "memory":
  ```ini
  ADAPTER = memory
  ```

- <a name="cache.INTERVAL" href="#cache.INTERVAL">`cache.INTERVAL`</a>:
  For "memory" only, GC interval in seconds, default is 60:
  ```ini
  INTERVAL = 60
  ```

- <a name="cache.HOST" href="#cache.HOST">`cache.HOST`</a>:
  Connection host address, only if <a href="#cache.ADAPTER">`ADAPTER`</a> is `redis` or `memcache`:
  - `redis`:
    - `redis://127.0.0.1:6379/0?pool_size=100&idle_timeout=180s` or
    - `redis+cluster://127.0.0.1:6379/0?pool_size=100&idle_timeout=180s` for a Redis cluster
  - `memcache`: `127.0.0.1:11211`
  - `twoqueue`:
    - `{"size":50000,"recent_ratio":0.25,"ghost_ratio":0.5}` or
    - `50000`:
  ```ini
  HOST =
  ```

- <a name="cache.ITEM_TTL" href="#cache.ITEM_TTL">`cache.ITEM_TTL`</a>:
  Time to keep items in cache if not used, default is 16 hours.
  Setting it to -1 disables caching:
  ```ini
  ITEM_TTL = 16h
  ```

### <a name="cache.last_commit" href="#cache.last_commit">Cache last commit</a>
Last commit cache

```ini
[cache.last_commit]
```

- <a name="cache.last_commit.ITEM_TTL" href="#cache.last_commit.ITEM_TTL">`cache.last_commit.ITEM_TTL`</a>:
  Time to keep items in cache if not used, default is 8760 hours.
  Setting it to -1 disables caching:
  ```ini
  ITEM_TTL = 8760h
  ```

- <a name="cache.last_commit.COMMITS_COUNT" href="#cache.last_commit.COMMITS_COUNT">`cache.last_commit.COMMITS_COUNT`</a>:
  Only enable the cache when repository's commits count great than:
  ```ini
  COMMITS_COUNT = 1000
  ```

## <a name="session" href="#session">Session</a>

```ini
[session]
```

- <a name="session.PROVIDER" href="#session.PROVIDER">`session.PROVIDER`</a>:
  Session provider. Either `memory`, <a href="#log.file">`file`</a>, `redis`, `db`, `mysql`, `couchbase`, `memcache` or `postgres`.
  `db` will reuse the configuration in <a href="#database">`[database]`</a>.:
  ```ini
  PROVIDER = memory
  ```

- <a name="session.PROVIDER_CONFIG" href="#session.PROVIDER_CONFIG">`session.PROVIDER_CONFIG`</a>:
  Provider config options. Possible values depend on the <a href="#session.PROVIDER">`PROVIDER`</a>:
  - `memory`: doesn't have any config yet,
  - <a href="#log.file">`file`</a>: session file path, e.g. `data/sessions`, relative paths will be made absolute against _<a href="#AppWorkPath">`AppWorkPath`</a>_,
  - `redis`:
    - `redis://127.0.0.1:6379/0?pool_size=100&idle_timeout=180s` or
    - `redis+cluster://127.0.0.1:6379/0?pool_size=100&idle_timeout=180s` for a Redis cluster,
  - `mysql`: go-sql-driver/mysql dsn config string, e.g. `root:password@/session_table`:
  ```ini
  PROVIDER_CONFIG = data/sessions
  ```

- <a name="session.COOKIE_NAME" href="#session.COOKIE_NAME">`session.COOKIE_NAME`</a>:
  Session cookie name:
  ```ini
  COOKIE_NAME = i_like_gitea
  ```

- <a name="session.COOKIE_SECURE" href="#session.COOKIE_SECURE">`session.COOKIE_SECURE`</a>:
  If you use session in https only: true or false. If not set, it defaults to `true` if the ROOT_URL is an HTTPS URL:
  ```ini
  COOKIE_SECURE =
  ```

- <a name="session.GC_INTERVAL_TIME" href="#session.GC_INTERVAL_TIME">`session.GC_INTERVAL_TIME`</a>:
  Session GC time interval in seconds, default is 86400 (1 day):
  ```ini
  GC_INTERVAL_TIME = 86400
  ```

- <a name="session.SESSION_LIFE_TIME" href="#session.SESSION_LIFE_TIME">`session.SESSION_LIFE_TIME`</a>:
  Session life time in seconds, default is 86400 (1 day):
  ```ini
  SESSION_LIFE_TIME = 86400
  ```

- <a name="session.DOMAIN" href="#session.DOMAIN">`session.DOMAIN`</a>:
  Cookie domain name. Default is empty:
  ```ini
  DOMAIN =
  ```

- <a name="session.SAME_SITE" href="#session.SAME_SITE">`session.SAME_SITE`</a>:
  SameSite settings. Either "none", "lax", or "strict":
  ```ini
  SAME_SITE = lax
  ```

## <a name="picture" href="#picture">Picture</a>

```ini
[picture]
```

- <a name="picture.REPOSITORY_AVATAR_FALLBACK" href="#picture.REPOSITORY_AVATAR_FALLBACK">`picture.REPOSITORY_AVATAR_FALLBACK`</a>:
  How Forgejo deals with missing repository avatars
  none = no avatar will be displayed; random = random avatar will be displayed; image = default image will be used:
  ```ini
  REPOSITORY_AVATAR_FALLBACK = none
  ```

- <a name="picture.REPOSITORY_AVATAR_FALLBACK_IMAGE" href="#picture.REPOSITORY_AVATAR_FALLBACK_IMAGE">`picture.REPOSITORY_AVATAR_FALLBACK_IMAGE`</a>:
  ```ini
  REPOSITORY_AVATAR_FALLBACK_IMAGE = /img/repo_default.png
  ```

- <a name="picture.AVATAR_MAX_WIDTH" href="#picture.AVATAR_MAX_WIDTH">`picture.AVATAR_MAX_WIDTH`</a>:
  Max Width and Height of uploaded avatars.
  This is to limit the amount of RAM used when resizing the image:
  ```ini
  AVATAR_MAX_WIDTH = 4096
  ```

- <a name="picture.AVATAR_MAX_HEIGHT" href="#picture.AVATAR_MAX_HEIGHT">`picture.AVATAR_MAX_HEIGHT`</a>:
  ```ini
  AVATAR_MAX_HEIGHT = 4096
  ```

- <a name="picture.AVATAR_RENDERED_SIZE_FACTOR" href="#picture.AVATAR_RENDERED_SIZE_FACTOR">`picture.AVATAR_RENDERED_SIZE_FACTOR`</a>:
  The multiplication factor for rendered avatar images.
  Larger values result in finer rendering on HiDPI devices:
  ```ini
  AVATAR_RENDERED_SIZE_FACTOR = 2
  ```

- <a name="picture.AVATAR_MAX_FILE_SIZE" href="#picture.AVATAR_MAX_FILE_SIZE">`picture.AVATAR_MAX_FILE_SIZE`</a>:
  Maximum allowed file size for uploaded avatars.
  This is to limit the amount of RAM used when resizing the image:
  ```ini
  AVATAR_MAX_FILE_SIZE = 1048576
  ```

- <a name="picture.AVATAR_MAX_ORIGIN_SIZE" href="#picture.AVATAR_MAX_ORIGIN_SIZE">`picture.AVATAR_MAX_ORIGIN_SIZE`</a>:
  If the uploaded file is not larger than this byte size, the image will be used as is, without resizing/converting:
  ```ini
  AVATAR_MAX_ORIGIN_SIZE = 262144
  ```

- <a name="picture.GRAVATAR_SOURCE" href="#picture.GRAVATAR_SOURCE">`picture.GRAVATAR_SOURCE`</a>:
  Chinese users can choose "duoshuo"
  or a custom avatar source, like: http://cn.gravatar.com/avatar/:
  ```ini
  GRAVATAR_SOURCE = gravatar
  ```

- <a name="picture.DISABLE_GRAVATAR" href="#picture.DISABLE_GRAVATAR">`picture.DISABLE_GRAVATAR`</a>:
  This value will always be true in offline mode:
  ```ini
  DISABLE_GRAVATAR = false
  ```

- <a name="picture.ENABLE_FEDERATED_AVATAR" href="#picture.ENABLE_FEDERATED_AVATAR">`picture.ENABLE_FEDERATED_AVATAR`</a>:
  Federated avatar lookup uses DNS to discover avatar associated
  with emails, see https://www.libravatar.org
  This value will always be false in offline mode or when Gravatar is disabled:
  ```ini
  ENABLE_FEDERATED_AVATAR = false
  ```

## <a name="attachment" href="#attachment">Attachment</a>

```ini
[attachment]
```

- <a name="attachment.ENABLED" href="#attachment.ENABLED">`attachment.ENABLED`</a>:
  Whether issue and pull request attachments are enabled. Defaults to `true`:
  ```ini
  ENABLED = true
  ```

- <a name="attachment.ALLOWED_TYPES" href="#attachment.ALLOWED_TYPES">`attachment.ALLOWED_TYPES`</a>:
  Comma-separated list of allowed file extensions (`.zip`), mime types (`text/plain`) or wildcard type (`image/*`, `audio/*`, `video/*`). Empty value or `*/*` allows all types.:
  ```ini
  ALLOWED_TYPES = .avif,.cpuprofile,.csv,.dmp,.docx,.fodg,.fodp,.fods,.fodt,.gif,.gz,.jpeg,.jpg,.json,.jsonc,.log,.md,.mov,.mp4,.odf,.odg,.odp,.ods,.odt,.patch,.pdf,.png,.pptx,.svg,.tgz,.txt,.webm,.xls,.xlsx,.zip
  ```

- <a name="attachment.MAX_SIZE" href="#attachment.MAX_SIZE">`attachment.MAX_SIZE`</a>:
  Max size of each file in MB:
  ```ini
  MAX_SIZE = 2048
  ```

- <a name="attachment.MAX_FILES" href="#attachment.MAX_FILES">`attachment.MAX_FILES`</a>:
  Max number of files per upload:
  ```ini
  MAX_FILES = 5
  ```

- <a name="attachment.STORAGE_TYPE" href="#attachment.STORAGE_TYPE">`attachment.STORAGE_TYPE`</a>:
  Storage type for attachments, <a href="#repository.local">`local`</a> for local disk or `minio` for s3 compatible
  object storage service.`:
  ```ini
  STORAGE_TYPE = local
  ```

- <a name="attachment.SERVE_DIRECT" href="#attachment.SERVE_DIRECT">`attachment.SERVE_DIRECT`</a>:
  Allows the storage driver to redirect to authenticated URLs to serve files directly
  Currently, only `minio` is supported.:
  ```ini
  SERVE_DIRECT = false
  ```

- <a name="attachment.PATH" href="#attachment.PATH">`attachment.PATH`</a>:
  Path for attachments. Defaults to `attachments`. Only available when STORAGE_TYPE is <a href="#repository.local">`local`</a>
  Relative paths will be resolved to `${AppDataPath}/${attachment.PATH}`:
  ```ini
  PATH = attachments
  ```

- <a name="attachment.MINIO_ENDPOINT" href="#attachment.MINIO_ENDPOINT">`attachment.MINIO_ENDPOINT`</a>:
  Minio endpoint to connect only available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_ENDPOINT = localhost:9000
  ```

- <a name="attachment.MINIO_ACCESS_KEY_ID" href="#attachment.MINIO_ACCESS_KEY_ID">`attachment.MINIO_ACCESS_KEY_ID`</a>:
  Minio access key ID to connect, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`.
  If not provided and <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`, Forgejo will search for credentials in known
  environment variables (<a href="#attachment.MINIO_ACCESS_KEY_ID">`MINIO_ACCESS_KEY_ID`</a>, `AWS_ACCESS_KEY_ID`), credentials files
  (`~/.mc/config.json`, `~/.aws/credentials`), and EC2 instance metadata.:
  ```ini
  MINIO_ACCESS_KEY_ID =
  ```

- <a name="attachment.MINIO_SECRET_ACCESS_KEY" href="#attachment.MINIO_SECRET_ACCESS_KEY">`attachment.MINIO_SECRET_ACCESS_KEY`</a>:
  Minio secret access key to connect, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`:
  ```ini
  MINIO_SECRET_ACCESS_KEY =
  ```

- <a name="attachment.MINIO_BUCKET" href="#attachment.MINIO_BUCKET">`attachment.MINIO_BUCKET`</a>:
  Minio bucket to store the attachments, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`:
  ```ini
  MINIO_BUCKET = gitea
  ```

- <a name="attachment.MINIO_BUCKET_LOOKUP" href="#attachment.MINIO_BUCKET_LOOKUP">`attachment.MINIO_BUCKET_LOOKUP`</a>:
  URL lookup for the minio bucket, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`
  Available values: `auto`, `dns, `path`.
  If empty, it behaves the same as `auto` was set:
  ```ini
  MINIO_BUCKET_LOOKUP = auto
  ```

- <a name="attachment.MINIO_LOCATION" href="#attachment.MINIO_LOCATION">`attachment.MINIO_LOCATION`</a>:
  Minio location to create bucket, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`:
  ```ini
  MINIO_LOCATION = us-east-1
  ```

- <a name="attachment.MINIO_BASE_PATH" href="#attachment.MINIO_BASE_PATH">`attachment.MINIO_BASE_PATH`</a>:
  Minio base path on the bucket, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`:
  ```ini
  MINIO_BASE_PATH = attachments/
  ```

- <a name="attachment.MINIO_USE_SSL" href="#attachment.MINIO_USE_SSL">`attachment.MINIO_USE_SSL`</a>:
  Minio enabled SSL, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`:
  ```ini
  MINIO_USE_SSL = false
  ```

- <a name="attachment.MINIO_INSECURE_SKIP_VERIFY" href="#attachment.MINIO_INSECURE_SKIP_VERIFY">`attachment.MINIO_INSECURE_SKIP_VERIFY`</a>:
  Minio skip SSL verification, only available when <a href="#attachment.STORAGE_TYPE">`STORAGE_TYPE`</a> is `minio`:
  ```ini
  MINIO_INSECURE_SKIP_VERIFY = false
  ```

- <a name="attachment.MINIO_CHECKSUM_ALGORITHM" href="#attachment.MINIO_CHECKSUM_ALGORITHM">`attachment.MINIO_CHECKSUM_ALGORITHM`</a>:
  Minio checksum algorithm:
  - `default`: for MinIO or AWS S3, or
  - `md5`: for Cloudflare or Backblaze.:
  ```ini
  MINIO_CHECKSUM_ALGORITHM = default
  ```

- <a name="time.DEFAULT_UI_LOCATION" href="#time.DEFAULT_UI_LOCATION">`time.DEFAULT_UI_LOCATION`</a>:
  Location of the UI time display i.e. `Asia/Shanghai`
  Empty means server's location setting.:
  ```ini
  DEFAULT_UI_LOCATION =
  ```

## <a name="cron" href="#cron">Cron</a>

```ini
[cron]
```

- <a name="cron.ENABLED" href="#cron.ENABLED">`cron.ENABLED`</a>:
  Whether to enable cron tasks execution:
  ```ini
  ENABLED = false
  ```

- <a name="cron.RUN_AT_START" href="#cron.RUN_AT_START">`cron.RUN_AT_START`</a>:
  Whether to run all enabled cron tasks when Forgejo starts:
  ```ini
  RUN_AT_START = false
  ```

### <a name="cron.archive_cleanup" href="#cron.archive_cleanup">Cron archive cleanup</a>
Clean up old repository archives.
Note: ``SCHEDULE`` accept formats
- Full crontab specs, e.g. `* * * * * ?`
- Descriptors, e.g. `@midnight`, `@every 1h30m`
See more: <https://pkg.go.dev/github.com/gogs/cron@v0.0.0-20171120032916-9f6c956d3e14>

```ini
[cron.archive_cleanup]
```

- <a name="cron.archive_cleanup.ENABLED" href="#cron.archive_cleanup.ENABLED">`cron.archive_cleanup.ENABLED`</a>:
  Whether to enable the job:
  ```ini
  ENABLED = true
  ```

- <a name="cron.archive_cleanup.RUN_AT_START" href="#cron.archive_cleanup.RUN_AT_START">`cron.archive_cleanup.RUN_AT_START`</a>:
  Whether to always run at least once at start up time (if <a href="#cron.archive_cleanup.ENABLED">`ENABLED`</a>):
  ```ini
  RUN_AT_START = true
  ```

- <a name="cron.archive_cleanup.NOTICE_ON_SUCCESS" href="#cron.archive_cleanup.NOTICE_ON_SUCCESS">`cron.archive_cleanup.NOTICE_ON_SUCCESS`</a>:
  Whether to emit notice on successful execution too:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.archive_cleanup.SCHEDULE" href="#cron.archive_cleanup.SCHEDULE">`cron.archive_cleanup.SCHEDULE`</a>:
  Time interval for job to run:
  ```ini
  SCHEDULE = @midnight
  ```

- <a name="cron.archive_cleanup.OLDER_THAN" href="#cron.archive_cleanup.OLDER_THAN">`cron.archive_cleanup.OLDER_THAN`</a>:
  Archives created more than <a href="#cron.archive_cleanup.OLDER_THAN">`OLDER_THAN`</a> ago are subject to deletion:
  ```ini
  OLDER_THAN = 24h
  ```

### <a name="cron.update_mirrors" href="#cron.update_mirrors">Cron update mirrors</a>
Update mirrors

```ini
[cron.update_mirrors]
```

- <a name="cron.update_mirrors.SCHEDULE" href="#cron.update_mirrors.SCHEDULE">`cron.update_mirrors.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 10m
  ```

- <a name="cron.update_mirrors.ENABLED" href="#cron.update_mirrors.ENABLED">`cron.update_mirrors.ENABLED`</a>:
  Enable running Update mirrors task periodically:
  ```ini
  ENABLED = true
  ```

- <a name="cron.update_mirrors.RUN_AT_START" href="#cron.update_mirrors.RUN_AT_START">`cron.update_mirrors.RUN_AT_START`</a>:
  Run Update mirrors task when Forgejo starts:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.update_mirrors.NOTICE_ON_SUCCESS" href="#cron.update_mirrors.NOTICE_ON_SUCCESS">`cron.update_mirrors.NOTICE_ON_SUCCESS`</a>:
  Notice if not success:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.update_mirrors.PULL_LIMIT" href="#cron.update_mirrors.PULL_LIMIT">`cron.update_mirrors.PULL_LIMIT`</a>:
  Limit the number of mirrors added to the queue to this number
  (negative values mean no limit, 0 will result in no result in no mirrors being queued effectively disabling pull mirror updating.):
  ```ini
  PULL_LIMIT = 50
  ```

- <a name="cron.update_mirrors.PUSH_LIMIT" href="#cron.update_mirrors.PUSH_LIMIT">`cron.update_mirrors.PUSH_LIMIT`</a>:
  Limit the number of mirrors added to the queue to this number
  (negative values mean no limit, 0 will result in no mirrors being queued effectively disabling push mirror updating):
  ```ini
  PUSH_LIMIT = 50
  ```

### <a name="cron.repo_health_check" href="#cron.repo_health_check">Cron repo health check</a>
Repository health check

```ini
[cron.repo_health_check]
```

- <a name="cron.repo_health_check.SCHEDULE" href="#cron.repo_health_check.SCHEDULE">`cron.repo_health_check.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @midnight
  ```

- <a name="cron.repo_health_check.ENABLED" href="#cron.repo_health_check.ENABLED">`cron.repo_health_check.ENABLED`</a>:
  Enable running Repository health check task periodically:
  ```ini
  ENABLED = true
  ```

- <a name="cron.repo_health_check.RUN_AT_START" href="#cron.repo_health_check.RUN_AT_START">`cron.repo_health_check.RUN_AT_START`</a>:
  Run Repository health check task when Forgejo starts:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.repo_health_check.NOTICE_ON_SUCCESS" href="#cron.repo_health_check.NOTICE_ON_SUCCESS">`cron.repo_health_check.NOTICE_ON_SUCCESS`</a>:
  Notice if not success:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.repo_health_check.TIMEOUT" href="#cron.repo_health_check.TIMEOUT">`cron.repo_health_check.TIMEOUT`</a>:
  ```ini
  TIMEOUT = 60s
  ```

- <a name="cron.repo_health_check.ARGS" href="#cron.repo_health_check.ARGS">`cron.repo_health_check.ARGS`</a>:
  Arguments for command 'git fsck', e.g. "--unreachable --tags"
  see more on http://git-scm.com/docs/git-fsck:
  ```ini
  ARGS =
  ```

### <a name="cron.check_repo_stats" href="#cron.check_repo_stats">Cron check repo stats</a>
Check repository statistics

```ini
[cron.check_repo_stats]
```

- <a name="cron.check_repo_stats.ENABLED" href="#cron.check_repo_stats.ENABLED">`cron.check_repo_stats.ENABLED`</a>:
  Enable running check repository statistics task periodically:
  ```ini
  ENABLED = true
  ```

- <a name="cron.check_repo_stats.RUN_AT_START" href="#cron.check_repo_stats.RUN_AT_START">`cron.check_repo_stats.RUN_AT_START`</a>:
  Run check repository statistics task when Forgejo starts:
  ```ini
  RUN_AT_START = true
  ```

- <a name="cron.check_repo_stats.NOTICE_ON_SUCCESS" href="#cron.check_repo_stats.NOTICE_ON_SUCCESS">`cron.check_repo_stats.NOTICE_ON_SUCCESS`</a>:
  Notice if not success:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.check_repo_stats.SCHEDULE" href="#cron.check_repo_stats.SCHEDULE">`cron.check_repo_stats.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @midnight
  ```

### <a name="cron.update_migration_poster_id" href="#cron.update_migration_poster_id">Cron update migration poster id</a>

```ini
[cron.update_migration_poster_id]
```

- <a name="cron.update_migration_poster_id.ENABLED" href="#cron.update_migration_poster_id.ENABLED">`cron.update_migration_poster_id.ENABLED`</a>:
  Update migrated repositories' issues and comments' posterid, it will always attempt synchronization when the instance starts:
  ```ini
  ENABLED = true
  ```

- <a name="cron.update_migration_poster_id.RUN_AT_START" href="#cron.update_migration_poster_id.RUN_AT_START">`cron.update_migration_poster_id.RUN_AT_START`</a>:
  Update migrated repositories' issues and comments' posterid when starting server (default true):
  ```ini
  RUN_AT_START = true
  ```

- <a name="cron.update_migration_poster_id.NOTICE_ON_SUCCESS" href="#cron.update_migration_poster_id.NOTICE_ON_SUCCESS">`cron.update_migration_poster_id.NOTICE_ON_SUCCESS`</a>:
  Notice if not success:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.update_migration_poster_id.SCHEDULE" href="#cron.update_migration_poster_id.SCHEDULE">`cron.update_migration_poster_id.SCHEDULE`</a>:
  Interval as a duration between each synchronization. (default every 24h):
  ```ini
  SCHEDULE = @midnight
  ```

### <a name="cron.sync_external_users" href="#cron.sync_external_users">Cron sync external users</a>
Synchronize external user data (only LDAP user synchronization is supported)

```ini
[cron.sync_external_users]
```

- <a name="cron.sync_external_users.ENABLED" href="#cron.sync_external_users.ENABLED">`cron.sync_external_users.ENABLED`</a>:
  ```ini
  ENABLED = true
  ```

- <a name="cron.sync_external_users.RUN_AT_START" href="#cron.sync_external_users.RUN_AT_START">`cron.sync_external_users.RUN_AT_START`</a>:
  Synchronize external user data when starting server (default false):
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.sync_external_users.NOTICE_ON_SUCCESS" href="#cron.sync_external_users.NOTICE_ON_SUCCESS">`cron.sync_external_users.NOTICE_ON_SUCCESS`</a>:
  Notice if not success:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.sync_external_users.SCHEDULE" href="#cron.sync_external_users.SCHEDULE">`cron.sync_external_users.SCHEDULE`</a>:
  Interval as a duration between each synchronization (default every 24h):
  ```ini
  SCHEDULE = @midnight
  ```

- <a name="cron.sync_external_users.UPDATE_EXISTING" href="#cron.sync_external_users.UPDATE_EXISTING">`cron.sync_external_users.UPDATE_EXISTING`</a>:
  Create new users, update existing user data and disable users that are not in external source anymore (default)
  or only create new users if UPDATE_EXISTING is set to false:
  ```ini
  UPDATE_EXISTING = true
  ```

### <a name="cron.cleanup_actions" href="#cron.cleanup_actions">Cron cleanup actions</a>
Cleanup expired actions assets

```ini
[cron.cleanup_actions]
```

- <a name="cron.cleanup_actions.ENABLED" href="#cron.cleanup_actions.ENABLED">`cron.cleanup_actions.ENABLED`</a>:
  ```ini
  ENABLED = true
  ```

- <a name="cron.cleanup_actions.RUN_AT_START" href="#cron.cleanup_actions.RUN_AT_START">`cron.cleanup_actions.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = true
  ```

- <a name="cron.cleanup_actions.SCHEDULE" href="#cron.cleanup_actions.SCHEDULE">`cron.cleanup_actions.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @midnight
  ```

### <a name="cron.deleted_branches_cleanup" href="#cron.deleted_branches_cleanup">Cron deleted branches cleanup</a>
Clean-up deleted branches

```ini
[cron.deleted_branches_cleanup]
```

- <a name="cron.deleted_branches_cleanup.ENABLED" href="#cron.deleted_branches_cleanup.ENABLED">`cron.deleted_branches_cleanup.ENABLED`</a>:
  ```ini
  ENABLED = true
  ```

- <a name="cron.deleted_branches_cleanup.RUN_AT_START" href="#cron.deleted_branches_cleanup.RUN_AT_START">`cron.deleted_branches_cleanup.RUN_AT_START`</a>:
  Clean-up deleted branches when starting server (default true):
  ```ini
  RUN_AT_START = true
  ```

- <a name="cron.deleted_branches_cleanup.NOTICE_ON_SUCCESS" href="#cron.deleted_branches_cleanup.NOTICE_ON_SUCCESS">`cron.deleted_branches_cleanup.NOTICE_ON_SUCCESS`</a>:
  Notice if not success:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.deleted_branches_cleanup.SCHEDULE" href="#cron.deleted_branches_cleanup.SCHEDULE">`cron.deleted_branches_cleanup.SCHEDULE`</a>:
  Interval as a duration between each synchronization (default every 24h):
  ```ini
  SCHEDULE = @midnight
  ```

- <a name="cron.deleted_branches_cleanup.OLDER_THAN" href="#cron.deleted_branches_cleanup.OLDER_THAN">`cron.deleted_branches_cleanup.OLDER_THAN`</a>:
  deleted branches than OLDER_THAN ago are subject to deletion:
  ```ini
  OLDER_THAN = 24h
  ```

### <a name="cron.cleanup_hook_task_table" href="#cron.cleanup_hook_task_table">Cron cleanup hook task table</a>
Cleanup hook_task table

```ini
[cron.cleanup_hook_task_table]
```

- <a name="cron.cleanup_hook_task_table.ENABLED" href="#cron.cleanup_hook_task_table.ENABLED">`cron.cleanup_hook_task_table.ENABLED`</a>:
  Whether to enable the job:
  ```ini
  ENABLED = true
  ```

- <a name="cron.cleanup_hook_task_table.RUN_AT_START" href="#cron.cleanup_hook_task_table.RUN_AT_START">`cron.cleanup_hook_task_table.RUN_AT_START`</a>:
  Whether to always run at start up time (if ENABLED):
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.cleanup_hook_task_table.SCHEDULE" href="#cron.cleanup_hook_task_table.SCHEDULE">`cron.cleanup_hook_task_table.SCHEDULE`</a>:
  Time interval for job to run:
  ```ini
  SCHEDULE = @midnight
  ```

- <a name="cron.cleanup_hook_task_table.CLEANUP_TYPE" href="#cron.cleanup_hook_task_table.CLEANUP_TYPE">`cron.cleanup_hook_task_table.CLEANUP_TYPE`</a>:
  OlderThan or PerWebhook. How the records are removed, either by age (i.e. how long ago hook_task record was delivered) or by the number to keep per webhook (i.e. keep most recent x deliveries per webhook):
  ```ini
  CLEANUP_TYPE = OlderThan
  ```

- <a name="cron.cleanup_hook_task_table.OLDER_THAN" href="#cron.cleanup_hook_task_table.OLDER_THAN">`cron.cleanup_hook_task_table.OLDER_THAN`</a>:
  If CLEANUP_TYPE is set to OlderThan, then any delivered hook_task records older than this expression will be deleted:
  ```ini
  OLDER_THAN = 168h
  ```

- <a name="cron.cleanup_hook_task_table.NUMBER_TO_KEEP" href="#cron.cleanup_hook_task_table.NUMBER_TO_KEEP">`cron.cleanup_hook_task_table.NUMBER_TO_KEEP`</a>:
  If CLEANUP_TYPE is set to PerWebhook, this is number of hook_task records to keep for a webhook (i.e. keep the most recent x deliveries):
  ```ini
  NUMBER_TO_KEEP = 10
  ```

### <a name="cron.cleanup_packages" href="#cron.cleanup_packages">Cron cleanup packages</a>
Cleanup expired packages

```ini
[cron.cleanup_packages]
```

- <a name="cron.cleanup_packages.ENABLED" href="#cron.cleanup_packages.ENABLED">`cron.cleanup_packages.ENABLED`</a>:
  Whether to enable the job:
  ```ini
  ENABLED = true
  ```

- <a name="cron.cleanup_packages.RUN_AT_START" href="#cron.cleanup_packages.RUN_AT_START">`cron.cleanup_packages.RUN_AT_START`</a>:
  Whether to always run at least once at start up time (if <a href="#cron.cleanup_packages.ENABLED">`ENABLED`</a>):
  ```ini
  RUN_AT_START = true
  ```

- <a name="cron.cleanup_packages.NOTICE_ON_SUCCESS" href="#cron.cleanup_packages.NOTICE_ON_SUCCESS">`cron.cleanup_packages.NOTICE_ON_SUCCESS`</a>:
  Whether to emit notice on successful execution too:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.cleanup_packages.SCHEDULE" href="#cron.cleanup_packages.SCHEDULE">`cron.cleanup_packages.SCHEDULE`</a>:
  Time interval for job to run:
  ```ini
  SCHEDULE = @midnight
  ```

- <a name="cron.cleanup_packages.OLDER_THAN" href="#cron.cleanup_packages.OLDER_THAN">`cron.cleanup_packages.OLDER_THAN`</a>:
  Unreferenced blobs created more than OLDER_THAN ago are subject to deletion:
  ```ini
  OLDER_THAN = 24h
  ```

### <a name="cron.delete_inactive_accounts" href="#cron.delete_inactive_accounts">Cron delete inactive accounts</a>
Delete all unactivated accounts

```ini
[cron.delete_inactive_accounts]
```

- <a name="cron.delete_inactive_accounts.ENABLED" href="#cron.delete_inactive_accounts.ENABLED">`cron.delete_inactive_accounts.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.delete_inactive_accounts.RUN_AT_START" href="#cron.delete_inactive_accounts.RUN_AT_START">`cron.delete_inactive_accounts.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.delete_inactive_accounts.NOTICE_ON_SUCCESS" href="#cron.delete_inactive_accounts.NOTICE_ON_SUCCESS">`cron.delete_inactive_accounts.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.delete_inactive_accounts.SCHEDULE" href="#cron.delete_inactive_accounts.SCHEDULE">`cron.delete_inactive_accounts.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @annually
  ```

- <a name="cron.delete_inactive_accounts.OLDER_THAN" href="#cron.delete_inactive_accounts.OLDER_THAN">`cron.delete_inactive_accounts.OLDER_THAN`</a>:
  ```ini
  OLDER_THAN = 168h
  ```

### <a name="cron.delete_repo_archives" href="#cron.delete_repo_archives">Cron delete repo archives</a>
Delete all repository archives

```ini
[cron.delete_repo_archives]
```

- <a name="cron.delete_repo_archives.ENABLED" href="#cron.delete_repo_archives.ENABLED">`cron.delete_repo_archives.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.delete_repo_archives.RUN_AT_START" href="#cron.delete_repo_archives.RUN_AT_START">`cron.delete_repo_archives.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.delete_repo_archives.NOTICE_ON_SUCCESS" href="#cron.delete_repo_archives.NOTICE_ON_SUCCESS">`cron.delete_repo_archives.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.delete_repo_archives.SCHEDULE" href="#cron.delete_repo_archives.SCHEDULE">`cron.delete_repo_archives.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @annually
  ```

### <a name="cron.git_gc_repos" href="#cron.git_gc_repos">Cron git gc repos</a>
Garbage collect all repositories

```ini
[cron.git_gc_repos]
```

- <a name="cron.git_gc_repos.ENABLED" href="#cron.git_gc_repos.ENABLED">`cron.git_gc_repos.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.git_gc_repos.RUN_AT_START" href="#cron.git_gc_repos.RUN_AT_START">`cron.git_gc_repos.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.git_gc_repos.NOTICE_ON_SUCCESS" href="#cron.git_gc_repos.NOTICE_ON_SUCCESS">`cron.git_gc_repos.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.git_gc_repos.SCHEDULE" href="#cron.git_gc_repos.SCHEDULE">`cron.git_gc_repos.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 72h
  ```

- <a name="cron.git_gc_repos.TIMEOUT" href="#cron.git_gc_repos.TIMEOUT">`cron.git_gc_repos.TIMEOUT`</a>:
  ```ini
  TIMEOUT = 60s
  ```

- <a name="cron.git_gc_repos.ARGS" href="#cron.git_gc_repos.ARGS">`cron.git_gc_repos.ARGS`</a>:
  Arguments for command 'git gc'
  The default value is same with [git] -> GC_ARGS:
  ```ini
  ARGS =
  ```

### <a name="cron.resync_all_sshkeys" href="#cron.resync_all_sshkeys">Cron resync all sshkeys</a>
Update the '.ssh/authorized_keys' file with Forgejo SSH keys

```ini
[cron.resync_all_sshkeys]
```

- <a name="cron.resync_all_sshkeys.ENABLED" href="#cron.resync_all_sshkeys.ENABLED">`cron.resync_all_sshkeys.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.resync_all_sshkeys.RUN_AT_START" href="#cron.resync_all_sshkeys.RUN_AT_START">`cron.resync_all_sshkeys.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.resync_all_sshkeys.NOTICE_ON_SUCCESS" href="#cron.resync_all_sshkeys.NOTICE_ON_SUCCESS">`cron.resync_all_sshkeys.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.resync_all_sshkeys.SCHEDULE" href="#cron.resync_all_sshkeys.SCHEDULE">`cron.resync_all_sshkeys.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 72h
  ```

### <a name="cron.resync_all_hooks" href="#cron.resync_all_hooks">Cron resync all hooks</a>
Resynchronize pre-receive, update and post-receive hooks of all repositories

```ini
[cron.resync_all_hooks]
```

- <a name="cron.resync_all_hooks.ENABLED" href="#cron.resync_all_hooks.ENABLED">`cron.resync_all_hooks.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.resync_all_hooks.RUN_AT_START" href="#cron.resync_all_hooks.RUN_AT_START">`cron.resync_all_hooks.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.resync_all_hooks.NOTICE_ON_SUCCESS" href="#cron.resync_all_hooks.NOTICE_ON_SUCCESS">`cron.resync_all_hooks.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.resync_all_hooks.SCHEDULE" href="#cron.resync_all_hooks.SCHEDULE">`cron.resync_all_hooks.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 72h
  ```

### <a name="cron.reinit_missing_repos" href="#cron.reinit_missing_repos">Cron reinit missing repos</a>
Reinitialize all missing Git repositories for which records exist

```ini
[cron.reinit_missing_repos]
```

- <a name="cron.reinit_missing_repos.ENABLED" href="#cron.reinit_missing_repos.ENABLED">`cron.reinit_missing_repos.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.reinit_missing_repos.RUN_AT_START" href="#cron.reinit_missing_repos.RUN_AT_START">`cron.reinit_missing_repos.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.reinit_missing_repos.NOTICE_ON_SUCCESS" href="#cron.reinit_missing_repos.NOTICE_ON_SUCCESS">`cron.reinit_missing_repos.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.reinit_missing_repos.SCHEDULE" href="#cron.reinit_missing_repos.SCHEDULE">`cron.reinit_missing_repos.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 72h
  ```

### <a name="cron.delete_missing_repos" href="#cron.delete_missing_repos">Cron delete missing repos</a>
Delete all repositories missing their Git files

```ini
[cron.delete_missing_repos]
```

- <a name="cron.delete_missing_repos.ENABLED" href="#cron.delete_missing_repos.ENABLED">`cron.delete_missing_repos.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.delete_missing_repos.RUN_AT_START" href="#cron.delete_missing_repos.RUN_AT_START">`cron.delete_missing_repos.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.delete_missing_repos.NOTICE_ON_SUCCESS" href="#cron.delete_missing_repos.NOTICE_ON_SUCCESS">`cron.delete_missing_repos.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.delete_missing_repos.SCHEDULE" href="#cron.delete_missing_repos.SCHEDULE">`cron.delete_missing_repos.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 72h
  ```

### <a name="cron.delete_generated_repository_avatars" href="#cron.delete_generated_repository_avatars">Cron delete generated repository avatars</a>
Delete generated repository avatars

```ini
[cron.delete_generated_repository_avatars]
```

- <a name="cron.delete_generated_repository_avatars.ENABLED" href="#cron.delete_generated_repository_avatars.ENABLED">`cron.delete_generated_repository_avatars.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.delete_generated_repository_avatars.RUN_AT_START" href="#cron.delete_generated_repository_avatars.RUN_AT_START">`cron.delete_generated_repository_avatars.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.delete_generated_repository_avatars.NOTICE_ON_SUCCESS" href="#cron.delete_generated_repository_avatars.NOTICE_ON_SUCCESS">`cron.delete_generated_repository_avatars.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.delete_generated_repository_avatars.SCHEDULE" href="#cron.delete_generated_repository_avatars.SCHEDULE">`cron.delete_generated_repository_avatars.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 72h
  ```

### <a name="cron.delete_old_actions" href="#cron.delete_old_actions">Cron delete old actions</a>
Delete all old activities from database

```ini
[cron.delete_old_actions]
```

- <a name="cron.delete_old_actions.ENABLED" href="#cron.delete_old_actions.ENABLED">`cron.delete_old_actions.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.delete_old_actions.RUN_AT_START" href="#cron.delete_old_actions.RUN_AT_START">`cron.delete_old_actions.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.delete_old_actions.NOTICE_ON_SUCCESS" href="#cron.delete_old_actions.NOTICE_ON_SUCCESS">`cron.delete_old_actions.NOTICE_ON_SUCCESS`</a>:
  ```ini
  NOTICE_ON_SUCCESS = false
  ```

- <a name="cron.delete_old_actions.SCHEDULE" href="#cron.delete_old_actions.SCHEDULE">`cron.delete_old_actions.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 168h
  ```

- <a name="cron.delete_old_actions.OLDER_THAN" href="#cron.delete_old_actions.OLDER_THAN">`cron.delete_old_actions.OLDER_THAN`</a>:
  ```ini
  OLDER_THAN = 8760h
  ```

### <a name="cron.update_checker" href="#cron.update_checker">Cron update checker</a>
Check for new Forgejo versions

```ini
[cron.update_checker]
```

- <a name="cron.update_checker.ENABLED" href="#cron.update_checker.ENABLED">`cron.update_checker.ENABLED`</a>:
  ```ini
  ENABLED = true
  ```

- <a name="cron.update_checker.RUN_AT_START" href="#cron.update_checker.RUN_AT_START">`cron.update_checker.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.update_checker.ENABLE_SUCCESS_NOTICE" href="#cron.update_checker.ENABLE_SUCCESS_NOTICE">`cron.update_checker.ENABLE_SUCCESS_NOTICE`</a>:
  ```ini
  ENABLE_SUCCESS_NOTICE = false
  ```

- <a name="cron.update_checker.SCHEDULE" href="#cron.update_checker.SCHEDULE">`cron.update_checker.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 168h
  ```

- <a name="cron.update_checker.HTTP_ENDPOINT" href="#cron.update_checker.HTTP_ENDPOINT">`cron.update_checker.HTTP_ENDPOINT`</a>:
  ```ini
  HTTP_ENDPOINT = https://dl.gitea.com/gitea/version.json
  ```

- <a name="cron.update_checker.DOMAIN_ENDPOINT" href="#cron.update_checker.DOMAIN_ENDPOINT">`cron.update_checker.DOMAIN_ENDPOINT`</a>:
  ```ini
  DOMAIN_ENDPOINT = release.forgejo.org
  ```

### <a name="cron.delete_old_system_notices" href="#cron.delete_old_system_notices">Cron delete old system notices</a>
Delete all old system notices from database

```ini
[cron.delete_old_system_notices]
```

- <a name="cron.delete_old_system_notices.ENABLED" href="#cron.delete_old_system_notices.ENABLED">`cron.delete_old_system_notices.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.delete_old_system_notices.RUN_AT_START" href="#cron.delete_old_system_notices.RUN_AT_START">`cron.delete_old_system_notices.RUN_AT_START`</a>:
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.delete_old_system_notices.NO_SUCCESS_NOTICE" href="#cron.delete_old_system_notices.NO_SUCCESS_NOTICE">`cron.delete_old_system_notices.NO_SUCCESS_NOTICE`</a>:
  ```ini
  NO_SUCCESS_NOTICE = false
  ```

- <a name="cron.delete_old_system_notices.SCHEDULE" href="#cron.delete_old_system_notices.SCHEDULE">`cron.delete_old_system_notices.SCHEDULE`</a>:
  ```ini
  SCHEDULE = @every 168h
  ```

- <a name="cron.delete_old_system_notices.OLDER_THAN" href="#cron.delete_old_system_notices.OLDER_THAN">`cron.delete_old_system_notices.OLDER_THAN`</a>:
  ```ini
  OLDER_THAN = 8760h
  ```

### <a name="cron.gc_lfs" href="#cron.gc_lfs">Cron gc lfs</a>
Garbage collect LFS pointers in repositories

```ini
[cron.gc_lfs]
```

- <a name="cron.gc_lfs.ENABLED" href="#cron.gc_lfs.ENABLED">`cron.gc_lfs.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="cron.gc_lfs.RUN_AT_START" href="#cron.gc_lfs.RUN_AT_START">`cron.gc_lfs.RUN_AT_START`</a>:
  Garbage collect LFS pointers in repositories (default false):
  ```ini
  RUN_AT_START = false
  ```

- <a name="cron.gc_lfs.SCHEDULE" href="#cron.gc_lfs.SCHEDULE">`cron.gc_lfs.SCHEDULE`</a>:
  Interval as a duration between each gc run (default every 24h):
  ```ini
  SCHEDULE = @every 24h
  ```

- <a name="cron.gc_lfs.OLDER_THAN" href="#cron.gc_lfs.OLDER_THAN">`cron.gc_lfs.OLDER_THAN`</a>:
  Only attempt to garbage collect LFSMetaObjects older than this (default 7 days):
  ```ini
  OLDER_THAN = 168h
  ```

- <a name="cron.gc_lfs.LAST_UPDATED_MORE_THAN_AGO" href="#cron.gc_lfs.LAST_UPDATED_MORE_THAN_AGO">`cron.gc_lfs.LAST_UPDATED_MORE_THAN_AGO`</a>:
  Only attempt to garbage collect LFSMetaObjects that have not been attempted to be garbage collected for this long (default 3 days):
  ```ini
  LAST_UPDATED_MORE_THAN_AGO = 72h
  ```

- <a name="cron.gc_lfs.NUMBER_TO_CHECK_PER_REPO" href="#cron.gc_lfs.NUMBER_TO_CHECK_PER_REPO">`cron.gc_lfs.NUMBER_TO_CHECK_PER_REPO`</a>:
  Minimum number of stale LFSMetaObjects to check per repo. Set to `0` to always check all:
  ```ini
  NUMBER_TO_CHECK_PER_REPO = 100
  ```

- <a name="cron.gc_lfs.PROPORTION_TO_CHECK_PER_REPO" href="#cron.gc_lfs.PROPORTION_TO_CHECK_PER_REPO">`cron.gc_lfs.PROPORTION_TO_CHECK_PER_REPO`</a>:
  Check at least this proportion of LFSMetaObjects per repo. (This may cause all stale LFSMetaObjects to be checked.):
  ```ini
  PROPORTION_TO_CHECK_PER_REPO = 0.6
  ```

## <a name="mirror" href="#mirror">Mirror</a>

```ini
[mirror]
```

- <a name="mirror.ENABLED" href="#mirror.ENABLED">`mirror.ENABLED`</a>:
  Enables the mirror functionality. Set to **false** to disable all mirrors. Pre-existing mirrors remain valid but won't be updated; may be converted to regular repo:
  ```ini
  ENABLED = true
  ```

- <a name="mirror.DISABLE_NEW_PULL" href="#mirror.DISABLE_NEW_PULL">`mirror.DISABLE_NEW_PULL`</a>:
  Disable the creation of **new** pull mirrors. Pre-existing mirrors remain valid. Will be ignored if <a href="#mirror.ENABLED">`mirror.ENABLED`</a> is `false`:
  ```ini
  DISABLE_NEW_PULL = false
  ```

- <a name="mirror.DISABLE_NEW_PUSH" href="#mirror.DISABLE_NEW_PUSH">`mirror.DISABLE_NEW_PUSH`</a>:
  Disable the creation of **new** push mirrors. Pre-existing mirrors remain valid. Will be ignored if <a href="#mirror.ENABLED">`mirror.ENABLED`</a> is `false`:
  ```ini
  DISABLE_NEW_PUSH = false
  ```

- <a name="mirror.DEFAULT_INTERVAL" href="#mirror.DEFAULT_INTERVAL">`mirror.DEFAULT_INTERVAL`</a>:
  Default interval as a duration between each check:
  ```ini
  DEFAULT_INTERVAL = 8h
  ```

- <a name="mirror.MIN_INTERVAL" href="#mirror.MIN_INTERVAL">`mirror.MIN_INTERVAL`</a>:
  Min interval as a duration must be > 1m:
  ```ini
  MIN_INTERVAL = 10m
  ```

## <a name="api" href="#api">Api</a>

```ini
[api]
```

- <a name="api.ENABLE_SWAGGER" href="#api.ENABLE_SWAGGER">`api.ENABLE_SWAGGER`</a>:
  Enables the API documentation endpoints (/api/swagger, /api/v1/swagger, …). True or false:
  ```ini
  ENABLE_SWAGGER = true
  ```

- <a name="api.MAX_RESPONSE_ITEMS" href="#api.MAX_RESPONSE_ITEMS">`api.MAX_RESPONSE_ITEMS`</a>:
  Max number of items in a page:
  ```ini
  MAX_RESPONSE_ITEMS = 50
  ```

- <a name="api.DEFAULT_PAGING_NUM" href="#api.DEFAULT_PAGING_NUM">`api.DEFAULT_PAGING_NUM`</a>:
  Default paging number of api:
  ```ini
  DEFAULT_PAGING_NUM = 30
  ```

- <a name="api.DEFAULT_GIT_TREES_PER_PAGE" href="#api.DEFAULT_GIT_TREES_PER_PAGE">`api.DEFAULT_GIT_TREES_PER_PAGE`</a>:
  Default and maximum number of items per page for git trees api:
  ```ini
  DEFAULT_GIT_TREES_PER_PAGE = 1000
  ```

- <a name="api.DEFAULT_MAX_BLOB_SIZE" href="#api.DEFAULT_MAX_BLOB_SIZE">`api.DEFAULT_MAX_BLOB_SIZE`</a>:
  Default max size of a blob returned by the blobs API (default is 10MiB):
  ```ini
  DEFAULT_MAX_BLOB_SIZE = 10485760
  ```

## <a name="i18n" href="#i18n">I18n</a>

```ini
[i18n]
```

- <a name="i18n.LANGS" href="#i18n.LANGS">`i18n.LANGS`</a>:
  The first locale will be used as the default if user browser's language doesn't match any locale in the list:
  ```ini
  LANGS = en-US,zh-CN,zh-HK,zh-TW,de-DE,nds,fr-FR,nl-NL,lv-LV,ru-RU,uk-UA,ja-JP,es-ES,pt-BR,pt-PT,pl-PL,bg,it-IT,fi-FI,fil,eo,tr-TR,cs-CZ,sl,sv-SE,ko-KR,el-GR,fa-IR,hu-HU,id-ID
  ```

- <a name="i18n.NAMES" href="#i18n.NAMES">`i18n.NAMES`</a>:
  ```ini
  NAMES = English,简体中文,繁體中文（香港）,繁體中文（台灣）,Deutsch,Plattdüütsch,Français,Nederlands,Latviešu,Русский,Українська,日本語,Español,Português do Brasil,Português de Portugal,Polski,Български,Italiano,Suomi,Filipino,Esperanto,Türkçe,Čeština,Slovenščina,Svenska,한국어,Ελληνικά,فارسی,Magyar nyelv,Bahasa Indonesia
  ```

## <a name="other" href="#other">Other</a>
Extension mapping to highlight class
e.g. .toml=ini

```ini
[other]
```

- <a name="other.SHOW_FOOTER_VERSION" href="#other.SHOW_FOOTER_VERSION">`other.SHOW_FOOTER_VERSION`</a>:
  Show version information about Forgejo and Go in the footer:
  ```ini
  SHOW_FOOTER_VERSION = true
  ```

- <a name="other.SHOW_FOOTER_TEMPLATE_LOAD_TIME" href="#other.SHOW_FOOTER_TEMPLATE_LOAD_TIME">`other.SHOW_FOOTER_TEMPLATE_LOAD_TIME`</a>:
  Show template execution time in the footer:
  ```ini
  SHOW_FOOTER_TEMPLATE_LOAD_TIME = true
  ```

- <a name="other.SHOW_FOOTER_POWERED_BY" href="#other.SHOW_FOOTER_POWERED_BY">`other.SHOW_FOOTER_POWERED_BY`</a>:
  Show the "powered by" text in the footer:
  ```ini
  SHOW_FOOTER_POWERED_BY = true
  ```

- <a name="other.ENABLE_SITEMAP" href="#other.ENABLE_SITEMAP">`other.ENABLE_SITEMAP`</a>:
  Generate sitemap. Defaults to `true`:
  ```ini
  ENABLE_SITEMAP = true
  ```

- <a name="other.ENABLE_FEED" href="#other.ENABLE_FEED">`other.ENABLE_FEED`</a>:
  Enable/Disable RSS/Atom feed:
  ```ini
  ENABLE_FEED = true
  ```

## <a name="markup" href="#markup">Markup</a>

```ini
[markup]
```

- <a name="markup.MERMAID_MAX_SOURCE_CHARACTERS" href="#markup.MERMAID_MAX_SOURCE_CHARACTERS">`markup.MERMAID_MAX_SOURCE_CHARACTERS`</a>:
  Set the maximum number of characters in a mermaid source. (Set to -1 to disable limits):
  ```ini
  MERMAID_MAX_SOURCE_CHARACTERS = 5000
  ```

- <a name="markup.FILEPREVIEW_MAX_LINES" href="#markup.FILEPREVIEW_MAX_LINES">`markup.FILEPREVIEW_MAX_LINES`</a>:
  Set the maximum number of lines allowed for a filepreview. (Set to -1 to disable limits; set to 0 to disable the feature):
  ```ini
  FILEPREVIEW_MAX_LINES = 50
  ```

### <a name="markup.sanitizer.1" href="#markup.sanitizer.1">Markup sanitizer</a>
This section can appear multiple times by adding a unique alphanumeric suffix to define multiple rules.
e.g., <a href="#markup.sanitizer.1">`[markup.sanitizer.1]`</a> -> `[markup.sanitizer.2]` -> `[markup.sanitizer.TeX]`
The following keys can appear once to define a sanitation policy rule.

```ini
[markup.sanitizer.1]
```

- <a name="markup.sanitizer.1.ELEMENT" href="#markup.sanitizer.1.ELEMENT">`markup.sanitizer.1.ELEMENT`</a>:
  ```ini
  ELEMENT = span
  ```

- <a name="markup.sanitizer.1.ALLOW_ATTR" href="#markup.sanitizer.1.ALLOW_ATTR">`markup.sanitizer.1.ALLOW_ATTR`</a>:
  ```ini
  ALLOW_ATTR = class
  ```

- <a name="markup.sanitizer.1.REGEXP" href="#markup.sanitizer.1.REGEXP">`markup.sanitizer.1.REGEXP`</a>:
  ```ini
  REGEXP = ^(info|warning|error)$
  ```

### <a name="markup.asciidoc" href="#markup.asciidoc">Markup asciidoc</a>
Other markup formats e.g. <a href="#markup.asciidoc">`asciidoc`</a>
uncomment and enable the below section.
(You can add other markup formats by copying the section and adjusting
the section name suffix <a href="#markup.asciidoc">`asciidoc`</a> to something else.)

```ini
[markup.asciidoc]
```

- <a name="markup.asciidoc.ENABLED" href="#markup.asciidoc.ENABLED">`markup.asciidoc.ENABLED`</a>:
  ```ini
  ENABLED = false
  ```

- <a name="markup.asciidoc.FILE_EXTENSIONS" href="#markup.asciidoc.FILE_EXTENSIONS">`markup.asciidoc.FILE_EXTENSIONS`</a>:
  List of file extensions that should be rendered by an external command:
  ```ini
  FILE_EXTENSIONS = .adoc,.asciidoc
  ```

- <a name="markup.asciidoc.RENDER_COMMAND" href="#markup.asciidoc.RENDER_COMMAND">`markup.asciidoc.RENDER_COMMAND`</a>:
  External command to render all matching extensions:
  ```ini
  RENDER_COMMAND = asciidoc --out-file=- -
  ```

- <a name="markup.asciidoc.IS_INPUT_FILE" href="#markup.asciidoc.IS_INPUT_FILE">`markup.asciidoc.IS_INPUT_FILE`</a>:
  Don't pass the file on STDIN, pass the filename as argument instead:
  ```ini
  IS_INPUT_FILE = false
  ```

- <a name="markup.asciidoc.RENDER_CONTENT_MODE" href="#markup.asciidoc.RENDER_CONTENT_MODE">`markup.asciidoc.RENDER_CONTENT_MODE`</a>:
  How the content will be rendered.
  * `sanitized`: Sanitize the content and render it inside current page, default to only allow a few HTML tags and attributes. Customized sanitizer rules can be defined in [markup.sanitizer.*] .
  * `no-sanitizer`: Disable the sanitizer and render the content inside current page. It's **insecure** and may lead to XSS attack if the content contains malicious code.
  * `iframe`: Render the content in a separate standalone page and embed it into current page by iframe. The iframe is in sandbox mode with same-origin disabled, and the JS code are safely isolated from parent page:
  ```ini
  RENDER_CONTENT_MODE = sanitized
  ```

## <a name="metrics" href="#metrics">Metrics</a>

```ini
[metrics]
```

- <a name="metrics.ENABLED" href="#metrics.ENABLED">`metrics.ENABLED`</a>:
  Enables metrics endpoint. True or false; default is false:
  ```ini
  ENABLED = false
  ```

- <a name="metrics.TOKEN" href="#metrics.TOKEN">`metrics.TOKEN`</a>:
  If you want to add authorization, specify a token here:
  ```ini
  TOKEN =
  ```

- <a name="metrics.ENABLED_ISSUE_BY_LABEL" href="#metrics.ENABLED_ISSUE_BY_LABEL">`metrics.ENABLED_ISSUE_BY_LABEL`</a>:
  Enable issue by label metrics; default is false:
  ```ini
  ENABLED_ISSUE_BY_LABEL = false
  ```

- <a name="metrics.ENABLED_ISSUE_BY_REPOSITORY" href="#metrics.ENABLED_ISSUE_BY_REPOSITORY">`metrics.ENABLED_ISSUE_BY_REPOSITORY`</a>:
  Enable issue by repository metrics; default is false:
  ```ini
  ENABLED_ISSUE_BY_REPOSITORY = false
  ```

## <a name="migrations" href="#migrations">Migrations</a>

```ini
[migrations]
```

- <a name="migrations.MAX_ATTEMPTS" href="#migrations.MAX_ATTEMPTS">`migrations.MAX_ATTEMPTS`</a>:
  Max attempts per http/https request on migrations:
  ```ini
  MAX_ATTEMPTS = 3
  ```

- <a name="migrations.RETRY_BACKOFF" href="#migrations.RETRY_BACKOFF">`migrations.RETRY_BACKOFF`</a>:
  Backoff time per http/https request retry (seconds):
  ```ini
  RETRY_BACKOFF = 3
  ```

- <a name="migrations.ALLOWED_DOMAINS" href="#migrations.ALLOWED_DOMAINS">`migrations.ALLOWED_DOMAINS`</a>:
  Allowed domains for migrating, default is blank. Blank means everything will be allowed.
  Multiple domains could be separated by commas.
  Wildcard is supported: "github.com, *.github.com":
  ```ini
  ALLOWED_DOMAINS =
  ```

- <a name="migrations.BLOCKED_DOMAINS" href="#migrations.BLOCKED_DOMAINS">`migrations.BLOCKED_DOMAINS`</a>:
  Blocklist for migrating, default is blank. Multiple domains could be separated by commas.
  When ALLOWED_DOMAINS is not blank, this option has a higher priority to deny domains.
  Wildcard is supported:
  ```ini
  BLOCKED_DOMAINS =
  ```

- <a name="migrations.ALLOW_LOCALNETWORKS" href="#migrations.ALLOW_LOCALNETWORKS">`migrations.ALLOW_LOCALNETWORKS`</a>:
  Allow private addresses defined by RFC 1918, RFC 1122, RFC 4632 and RFC 4291 (false by default)
  If a domain is allowed by ALLOWED_DOMAINS, this option will be ignored:
  ```ini
  ALLOW_LOCALNETWORKS = false
  ```

- <a name="migrations.SKIP_TLS_VERIFY" href="#migrations.SKIP_TLS_VERIFY">`migrations.SKIP_TLS_VERIFY`</a>:
  If set to true, completely ignores server certificate validation errors. This option is unsafe:
  ```ini
  SKIP_TLS_VERIFY = false
  ```

- <a name="F3.ENABLED" href="#F3.ENABLED">`F3.ENABLED`</a>:
  Enable/Disable Friendly Forge Format (F3):
  ```ini
  ENABLED = false
  ```

## <a name="federation" href="#federation">Federation</a>

```ini
[federation]
```

- <a name="federation.ENABLED" href="#federation.ENABLED">`federation.ENABLED`</a>:
  Enable/Disable federation capabilities:
  ```ini
  ENABLED = false
  ```

- <a name="federation.SHARE_USER_STATISTICS" href="#federation.SHARE_USER_STATISTICS">`federation.SHARE_USER_STATISTICS`</a>:
  Enable/Disable user statistics for nodeinfo if federation is enabled:
  ```ini
  SHARE_USER_STATISTICS = true
  ```

- <a name="federation.MAX_SIZE" href="#federation.MAX_SIZE">`federation.MAX_SIZE`</a>:
  Maximum federation request and response size (MB):
  ```ini
  MAX_SIZE = 4
  ```

- <a name="federation.ALGORITHMS" href="#federation.ALGORITHMS">`federation.ALGORITHMS`</a>:
  HTTP signature algorithms.
  WARNING: Changing the settings below can break federation.:
  ```ini
  ALGORITHMS = rsa-sha256, rsa-sha512, ed25519
  ```

- <a name="federation.DIGEST_ALGORITHM" href="#federation.DIGEST_ALGORITHM">`federation.DIGEST_ALGORITHM`</a>:
  HTTP signature digest algorithm:
  ```ini
  DIGEST_ALGORITHM = SHA-256
  ```

- <a name="federation.GET_HEADERS" href="#federation.GET_HEADERS">`federation.GET_HEADERS`</a>:
  GET headers for federation requests:
  ```ini
  GET_HEADERS = (request-target), Date
  ```

- <a name="federation.POST_HEADERS" href="#federation.POST_HEADERS">`federation.POST_HEADERS`</a>:
  POST headers for federation requests:
  ```ini
  POST_HEADERS = (request-target), Date, Digest
  ```

## <a name="packages" href="#packages">Packages</a>

```ini
[packages]
```

- <a name="packages.ENABLED" href="#packages.ENABLED">`packages.ENABLED`</a>:
  Enable/Disable package registry capabilities:
  ```ini
  ENABLED = true
  ```

- <a name="packages.STORAGE_TYPE" href="#packages.STORAGE_TYPE">`packages.STORAGE_TYPE`</a>:
  ```ini
  STORAGE_TYPE = local
  ```

- <a name="packages.MINIO_BASE_PATH" href="#packages.MINIO_BASE_PATH">`packages.MINIO_BASE_PATH`</a>:
  override the minio base path if storage type is minio:
  ```ini
  MINIO_BASE_PATH = packages/
  ```

- <a name="packages.CHUNKED_UPLOAD_PATH" href="#packages.CHUNKED_UPLOAD_PATH">`packages.CHUNKED_UPLOAD_PATH`</a>:
  Path for chunked uploads. Defaults to APP_DATA_PATH + `tmp/package-upload`:
  ```ini
  CHUNKED_UPLOAD_PATH = tmp/package-upload
  ```

- <a name="packages.LIMIT_TOTAL_OWNER_COUNT" href="#packages.LIMIT_TOTAL_OWNER_COUNT">`packages.LIMIT_TOTAL_OWNER_COUNT`</a>:
  Maximum count of package versions a single owner can have (`-1` means no limits):
  ```ini
  LIMIT_TOTAL_OWNER_COUNT = -1
  ```

- <a name="packages.LIMIT_TOTAL_OWNER_SIZE" href="#packages.LIMIT_TOTAL_OWNER_SIZE">`packages.LIMIT_TOTAL_OWNER_SIZE`</a>:
  Maximum size of packages a single owner can use (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_TOTAL_OWNER_SIZE = -1
  ```

- <a name="packages.LIMIT_SIZE_ALPINE" href="#packages.LIMIT_SIZE_ALPINE">`packages.LIMIT_SIZE_ALPINE`</a>:
  Maximum size of an Alpine upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_ALPINE = -1
  ```

- <a name="packages.LIMIT_SIZE_CARGO" href="#packages.LIMIT_SIZE_CARGO">`packages.LIMIT_SIZE_CARGO`</a>:
  Maximum size of a Cargo upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_CARGO = -1
  ```

- <a name="packages.LIMIT_SIZE_CHEF" href="#packages.LIMIT_SIZE_CHEF">`packages.LIMIT_SIZE_CHEF`</a>:
  Maximum size of a Chef upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_CHEF = -1
  ```

- <a name="packages.LIMIT_SIZE_COMPOSER" href="#packages.LIMIT_SIZE_COMPOSER">`packages.LIMIT_SIZE_COMPOSER`</a>:
  Maximum size of a Composer upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_COMPOSER = -1
  ```

- <a name="packages.LIMIT_SIZE_CONAN" href="#packages.LIMIT_SIZE_CONAN">`packages.LIMIT_SIZE_CONAN`</a>:
  Maximum size of a Conan upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_CONAN = -1
  ```

- <a name="packages.LIMIT_SIZE_CONDA" href="#packages.LIMIT_SIZE_CONDA">`packages.LIMIT_SIZE_CONDA`</a>:
  Maximum size of a Conda upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_CONDA = -1
  ```

- <a name="packages.LIMIT_SIZE_CONTAINER" href="#packages.LIMIT_SIZE_CONTAINER">`packages.LIMIT_SIZE_CONTAINER`</a>:
  Maximum size of a Container upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_CONTAINER = -1
  ```

- <a name="packages.LIMIT_SIZE_CRAN" href="#packages.LIMIT_SIZE_CRAN">`packages.LIMIT_SIZE_CRAN`</a>:
  Maximum size of a CRAN upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_CRAN = -1
  ```

- <a name="packages.LIMIT_SIZE_DEBIAN" href="#packages.LIMIT_SIZE_DEBIAN">`packages.LIMIT_SIZE_DEBIAN`</a>:
  Maximum size of a Debian upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_DEBIAN = -1
  ```

- <a name="packages.LIMIT_SIZE_GENERIC" href="#packages.LIMIT_SIZE_GENERIC">`packages.LIMIT_SIZE_GENERIC`</a>:
  Maximum size of a Generic upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_GENERIC = -1
  ```

- <a name="packages.LIMIT_SIZE_GO" href="#packages.LIMIT_SIZE_GO">`packages.LIMIT_SIZE_GO`</a>:
  Maximum size of a Go upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_GO = -1
  ```

- <a name="packages.LIMIT_SIZE_HELM" href="#packages.LIMIT_SIZE_HELM">`packages.LIMIT_SIZE_HELM`</a>:
  Maximum size of a Helm upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_HELM = -1
  ```

- <a name="packages.LIMIT_SIZE_MAVEN" href="#packages.LIMIT_SIZE_MAVEN">`packages.LIMIT_SIZE_MAVEN`</a>:
  Maximum size of a Maven upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_MAVEN = -1
  ```

- <a name="packages.LIMIT_SIZE_NPM" href="#packages.LIMIT_SIZE_NPM">`packages.LIMIT_SIZE_NPM`</a>:
  Maximum size of a npm upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_NPM = -1
  ```

- <a name="packages.LIMIT_SIZE_NUGET" href="#packages.LIMIT_SIZE_NUGET">`packages.LIMIT_SIZE_NUGET`</a>:
  Maximum size of a NuGet upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_NUGET = -1
  ```

- <a name="packages.LIMIT_SIZE_PUB" href="#packages.LIMIT_SIZE_PUB">`packages.LIMIT_SIZE_PUB`</a>:
  Maximum size of a Pub upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_PUB = -1
  ```

- <a name="packages.LIMIT_SIZE_PYPI" href="#packages.LIMIT_SIZE_PYPI">`packages.LIMIT_SIZE_PYPI`</a>:
  Maximum size of a PyPI upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_PYPI = -1
  ```

- <a name="packages.LIMIT_SIZE_RPM" href="#packages.LIMIT_SIZE_RPM">`packages.LIMIT_SIZE_RPM`</a>:
  Maximum size of a RPM upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_RPM = -1
  ```

- <a name="packages.LIMIT_SIZE_RUBYGEMS" href="#packages.LIMIT_SIZE_RUBYGEMS">`packages.LIMIT_SIZE_RUBYGEMS`</a>:
  Maximum size of a RubyGems upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_RUBYGEMS = -1
  ```

- <a name="packages.LIMIT_SIZE_SWIFT" href="#packages.LIMIT_SIZE_SWIFT">`packages.LIMIT_SIZE_SWIFT`</a>:
  Maximum size of a Swift upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_SWIFT = -1
  ```

- <a name="packages.LIMIT_SIZE_VAGRANT" href="#packages.LIMIT_SIZE_VAGRANT">`packages.LIMIT_SIZE_VAGRANT`</a>:
  Maximum size of a Vagrant upload (`-1` means no limits, format `1000`, `1 MB`, `1 GiB`):
  ```ini
  LIMIT_SIZE_VAGRANT = -1
  ```

- <a name="packages.DEFAULT_RPM_SIGN_ENABLED" href="#packages.DEFAULT_RPM_SIGN_ENABLED">`packages.DEFAULT_RPM_SIGN_ENABLED`</a>:
  Enable RPM re-signing by default. (It will overwrite the old signature ,using v4 format, not compatible with CentOS 6 or older):
  ```ini
  DEFAULT_RPM_SIGN_ENABLED = false
  ```

## <a name="storage" href="#storage">Storage</a>
default storage for attachments, lfs and avatars

```ini
[storage]
```

- <a name="storage.STORAGE_TYPE" href="#storage.STORAGE_TYPE">`storage.STORAGE_TYPE`</a>:
  storage type:
  ```ini
  STORAGE_TYPE = local
  ```

### <a name="storage.repo-archive" href="#storage.repo-archive">Storage repo archive</a>
settings for repository archives, will override storage setting

```ini
[storage.repo-archive]
```

- <a name="storage.repo-archive.STORAGE_TYPE" href="#storage.repo-archive.STORAGE_TYPE">`storage.repo-archive.STORAGE_TYPE`</a>:
  storage type:
  ```ini
  STORAGE_TYPE = local
  ```

### <a name="storage.packages" href="#storage.packages">Storage packages</a>
settings for packages, will override storage setting

```ini
[storage.packages]
```

- <a name="storage.packages.STORAGE_TYPE" href="#storage.packages.STORAGE_TYPE">`storage.packages.STORAGE_TYPE`</a>:
  storage type:
  ```ini
  STORAGE_TYPE = local
  ```

### <a name="storage.my_minio" href="#storage.my_minio">Storage my minio</a>
customize storage

```ini
[storage.my_minio]
```

- <a name="storage.my_minio.STORAGE_TYPE" href="#storage.my_minio.STORAGE_TYPE">`storage.my_minio.STORAGE_TYPE`</a>:
  ```ini
  STORAGE_TYPE = minio
  ```

- <a name="storage.my_minio.MINIO_ENDPOINT" href="#storage.my_minio.MINIO_ENDPOINT">`storage.my_minio.MINIO_ENDPOINT`</a>:
  Minio endpoint to connect only available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_ENDPOINT = localhost:9000
  ```

- <a name="storage.my_minio.MINIO_ACCESS_KEY_ID" href="#storage.my_minio.MINIO_ACCESS_KEY_ID">`storage.my_minio.MINIO_ACCESS_KEY_ID`</a>:
  Minio accessKeyID to connect only available when STORAGE_TYPE is `minio`.
  If not provided and STORAGE_TYPE is `minio`, will search for credentials in known
  environment variables (MINIO_ACCESS_KEY_ID, AWS_ACCESS_KEY_ID), credentials files
  (~/.mc/config.json, ~/.aws/credentials), and EC2 instance metadata:
  ```ini
  MINIO_ACCESS_KEY_ID =
  ```

- <a name="storage.my_minio.MINIO_SECRET_ACCESS_KEY" href="#storage.my_minio.MINIO_SECRET_ACCESS_KEY">`storage.my_minio.MINIO_SECRET_ACCESS_KEY`</a>:
  Minio secretAccessKey to connect only available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_SECRET_ACCESS_KEY =
  ```

- <a name="storage.my_minio.MINIO_BUCKET" href="#storage.my_minio.MINIO_BUCKET">`storage.my_minio.MINIO_BUCKET`</a>:
  Minio bucket to store the attachments only available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_BUCKET = gitea
  ```

- <a name="storage.my_minio.MINIO_BUCKET_LOOKUP" href="#storage.my_minio.MINIO_BUCKET_LOOKUP">`storage.my_minio.MINIO_BUCKET_LOOKUP`</a>:
  Url lookup for the minio bucket only available when STORAGE_TYPE is `minio`
  Available values: auto, dns, path
  If empty, it behaves the same as "auto" was set:
  ```ini
  MINIO_BUCKET_LOOKUP =
  ```

- <a name="storage.my_minio.MINIO_LOCATION" href="#storage.my_minio.MINIO_LOCATION">`storage.my_minio.MINIO_LOCATION`</a>:
  Minio location to create bucket only available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_LOCATION = us-east-1
  ```

- <a name="storage.my_minio.MINIO_USE_SSL" href="#storage.my_minio.MINIO_USE_SSL">`storage.my_minio.MINIO_USE_SSL`</a>:
  Minio enabled ssl only available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_USE_SSL = false
  ```

- <a name="storage.my_minio.MINIO_INSECURE_SKIP_VERIFY" href="#storage.my_minio.MINIO_INSECURE_SKIP_VERIFY">`storage.my_minio.MINIO_INSECURE_SKIP_VERIFY`</a>:
  Minio skip SSL verification available when STORAGE_TYPE is `minio`:
  ```ini
  MINIO_INSECURE_SKIP_VERIFY = false
  ```

### <a name="storage.actions_log" href="#storage.actions_log">Storage actions log</a>
settings for action logs, will override storage setting

```ini
[storage.actions_log]
```

- <a name="storage.actions_log.STORAGE_TYPE" href="#storage.actions_log.STORAGE_TYPE">`storage.actions_log.STORAGE_TYPE`</a>:
  storage type:
  ```ini
  STORAGE_TYPE = local
  ```

### <a name="storage.actions_artifacts" href="#storage.actions_artifacts">Storage actions artifacts</a>
settings for action artifacts, will override storage setting

```ini
[storage.actions_artifacts]
```

- <a name="storage.actions_artifacts.STORAGE_TYPE" href="#storage.actions_artifacts.STORAGE_TYPE">`storage.actions_artifacts.STORAGE_TYPE`</a>:
  storage type:
  ```ini
  STORAGE_TYPE = local
  ```

## <a name="repo-archive" href="#repo-archive">Repo archive</a>
repo-archive storage will override storage

```ini
[repo-archive]
```

- <a name="repo-archive.STORAGE_TYPE" href="#repo-archive.STORAGE_TYPE">`repo-archive.STORAGE_TYPE`</a>:
  ```ini
  STORAGE_TYPE = local
  ```

- <a name="repo-archive.PATH" href="#repo-archive.PATH">`repo-archive.PATH`</a>:
  Where your lfs files reside, default is data/lfs:
  ```ini
  PATH = data/repo-archive
  ```

- <a name="repo-archive.MINIO_BASE_PATH" href="#repo-archive.MINIO_BASE_PATH">`repo-archive.MINIO_BASE_PATH`</a>:
  override the minio base path if storage type is minio:
  ```ini
  MINIO_BASE_PATH = repo-archive/
  ```

## <a name="lfs" href="#lfs">Lfs</a>
lfs storage will override storage

```ini
[lfs]
```

- <a name="lfs.STORAGE_TYPE" href="#lfs.STORAGE_TYPE">`lfs.STORAGE_TYPE`</a>:
  ```ini
  STORAGE_TYPE = local
  ```

- <a name="lfs.PATH" href="#lfs.PATH">`lfs.PATH`</a>:
  Where your lfs files reside, default is data/lfs:
  ```ini
  PATH = data/lfs
  ```

- <a name="lfs.MINIO_BASE_PATH" href="#lfs.MINIO_BASE_PATH">`lfs.MINIO_BASE_PATH`</a>:
  override the minio base path if storage type is minio:
  ```ini
  MINIO_BASE_PATH = lfs/
  ```

## <a name="lfs_client" href="#lfs_client">Lfs client</a>
settings for Forgejo's LFS client (eg: mirroring an upstream lfs endpoint)

```ini
[lfs_client]
```

- <a name="lfs_client.BATCH_SIZE" href="#lfs_client.BATCH_SIZE">`lfs_client.BATCH_SIZE`</a>:
  Limit the number of pointers in each batch request to this number:
  ```ini
  BATCH_SIZE = 20
  ```

- <a name="lfs_client.BATCH_OPERATION_CONCURRENCY" href="#lfs_client.BATCH_OPERATION_CONCURRENCY">`lfs_client.BATCH_OPERATION_CONCURRENCY`</a>:
  Limit the number of concurrent upload/download operations within a batch:
  ```ini
  BATCH_OPERATION_CONCURRENCY = 8
  ```

## <a name="proxy" href="#proxy">Proxy</a>

```ini
[proxy]
```

- <a name="proxy.PROXY_ENABLED" href="#proxy.PROXY_ENABLED">`proxy.PROXY_ENABLED`</a>:
  Enable the proxy, all requests to external via HTTP will be affected:
  ```ini
  PROXY_ENABLED = false
  ```

- <a name="proxy.PROXY_URL" href="#proxy.PROXY_URL">`proxy.PROXY_URL`</a>:
  Proxy server URL, support http://, https//, socks://, blank will follow environment http_proxy/https_proxy/no_proxy:
  ```ini
  PROXY_URL =
  ```

- <a name="proxy.PROXY_HOSTS" href="#proxy.PROXY_HOSTS">`proxy.PROXY_HOSTS`</a>:
  Comma separated list of host names requiring proxy. Glob patterns (*) are accepted; use ** to match all hosts:
  ```ini
  PROXY_HOSTS =
  ```

## <a name="actions" href="#actions">Actions</a>

```ini
[actions]
```

- <a name="actions.ENABLED" href="#actions.ENABLED">`actions.ENABLED`</a>:
  Enable/Disable actions capabilities:
  ```ini
  ENABLED = true
  ```

- <a name="actions.DEFAULT_ACTIONS_URL" href="#actions.DEFAULT_ACTIONS_URL">`actions.DEFAULT_ACTIONS_URL`</a>:
  Default address to get action plugins, e.g. the default value means downloading from "https://code.forgejo.org/actions/checkout" for "uses: actions/checkout@v3":
  ```ini
  DEFAULT_ACTIONS_URL = https://code.forgejo.org
  ```

- <a name="actions.LOG_RETENTION_DAYS" href="#actions.LOG_RETENTION_DAYS">`actions.LOG_RETENTION_DAYS`</a>:
  Logs retention time in days. Old logs will be deleted after this period:
  ```ini
  LOG_RETENTION_DAYS = 365
  ```

- <a name="actions.LOG_COMPRESSION" href="#actions.LOG_COMPRESSION">`actions.LOG_COMPRESSION`</a>:
  Log compression type, `none` for no compression, `zstd` for zstd compression.
  Other compression types like `gzip` are NOT supported, since seekable stream is required for log view.
  It's always recommended to use compression when using local disk as log storage if CPU or memory is not a bottleneck.
  And for object storage services like S3, which is billed for requests, it would cause extra 2 times of get requests for each log view.
  But it will save storage space and network bandwidth, so it's still recommended to use compression:
  ```ini
  LOG_COMPRESSION = zstd
  ```

- <a name="actions.ARTIFACT_RETENTION_DAYS" href="#actions.ARTIFACT_RETENTION_DAYS">`actions.ARTIFACT_RETENTION_DAYS`</a>:
  Default artifact retention time in days. Artifacts could have their own retention periods by setting the `retention-days` option in `actions/upload-artifact` step:
  ```ini
  ARTIFACT_RETENTION_DAYS = 90
  ```

- <a name="actions.ZOMBIE_TASK_TIMEOUT" href="#actions.ZOMBIE_TASK_TIMEOUT">`actions.ZOMBIE_TASK_TIMEOUT`</a>:
  Timeout to stop the task which have running status, but haven't been updated for a long time:
  ```ini
  ZOMBIE_TASK_TIMEOUT = 10m
  ```

- <a name="actions.ENDLESS_TASK_TIMEOUT" href="#actions.ENDLESS_TASK_TIMEOUT">`actions.ENDLESS_TASK_TIMEOUT`</a>:
  Timeout to stop the tasks which have running status and continuous updates, but don't end for a long time:
  ```ini
  ENDLESS_TASK_TIMEOUT = 3h
  ```

- <a name="actions.ABANDONED_JOB_TIMEOUT" href="#actions.ABANDONED_JOB_TIMEOUT">`actions.ABANDONED_JOB_TIMEOUT`</a>:
  Timeout to cancel the jobs which have waiting status, but haven't been picked by a runner for a long time:
  ```ini
  ABANDONED_JOB_TIMEOUT = 24h
  ```

- <a name="actions.SKIP_WORKFLOW_STRINGS" href="#actions.SKIP_WORKFLOW_STRINGS">`actions.SKIP_WORKFLOW_STRINGS`</a>:
  Strings committers can place inside a commit message or PR title to skip executing the corresponding actions workflow:
  ```ini
  SKIP_WORKFLOW_STRINGS = [skip ci],[ci skip],[no ci],[skip actions],[actions skip]
  ```

- <a name="actions.LIMIT_DISPATCH_INPUTS" href="#actions.LIMIT_DISPATCH_INPUTS">`actions.LIMIT_DISPATCH_INPUTS`</a>:
  Limit on inputs for manual / workflow_dispatch triggers, default is 10:
  ```ini
  LIMIT_DISPATCH_INPUTS = 10
  ```
