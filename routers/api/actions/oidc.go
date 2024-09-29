package actions

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rakutentech/jwk-go/jwk"
	"github.com/rakutentech/jwk-go/okp"
)

type oidcRoutes struct {
	ca                  ed25519.PrivateKey
	jwks                []*jwk.KeySpec
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

	// TODO: generate this once and store it across restarts. In the database I assume?
	caPublicKey, caPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	r := oidcRoutes{
		ca: caPrivateKey,
		jwks: []*jwk.KeySpec{ // https://token.actions.githubusercontent.com/.well-known/jwks
			jwk.NewSpec(okp.NewCurve25519(caPublicKey, caPrivateKey)),
		},
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

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"jti":                   uuid.New().String(),
		"sub":                   fmt.Sprintf("repo:%s:ref:%s", repo, task.Job.Run.Ref),
		"aud":                   "", // TODO: Allow customizing this in the query param
		"ref":                   task.Job.Run.Ref,
		"sha":                   task.Job.Run.CommitSHA,
		"repository":            repo,
		"repository_owner":      task.Job.Run.Repo.OwnerName,
		"repository_owner_id":   task.Job.Run.Repo.OwnerID,
		"run_id":                task.Job.RunID,
		"run_number":            0, // TODO: how do i check this?
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
		"workflow_sha":          "", // TODO: is this just a hash of the yaml? if so that's easy enough to calculate
		"job_workflow_ref":      fmt.Sprintf("%s/.forgejo/workflow/%s@%s", repo, task.Job.Run.WorkflowID, task.Job.Run.Ref),
		"job_workflow_sha":      "",                                    // TODO: is this just a hash of the yaml? if so that's easy enough to calculate
		"runner_environment":    "self-hosted",                         // not sure what this should be set to, github will have either "github-hosted" or "self-hosted"
		"iss":                   setting.AppURL + "/api/actions_token", // TODO: how do i check the public domain?
		"nbf":                   jwt.NewNumericDate(iat),
		"exp":                   jwt.NewNumericDate(iat.Add(time.Minute * 15)),
		"iat":                   jwt.NewNumericDate(iat),
	})

	signedJWT, err := token.SignedString(o.ca)
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
	err := json.NewEncoder(resp).Encode(o.jwks)
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
