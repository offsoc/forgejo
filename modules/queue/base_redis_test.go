// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package queue

import (
	"context"
	"testing"

	"forgejo.org/modules/queue/mock"
	"forgejo.org/modules/setting"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type baseRedisUnitTestSuite struct {
	suite.Suite

	mockController *gomock.Controller
}

func TestBaseRedis(t *testing.T) {
	suite.Run(t, &baseRedisUnitTestSuite{})
}

func (suite *baseRedisUnitTestSuite) SetupSuite() {
	suite.mockController = gomock.NewController(suite.T())
}

func (suite *baseRedisUnitTestSuite) TestBasic() {
	queueName := "test-queue"
	testCases := []struct {
		Name             string
		ConnectionString string
		QueueName        string
		Unique           bool
	}{
		{
			Name:             "unique",
			ConnectionString: "redis://127.0.0.1/0",
			QueueName:        queueName,
			Unique:           true,
		},
		{
			Name:             "non-unique",
			ConnectionString: "redis://127.0.0.1/0",
			QueueName:        queueName,
			Unique:           false,
		},
		{
			Name:             "unique with prefix",
			ConnectionString: "redis://127.0.0.1/0?prefix=forgejo:queue:",
			QueueName:        "forgejo:queue:" + queueName,
			Unique:           true,
		},
		{
			Name:             "non-unique with prefix",
			ConnectionString: "redis://127.0.0.1/0?prefix=forgejo:queue:",
			QueueName:        "forgejo:queue:" + queueName,
			Unique:           false,
		},
	}

	for _, testCase := range testCases {
		suite.Run(testCase.Name, func() {
			queueSettings := setting.QueueSettings{
				Length:  10,
				ConnStr: testCase.ConnectionString,
			}

			// Configure expectations.
			mockRedisStore := mock.NewInMemoryMockRedis()
			redisClient := mock.NewMockRedisClient(suite.mockController)

			redisClient.EXPECT().
				Ping(gomock.Any()).
				Times(1).
				Return(&redis.StatusCmd{})
			redisClient.EXPECT().
				LLen(gomock.Any(), testCase.QueueName).
				Times(1).
				DoAndReturn(mockRedisStore.LLen)
			redisClient.EXPECT().
				LPop(gomock.Any(), testCase.QueueName).
				Times(1).
				DoAndReturn(mockRedisStore.LPop)
			redisClient.EXPECT().
				RPush(gomock.Any(), testCase.QueueName, gomock.Any()).
				Times(1).
				DoAndReturn(mockRedisStore.RPush)

			if testCase.Unique {
				redisClient.EXPECT().
					SAdd(gomock.Any(), testCase.QueueName+"_unique", gomock.Any()).
					Times(1).
					DoAndReturn(mockRedisStore.SAdd)
				redisClient.EXPECT().
					SRem(gomock.Any(), testCase.QueueName+"_unique", gomock.Any()).
					Times(1).
					DoAndReturn(mockRedisStore.SRem)
				redisClient.EXPECT().
					SIsMember(gomock.Any(), testCase.QueueName+"_unique", gomock.Any()).
					Times(2).
					DoAndReturn(mockRedisStore.SIsMember)
			}

			client, err := newBaseRedisGeneric(
				toBaseConfig(queueName, queueSettings),
				testCase.Unique,
				redisClient,
			)
			suite.Require().NoError(err)

			ctx := context.Background()
			expectedContent := []byte("test")

			suite.Require().NoError(client.PushItem(ctx, expectedContent))

			found, err := client.HasItem(ctx, expectedContent)
			suite.Require().NoError(err)
			if testCase.Unique {
				suite.True(found)
			} else {
				suite.False(found)
			}

			found, err = client.HasItem(ctx, []byte("not found content"))
			suite.Require().NoError(err)
			suite.False(found)

			content, err := client.PopItem(ctx)
			suite.Require().NoError(err)
			suite.Equal(expectedContent, content)
		})
	}
}
