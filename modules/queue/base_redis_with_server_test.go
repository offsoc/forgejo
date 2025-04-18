package queue

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"forgejo.org/modules/nosql"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/suite"
)

const defaultTestRedisServer = "127.0.0.1:6379"

type baseRedisWithServerTestSuite struct {
	suite.Suite
}

func TestBaseRedisWithServer(t *testing.T) {
	suite.Run(t, &baseRedisWithServerTestSuite{})
}

func (suite *baseRedisWithServerTestSuite) TestNormal() {
	redisAddress := "redis://" + suite.testRedisHost() + "/0"
	queueSettings := setting.QueueSettings{
		Length:  10,
		ConnStr: redisAddress,
	}

	redisServer, accessible := suite.startRedisServer(redisAddress)

	// If it's accessible, but redisServer command is nil, that means we are using
	// an already running redis server.
	if redisServer == nil && !accessible {
		suite.T().Skip("redis-server not found in Forgejo test yet")

		return
	}

	defer func() {
		if redisServer != nil {
			_ = redisServer.Process.Signal(os.Interrupt)
			_ = redisServer.Wait()
		}
	}()

	testQueueBasic(suite.T(), newBaseRedisSimple, toBaseConfig("baseRedis", queueSettings), false)
	testQueueBasic(suite.T(), newBaseRedisUnique, toBaseConfig("baseRedisUnique", queueSettings), true)
}

func (suite *baseRedisWithServerTestSuite) TestWithPrefix() {
	redisAddress := "redis://" + suite.testRedisHost() + "/0?prefix=forgejo:queue:"
	queueSettings := setting.QueueSettings{
		Length:  10,
		ConnStr: redisAddress,
	}

	redisServer, accessible := suite.startRedisServer(redisAddress)

	// If it's accessible, but redisServer command is nil, that means we are using
	// an already running redis server.
	if redisServer == nil && !accessible {
		suite.T().Skip("redis-server not found in Forgejo test yet")

		return
	}

	defer func() {
		if redisServer != nil {
			_ = redisServer.Process.Signal(os.Interrupt)
			_ = redisServer.Wait()
		}
	}()

	testQueueBasic(suite.T(), newBaseRedisSimple, toBaseConfig("baseRedis", queueSettings), false)
	testQueueBasic(suite.T(), newBaseRedisUnique, toBaseConfig("baseRedisUnique", queueSettings), true)
}

func (suite *baseRedisWithServerTestSuite) startRedisServer(address string) (*exec.Cmd, bool) {
	var redisServer *exec.Cmd

	if !suite.waitRedisReady(address, 0) {
		redisServerProg, err := exec.LookPath("redis-server")
		if err != nil {
			return nil, false
		}
		redisServer = &exec.Cmd{
			Path:   redisServerProg,
			Args:   []string{redisServerProg, "--bind", "127.0.0.1", "--port", "6379"},
			Dir:    suite.T().TempDir(),
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		suite.Require().NoError(redisServer.Start())

		if !suite.True(suite.waitRedisReady(address, 5*time.Second), "start redis-server") {
			// Return with redis server even if it's not available. It was started,
			// even if it's not reachable for any reasons, it's still started, the
			// parent will close it.
			return redisServer, false
		}
	}

	return redisServer, true
}

func (suite *baseRedisWithServerTestSuite) waitRedisReady(conn string, dur time.Duration) (ready bool) {
	ctxTimed, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	for t := time.Now(); ; time.Sleep(50 * time.Millisecond) {
		ret := nosql.GetManager().GetRedisClient(conn).Ping(ctxTimed)
		if ret.Err() == nil {
			return true
		}
		if time.Since(t) > dur {
			return false
		}
	}
}

func (suite *baseRedisWithServerTestSuite) testRedisHost() string {
	value := os.Getenv("TEST_REDIS_SERVER")
	if value != "" {
		return value
	}

	return defaultTestRedisServer
}
