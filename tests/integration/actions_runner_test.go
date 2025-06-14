// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/modules/setting"

	pingv1 "code.gitea.io/actions-proto-go/ping/v1"
	"code.gitea.io/actions-proto-go/ping/v1/pingv1connect"
	runnerv1 "code.gitea.io/actions-proto-go/runner/v1"
	"code.gitea.io/actions-proto-go/runner/v1/runnerv1connect"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockRunner struct {
	client *mockRunnerClient
}

type mockRunnerClient struct {
	pingServiceClient   pingv1connect.PingServiceClient
	runnerServiceClient runnerv1connect.RunnerServiceClient
}

func newMockRunner() *mockRunner {
	client := newMockRunnerClient("", "")
	return &mockRunner{client: client}
}

func newMockRunnerClient(uuid, token string) *mockRunnerClient {
	baseURL := fmt.Sprintf("%sapi/actions", setting.AppURL)

	opt := connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if uuid != "" {
				req.Header().Set("x-runner-uuid", uuid)
			}
			if token != "" {
				req.Header().Set("x-runner-token", token)
			}
			return next(ctx, req)
		}
	}))

	client := &mockRunnerClient{
		pingServiceClient:   pingv1connect.NewPingServiceClient(http.DefaultClient, baseURL, opt),
		runnerServiceClient: runnerv1connect.NewRunnerServiceClient(http.DefaultClient, baseURL, opt),
	}

	return client
}

func (r *mockRunner) doPing(t *testing.T) {
	resp, err := r.client.pingServiceClient.Ping(t.Context(), connect.NewRequest(&pingv1.PingRequest{
		Data: "mock-runner",
	}))
	require.NoError(t, err)
	require.Equal(t, "Hello, mock-runner!", resp.Msg.Data)
}

func (r *mockRunner) doRegister(t *testing.T, name, token string, labels []string) {
	r.doPing(t)
	resp, err := r.client.runnerServiceClient.Register(t.Context(), connect.NewRequest(&runnerv1.RegisterRequest{
		Name:    name,
		Token:   token,
		Version: "mock-runner-version",
		Labels:  labels,
	}))
	require.NoError(t, err)
	r.client = newMockRunnerClient(resp.Msg.Runner.Uuid, resp.Msg.Runner.Token)
}

func (r *mockRunner) registerAsRepoRunner(t *testing.T, ownerName, repoName, runnerName string, labels []string) {
	if !setting.Database.Type.IsSQLite3() {
		// registering a mock runner when using a database other than SQLite leaves leftovers
		t.FailNow()
	}
	session := loginUser(t, ownerName)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runners/registration-token", ownerName, repoName)).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var registrationToken struct {
		Token string `json:"token"`
	}
	DecodeJSON(t, resp, &registrationToken)
	r.doRegister(t, runnerName, registrationToken.Token, labels)
}

func (r *mockRunner) fetchTask(t *testing.T, timeout ...time.Duration) *runnerv1.Task {
	fetchTimeout := 10 * time.Second
	if len(timeout) > 0 {
		fetchTimeout = timeout[0]
	}

	var task *runnerv1.Task
	assert.Eventually(t, func() bool {
		resp, err := r.client.runnerServiceClient.FetchTask(t.Context(), connect.NewRequest(&runnerv1.FetchTaskRequest{
			TasksVersion: 0,
		}))
		require.NoError(t, err)
		if resp.Msg.Task != nil {
			task = resp.Msg.Task
			return true
		}
		return false
	}, fetchTimeout, time.Millisecond*100, "failed to fetch a task")
	return task
}

type mockTaskOutcome struct {
	result  runnerv1.Result
	outputs map[string]string
	logRows []*runnerv1.LogRow
}

func (r *mockRunner) execTask(t *testing.T, task *runnerv1.Task, outcome *mockTaskOutcome) {
	for idx, lr := range outcome.logRows {
		resp, err := r.client.runnerServiceClient.UpdateLog(t.Context(), connect.NewRequest(&runnerv1.UpdateLogRequest{
			TaskId: task.Id,
			Index:  int64(idx),
			Rows:   []*runnerv1.LogRow{lr},
			NoMore: idx == len(outcome.logRows)-1,
		}))
		require.NoError(t, err)
		assert.EqualValues(t, idx+1, resp.Msg.AckIndex)
	}
	sentOutputKeys := make([]string, 0, len(outcome.outputs))
	for outputKey, outputValue := range outcome.outputs {
		resp, err := r.client.runnerServiceClient.UpdateTask(t.Context(), connect.NewRequest(&runnerv1.UpdateTaskRequest{
			State: &runnerv1.TaskState{
				Id:     task.Id,
				Result: runnerv1.Result_RESULT_UNSPECIFIED,
			},
			Outputs: map[string]string{outputKey: outputValue},
		}))
		require.NoError(t, err)
		sentOutputKeys = append(sentOutputKeys, outputKey)
		assert.ElementsMatch(t, sentOutputKeys, resp.Msg.SentOutputs)
	}
	resp, err := r.client.runnerServiceClient.UpdateTask(t.Context(), connect.NewRequest(&runnerv1.UpdateTaskRequest{
		State: &runnerv1.TaskState{
			Id:        task.Id,
			Result:    outcome.result,
			StoppedAt: timestamppb.Now(),
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, outcome.result, resp.Msg.State.Result)
}

// Simply pretend we're running the task and succeed at that.
// We're that great!
func (r *mockRunner) succeedAtTask(t *testing.T, task *runnerv1.Task) {
	resp, err := r.client.runnerServiceClient.UpdateTask(t.Context(), connect.NewRequest(&runnerv1.UpdateTaskRequest{
		State: &runnerv1.TaskState{
			Id:        task.Id,
			Result:    runnerv1.Result_RESULT_SUCCESS,
			StoppedAt: timestamppb.Now(),
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, runnerv1.Result_RESULT_SUCCESS, resp.Msg.State.Result)
}

// Pretend we're running the task, do nothing and fail at that.
func (r *mockRunner) failAtTask(t *testing.T, task *runnerv1.Task) {
	resp, err := r.client.runnerServiceClient.UpdateTask(t.Context(), connect.NewRequest(&runnerv1.UpdateTaskRequest{
		State: &runnerv1.TaskState{
			Id:        task.Id,
			Result:    runnerv1.Result_RESULT_FAILURE,
			StoppedAt: timestamppb.Now(),
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, runnerv1.Result_RESULT_FAILURE, resp.Msg.State.Result)
}
