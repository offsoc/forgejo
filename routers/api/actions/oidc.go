package actions

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/jwtx"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type oidcRoutes struct {
	signingKey          jwtx.JWTSigningKey
	openIDConfiguration openIDConfiguration
}

type openIDConfiguration struct {
	Issuer                           string   `json:"issuer"`
	JwksURI                          string   `json:"jwks_uri"`
	SubjecTypesSupported             []string `json:"subject_types_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
}

func OIDCRoutes(prefix string) *web.Route {
	m := web.NewRoute()

	prefix = strings.TrimPrefix(prefix, "/")

	rawKey, err := jwtx.LoadOrCreateAsymmetricKey(setting.Actions.JWTSigningPrivateKeyFile, setting.Actions.JWTSigningAlgorithm)
	if err != nil {
		log.Fatal("error loading jwt: %v", err)
	}

	key, err := jwtx.CreateJWTSigningKey(setting.Actions.JWTSigningAlgorithm, rawKey)
	if err != nil {
		log.Fatal("error parsing jwt: %v", err)
	}

	r := oidcRoutes{
		signingKey: key,
		openIDConfiguration: openIDConfiguration{
			Issuer:                 setting.AppURL + setting.AppSubURL + prefix,                       // TODO: how do i check the public domain?
			JwksURI:                setting.AppURL + setting.AppSubURL + prefix + "/.well-known/jwks", // TODO: how do i check the public domain?
			SubjecTypesSupported:   []string{"public", "pairwise"},
			ResponseTypesSupported: []string{"id_token"},
			ClaimsSupported: []string{
				"sub", "aud", "exp", "iat", "iss", "jti", "nbf", "ref", "sha", "repository", "repository_id",
				"repository_owner", "repository_owner_id", "enterprise", "enterprise_id", "run_id",
				"run_number", "run_attempt", "actor", "actor_id", "workflow", "workflow_ref", "workflow_sha",
				"head_ref", "base_ref", "event_name", "ref_type", "ref_protected", "environment",
				"environment_node_id", "job_workflow_ref", "job_workflow_sha", "repository_visibility",
				"runner_environment", "issuer_scope",
			},
			IDTokenSigningAlgValuesSupported: []string{"RS256"},
			ScopesSupported:                  []string{"openid"},
		},
	}
	m.Get("", ArtifactContexter(), r.getToken)
	m.Get("/.well-known/jwks", r.getJWKS)
	m.Get("/.well-known/openid-configuration", r.getOpenIDConfiguration)

	return m
}

// sample JWT github gave me:
//
//	{
//		"jti": "18b3eacd-6330-47e2-a113-604effd4cf91",
//		"sub": "repo:thefinn93/actions-test:ref:refs/heads/main",
//		"aud": "CUSTOM_AUDIENCE",
//		"ref": "refs/heads/main",
//		"sha": "5d92aa8ec38679dbd7b90eee4ea8dfa342e2c6e3",
//		"repository": "thefinn93/actions-test",
//		"repository_owner": "thefinn93",
//		"repository_owner_id": "692970",
//		"run_id": "8414792498",
//		"run_number": "2",
//		"run_attempt": "1",
//		"repository_visibility": "public",
//		"repository_id": "777026467",
//		"actor_id": "692970",
//		"actor": "thefinn93",
//		"workflow": ".github/workflows/test.yaml",
//		"head_ref": "",
//		"base_ref": "",
//		"event_name": "push",
//		"ref_protected": "false",
//		"ref_type": "branch",
//		"workflow_ref": "thefinn93/actions-test/.github/workflows/test.yaml@refs/heads/main",
//		"workflow_sha": "5d92aa8ec38679dbd7b90eee4ea8dfa342e2c6e3",
//		"job_workflow_ref": "thefinn93/actions-test/.github/workflows/test.yaml@refs/heads/main",
//		"job_workflow_sha": "5d92aa8ec38679dbd7b90eee4ea8dfa342e2c6e3",
//		"runner_environment": "github-hosted",
//		"iss": "https://token.actions.githubusercontent.com",
//		"nbf": 1711337835,
//		"exp": 1711338735,
//		"iat": 1711338435
//	  }
func (o oidcRoutes) getToken(ctx *ArtifactContext) {
	task := ctx.ActionTask

	aud := ctx.Req.URL.Query().Get("audience")

	if err := task.Job.LoadRun(ctx); err != nil {
		log.Error("Error loading run: %v", err)
		ctx.Error(http.StatusInternalServerError, "Error loading run")
		return
	}

	if err := task.Job.Run.LoadAttributes(ctx); err != nil {
		log.Error("Error loading attributes: %v", err)
		ctx.Error(http.StatusInternalServerError, "Error loading attributes")
		return
	}

	// there's probably better ways get some of these values
	repo := fmt.Sprintf("%s/%s", task.Job.Run.Repo.OwnerName, task.Job.Run.Repo.Name)
	repositoryVisibility := "public"
	if task.Job.Run.Repo.IsPrivate { // are there options other than public and private?
		repositoryVisibility = "private"
	}
	iat := time.Now()

	token := jwt.NewWithClaims(jwt.GetSigningMethod(setting.Actions.JWTSigningAlgorithm), jwt.MapClaims{
		"jti":                   uuid.New().String(),
		"sub":                   fmt.Sprintf("repo:%s:ref:%s", repo, task.Job.Run.Ref),
		"aud":                   aud,
		"ref":                   task.Job.Run.Ref,
		"sha":                   task.Job.Run.CommitSHA,
		"repository":            repo,
		"repository_owner":      task.Job.Run.Repo.OwnerName,
		"repository_owner_id":   task.Job.Run.Repo.OwnerID,
		"run_id":                task.Job.RunID,
		"run_number":            task.Job.Run.Index,
		"run_attempt":           0, // TODO: how do i check this?
		"repository_visibility": repositoryVisibility,
		"repository_id":         task.Job.Run.Repo.ID,
		"actor_id":              task.Job.Run.TriggerUserID,
		"actor":                 task.Job.Run.TriggerUser.Name,
		"workflow":              fmt.Sprintf(".forgejo/workflow/%s", task.Job.Run.WorkflowID), // TODO: remove hard-coded prefix, fetch it from wherever that data is stored
		"head_ref":              "",                                                           // this is empty in my GH test. Maybe we should take it out here?
		"base_ref":              "",                                                           // this is empty in my GH test. Maybe we should take it out here?
		"event_name":            task.Job.Run.TriggerEvent,
		"ref_protected":         false,    // TODO: how do i check this?
		"ref_type":              "branch", // TODO: how do i check this?
		"workflow_ref":          fmt.Sprintf("%s/.forgejo/workflow/%s@%s", repo, task.Job.Run.WorkflowID, task.Job.Run.Ref),
		"workflow_sha":          task.Job.Run.CommitSHA,
		"job_workflow_ref":      fmt.Sprintf("%s/.forgejo/workflow/%s@%s", repo, task.Job.Run.WorkflowID, task.Job.Run.Ref),
		"job_workflow_sha":      task.Job.Run.CommitSHA,
		"runner_environment":    "self-hosted", // not sure what this should be set to, github will have either "github-hosted" or "self-hosted"
		"iss":                   setting.AppURL + setting.AppSubURL + "/api/actions_token",
		"nbf":                   jwt.NewNumericDate(iat),
		"exp":                   jwt.NewNumericDate(iat.Add(time.Minute * 15)),
		"iat":                   jwt.NewNumericDate(iat),
	}, addTokenHeaders(o.signingKey))

	signedJWT, err := token.SignedString(o.signingKey.SignKey())
	if err != nil {
		log.Error("Error signing JWT: %v", err)
		ctx.Error(http.StatusInternalServerError, "Error signing JWT")
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"count": 0, // TODO: unclear what this is, github gave me a value of 1857
		"value": signedJWT,
	})
}

func (o oidcRoutes) getJWKS(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")

	jwk, err := o.signingKey.ToJWK()
	if err != nil {
		log.Error("Error converting signing key to JWK: %v", err)
		http.Error(resp, "error converting signing key to JWT", http.StatusInternalServerError)
		return
	}

	jwk["use"] = "sig"

	jwks := map[string][]map[string]string{
		"keys": {
			jwk,
		},
	}

	err = json.NewEncoder(resp).Encode(jwks)
	if err != nil {
		log.Error("error encoding jwks response: ", err)
		http.Error(resp, "error encoding jwks response", http.StatusInternalServerError)
		return
	}
}

func (o oidcRoutes) getOpenIDConfiguration(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(resp).Encode(o.openIDConfiguration)
	if err != nil {
		log.Error("error encoding jwks response: ", err)
		http.Error(resp, "error encoding jwks response", http.StatusInternalServerError)
		return
	}
}

func addTokenHeaders(key jwtx.JWTSigningKey) jwt.TokenOption {
	return func(t *jwt.Token) {
		kid := key.KID()
		if kid != "" {
			t.Header["kid"] = kid
		}
	}
}
