// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/models/auth"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/assert"
)

const bsd2ClauseLicense = `
Copyright (c) 2024 Gitea

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
`

const bsd3ClauseLicense = `
Copyright (c) 2024 Forgejo

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS “AS IS” AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
`

func TestAPIRepoLicense(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

		session := loginUser(t, user.Name)

		createOrReplaceFileInBranch(user, repo, "LICENSE", repo.DefaultBranch, bsd2ClauseLicense)

		// let gitea update repo license
		time.Sleep(time.Second)
		checkRepoLicense(t, "user2", "repo1", []*api.RepoLicense{&api.RepoLicense{Name: "BSD-2-Clause", Path: "LICENSE"}})

		createOrReplaceFileInBranch(user, repo, "COPYING", repo.DefaultBranch, bsd3ClauseLicense)

		// let gitea update repo license
		time.Sleep(time.Second)
		checkRepoLicense(t, "user2", "repo1", []*api.RepoLicense{&api.RepoLicense{Name: "BSD-2-Clause", Path: "LICENSE"}, &api.RepoLicense{Name: "BSD-3-Clause", Path: "COPYING"}})

		deleteFileInBranch(user, repo, "LICENSE", repo.DefaultBranch)

		// let gitea update repo license
		time.Sleep(time.Second)
		checkRepoLicense(t, "user2", "repo1", []*api.RepoLicense{&api.RepoLicense{Name: "BSD-3-Clause", Path: "COPYING"}})

		// Change default branch
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		branchName := "DefaultBranch"
		req := NewRequestWithJSON(t, "PATCH", "/api/v1/repos/user2/repo1", api.EditRepoOption{
			DefaultBranch: &branchName,
		}).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusOK)

		// let gitea update repo license
		time.Sleep(time.Second)
		checkRepoLicense(t, "user2", "repo1", []*api.RepoLicense{&api.RepoLicense{Name: "MIT", Path: "LICENSE"}})
	})
}

func checkRepoLicense(t *testing.T, owner, repo string, expected []*api.RepoLicense) {
	t.Helper()

	reqURL := fmt.Sprintf("/api/v1/repos/%s/%s/licenses", owner, repo)
	req := NewRequest(t, "GET", reqURL)
	resp := MakeRequest(t, req, http.StatusOK)

	var licenses api.RepoLicenseList
	DecodeJSON(t, resp, &licenses)

	assert.ElementsMatch(t, expected, licenses.Licenses, 0)
}
