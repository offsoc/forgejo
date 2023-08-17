// SPDX-License-Identifier: MIT

package driver

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	config_types "lab.forgefriends.org/friendlyforgeformat/gof3/config/types"
	f3_tests "lab.forgefriends.org/friendlyforgeformat/gof3/forges/tests"
)

func TestForgejoDriverMethods(t *testing.T) {
	unittest.PrepareTestEnv(t)
	TestForgeMethods(t)
}

func TestForgeMethods(t *testing.T) {
	testUsersProviderOptions := f3_tests.TestUsersProviderOptions
	testUsersProviderOptions.ModifiedPut = true

	testIssueProviderOptions := f3_tests.TestIssueProviderOptions
	testIssueProviderOptions.ModifiedPut = true

	testLabelProviderOptions := f3_tests.TestLabelProviderOptions
	testLabelProviderOptions.ModifiedPut = true

	testMilestonesProviderOptions := f3_tests.TestMilestonesProviderOptions
	testMilestonesProviderOptions.ModifiedPut = true

	testCommentProviderOptions := f3_tests.TestCommentProviderOptions
	testCommentProviderOptions.ModifiedPut = true

	for _, testCase := range []struct {
		name string
		fun  func(f3_tests.ForgeTestInterface, f3_tests.ProviderOptions)
		opts f3_tests.ProviderOptions
	}{
		{name: "asset", fun: f3_tests.TestAssets, opts: f3_tests.TestAssetsProviderOptions},
		{name: "repository", fun: f3_tests.TestRepository, opts: f3_tests.TestRepositoryProviderOptions},
		{name: "comment", fun: f3_tests.TestComment, opts: testCommentProviderOptions},
		{name: "issue", fun: f3_tests.TestIssue, opts: testIssueProviderOptions},
		{name: "label", fun: f3_tests.TestLabel, opts: testLabelProviderOptions},
		{name: "milestone", fun: f3_tests.TestMilestones, opts: testMilestonesProviderOptions},
		{name: "project", fun: f3_tests.TestProject, opts: f3_tests.TestProjectProviderOptions},
		{name: "user", fun: f3_tests.TestUsers, opts: testUsersProviderOptions},
		{name: "topic", fun: f3_tests.TestTopic, opts: f3_tests.TestTopicProviderOptions},
		{name: "pull_request", fun: f3_tests.TestPullRequests, opts: f3_tests.TestPullRequestsProviderOptions},
		{name: "release", fun: f3_tests.TestReleases, opts: f3_tests.TestReleasesProviderOptions},
		{name: "review", fun: f3_tests.TestReview, opts: f3_tests.TestReviewProviderOptions},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.fun(NewTestForgejo(t), testCase.opts)
		})
	}
}

type forgejoInstance struct {
	f3_tests.ForgeInstance
}

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", "..", ".."),
	})
}

func (o *forgejoInstance) Init(t f3_tests.TestingT) {
	g := &Forgejo{}
	o.ForgeInstance.Init(t, g)

	doer, err := user_model.GetAdminUser(o.GetCtx())
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %v", err))
	}

	options := &Options{
		Options: config_types.Options{
			Configuration: config_types.Configuration{
				Type: strings.ToLower(Name),
			},
			Features: config_types.AllFeatures,
			Logger:   ToF3Logger(nil),
		},
		Doer: doer,
	}
	options.SetDefaults()
	g.Init(options)
}

func NewTestForgejo(t f3_tests.TestingT) f3_tests.ForgeTestInterface {
	o := forgejoInstance{}
	o.Init(t)
	return &o
}
