// Copyright 2023, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
)

func (id ActorID) AsWellKnownNodeInfoURI() string {
	wellKnownPath := ".well-known/nodeinfo"
	var result string
	if id.HostPort == 0 {
		result = fmt.Sprintf("%s://%s/%s", id.HostSchema, id.Host, wellKnownPath)
	} else {
		result = fmt.Sprintf("%s://%s:%d/%s", id.HostSchema, id.Host, id.HostPort, wellKnownPath)
	}
	return result
}
