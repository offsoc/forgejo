// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"encoding/hex"
	"fmt"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/util"

	gouuid "github.com/google/uuid"
)

func RegisterRunner(ctx context.Context, ownerID, repoID int64, token string, labels *[]string, name, version string) (*ActionRunner, error) {
	uuid, err := gouuid.FromBytes([]byte(token[:16]))
	if err != nil {
		return nil, fmt.Errorf("gouuid.FromBytes %v", err)
	}
	uuidString := uuid.String()

	var runner ActionRunner

	has, err := db.GetEngine(ctx).Where("uuid=?", uuidString).Get(&runner)
	if err != nil {
		return nil, fmt.Errorf("GetRunner %v", err)
	} else if !has {
		//
		// The runner does not exist yet, create it
		//
		saltBytes, err := util.CryptoRandomBytes(16)
		if err != nil {
			return nil, fmt.Errorf("CryptoRandomBytes %v", err)
		}
		salt := hex.EncodeToString(saltBytes)

		hash := auth_model.HashToken(token, salt)

		runner = ActionRunner{
			UUID:        uuidString,
			TokenHash:   hash,
			TokenSalt:   salt,
			AgentLabels: []string{},
		}

		if err := CreateRunner(ctx, &runner); err != nil {
			return &runner, fmt.Errorf("can't create new runner %w", err)
		}
	}

	//
	// Update the existing runner
	//
	name, _ = util.SplitStringAtByteN(name, 255)

	cols := []string{"name", "owner_id", "repo_id", "version"}
	runner.Name = name
	runner.OwnerID = ownerID
	runner.RepoID = repoID
	runner.Version = version
	if labels != nil {
		runner.AgentLabels = *labels
		cols = append(cols, "agent_labels")
	}

	if err := UpdateRunner(ctx, &runner, cols...); err != nil {
		return &runner, fmt.Errorf("can't update the runner %+v %w", runner, err)
	}

	return &runner, nil
}
