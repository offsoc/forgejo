// Copyright twenty-panda <twenty-panda@posteo.com>
// SPDX-License-Identifier: MIT

package pushoptions

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Key string

const (
	RepoPrivate     = Key("repo.private")
	RepoTemplate    = Key("repo.template")
	AgitTopic       = Key("topic")
	AgitForcePush   = Key("force-push")
	AgitTitle       = Key("title")
	AgitDescription = Key("description")

	envPrefix = "GIT_PUSH_OPTION"
	EnvCount  = envPrefix + "_COUNT"
	EnvFormat = envPrefix + "_%d"
)

type Interface interface {
	ReadEnv() Interface
	Parse(string) bool
	Map() map[string]string

	ChangeRepoSettings() bool

	Empty() bool

	GetBool(key Key, def bool) bool
	GetString(key Key) (val string, ok bool)
}

type gitPushOptions map[string]string

func New() Interface {
	pushOptions := gitPushOptions(make(map[string]string))
	return &pushOptions
}

func NewFromMap(o *map[string]string) Interface {
	return (*gitPushOptions)(o)
}

func (o *gitPushOptions) ReadEnv() Interface {
	if pushCount, err := strconv.Atoi(os.Getenv(EnvCount)); err == nil {
		for idx := 0; idx < pushCount; idx++ {
			_ = o.Parse(os.Getenv(fmt.Sprintf(EnvFormat, idx)))
		}
	}
	return o
}

func (o *gitPushOptions) Parse(data string) bool {
	key, value, found := strings.Cut(data, "=")
	if !found {
		value = "true"
	}
	switch Key(key) {
	case RepoPrivate:
	case RepoTemplate:
	case AgitTopic:
	case AgitForcePush:
	case AgitTitle:
	case AgitDescription:
	default:
		return false
	}
	(*o)[key] = value
	return true
}

func (o gitPushOptions) Map() map[string]string {
	return o
}

func (o gitPushOptions) ChangeRepoSettings() bool {
	if o.Empty() {
		return false
	}
	for _, key := range []Key{RepoPrivate, RepoTemplate} {
		_, ok := o[string(key)]
		if ok {
			return true
		}
	}
	return false
}

func (o gitPushOptions) Empty() bool {
	return len(o) == 0
}

func (o gitPushOptions) GetBool(key Key, def bool) bool {
	if val, ok := o[string(key)]; ok {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return def
}

func (o gitPushOptions) GetString(key Key) (string, bool) {
	val, ok := o[string(key)]
	return val, ok
}
