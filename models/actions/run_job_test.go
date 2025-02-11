// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionRunJob_ItRunsOn(t *testing.T) {
	actionJob := ActionRunJob{RunsOn: []string{"ubuntu"}}
	agentLabels := []string{"ubuntu", "node-20"}

	assert.True(t, actionJob.ItRunsOn(agentLabels))
	assert.False(t, actionJob.ItRunsOn([]string{}))

	actionJob.RunsOn = append(actionJob.RunsOn, "node-20")

	assert.True(t, actionJob.ItRunsOn(agentLabels))

	agentLabels = []string{"ubuntu"}

	assert.False(t, actionJob.ItRunsOn(agentLabels))

	actionJob.RunsOn = []string{}

	assert.False(t, actionJob.ItRunsOn(agentLabels))
}
