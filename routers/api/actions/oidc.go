package actions

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type oidcRoutes struct {
	ca ed25519.PrivateKey
}

func OIDCRoutes() *web.Route {
	m := web.NewRoute()
	m.Use(ArtifactContexter())

	_, caPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	r := oidcRoutes{
		ca: caPrivateKey,
	}

	m.Get("token", r.getToken)
	m.Get(".well-known/jwks.json", r.jwks)

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
	task, runID, ok := validateRunID(ctx)
	if !ok {
		return
	}

	// there's probably better ways get some of these values
	repo := fmt.Sprintf("%s/%s", task.Job.Run.Repo.OwnerName, task.Job.Run.Repo.Name)
	repositoryVisibility := "public"
	if task.Job.Run.Repo.IsPrivate { // are there options other than public and private?
		repositoryVisibility = "private"
	}
	iat := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, jwt.MapClaims{
		"jti":                   uuid.New().String(),
		"sub":                   fmt.Sprintf("repo:%s:ref:%s", repo, task.Job.Run.Ref),
		"aud":                   "TODO: Allow customizing this in the query param",
		"ref":                   task.Job.Run.Ref,
		"sha":                   task.Job.Run.CommitSHA,
		"repository":            repo,
		"repository_owner":      task.Job.Run.Repo.OwnerName,
		"repository_owner_id":   task.Job.Run.Repo.OwnerID,
		"run_id":                runID,
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
		"runner_environment":    "self-hosted",                         // not sure what this should be set to
		"iss":                   "https://git.example.org/api/actions", // TODO: how do i check the public domain?
		"nbf":                   iat,
		"exp":                   iat.Add(time.Hour), // TODO: should this be customizable?
		"iat":                   iat,
	})

	signedJWT, err := token.SignedString(o.ca)
	if err != nil {
		log.Error("Error signing JWT: %v", err)
		ctx.Error(http.StatusInternalServerError, "Error signing JWT")
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"count": 0, // TODO: unclear what this is, github gave me a value of 1857
		"token": signedJWT,
	})
}

func (o oidcRoutes) jwks(ctx *ArtifactContext) {

}
