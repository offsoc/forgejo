// Copyright twenty-panda <twenty-panda@posteo.com>
// SPDX-License-Identifier: MIT

package pushoptions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmpty(t *testing.T) {
	options := New()
	assert.True(t, options.Empty())
	options.Parse(fmt.Sprintf("%v", RepoPrivate))
	assert.False(t, options.Empty())
}

func TestToAndFromMap(t *testing.T) {
	options := New()
	options.Parse(fmt.Sprintf("%v", RepoPrivate))
	actual := options.Map()
	expected := map[string]string{string(RepoPrivate): "true"}
	assert.Equal(t, expected, actual)
	assert.Equal(t, expected, NewFromMap(&actual).Map())
}

func TestChangeRepositorySettings(t *testing.T) {
	options := New()
	assert.False(t, options.ChangeRepoSettings())
	assert.True(t, options.Parse(fmt.Sprintf("%v=description", AgitDescription)))
	assert.False(t, options.ChangeRepoSettings())

	options.Parse(fmt.Sprintf("%v", RepoPrivate))
	assert.True(t, options.ChangeRepoSettings())

	options = New()
	options.Parse(fmt.Sprintf("%v", RepoTemplate))
	assert.True(t, options.ChangeRepoSettings())
}

func TestParse(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		options := New()

		val, ok := options.GetString(RepoPrivate)
		assert.False(t, ok)
		assert.Empty(t, val)

		assert.True(t, options.GetBool(RepoPrivate, true))
		assert.False(t, options.GetBool(RepoPrivate, false))
	})

	t.Run("key=value", func(t *testing.T) {
		options := New()

		topic := "TOPIC"
		assert.True(t, options.Parse(fmt.Sprintf("%v=%s", AgitTopic, topic)))
		val, ok := options.GetString(AgitTopic)
		assert.True(t, ok)
		assert.Equal(t, topic, val)
	})

	t.Run("key=true", func(t *testing.T) {
		options := New()

		assert.True(t, options.Parse(fmt.Sprintf("%v=true", RepoPrivate)))
		assert.True(t, options.GetBool(RepoPrivate, false))
		assert.True(t, options.Parse(fmt.Sprintf("%v=TRUE", RepoTemplate)))
		assert.True(t, options.GetBool(RepoTemplate, false))
	})

	t.Run("key=false", func(t *testing.T) {
		options := New()

		assert.True(t, options.Parse(fmt.Sprintf("%v=false", RepoPrivate)))
		assert.False(t, options.GetBool(RepoPrivate, true))
	})

	t.Run("key", func(t *testing.T) {
		options := New()

		assert.True(t, options.Parse(fmt.Sprintf("%v", RepoPrivate)))
		assert.True(t, options.GetBool(RepoPrivate, false))
	})

	t.Run("unknown keys are ignored", func(t *testing.T) {
		options := New()

		assert.True(t, options.Empty())
		assert.False(t, options.Parse("unknown=value"))
		assert.True(t, options.Empty())
	})
}

func TestReadEnv(t *testing.T) {
	t.Setenv(envPrefix+"_0", fmt.Sprintf("%v=true", AgitForcePush))
	t.Setenv(envPrefix+"_1", fmt.Sprintf("%v", RepoPrivate))
	t.Setenv(envPrefix+"_2", fmt.Sprintf("%v=equal=in string", AgitTitle))
	t.Setenv(envPrefix+"_3", "not=valid")
	t.Setenv(envPrefix+"_4", fmt.Sprintf("%v=description", AgitDescription))
	t.Setenv(EnvCount, "5")

	options := New().ReadEnv()

	assert.True(t, options.GetBool(AgitForcePush, false))
	assert.True(t, options.GetBool(RepoPrivate, false))
	assert.False(t, options.GetBool(RepoTemplate, false))

	{
		val, ok := options.GetString(AgitTitle)
		assert.True(t, ok)
		assert.Equal(t, "equal=in string", val)
	}
	{
		val, ok := options.GetString(AgitDescription)
		assert.True(t, ok)
		assert.Equal(t, "description", val)
	}
	{
		_, ok := options.GetString(AgitTopic)
		assert.False(t, ok)
	}
}
