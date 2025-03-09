// Copyright 2014 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"os"
	"os/user"
)

// CurrentUsername return current login OS user name
func CurrentUsername() string {
	userinfo, err := user.Current()
	if err != nil {
		return fallbackCurrentUsername()
	}
	return userinfo.Username
}

// Old method, used if new method doesn't work on your OS for some reason
func fallbackCurrentUsername() string {
	curUserName := os.Getenv("USER")
	if len(curUserName) > 0 {
		return curUserName
	}

	return os.Getenv("USERNAME")
}
