// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/validation"
)

func init() {
	db.RegisterModel(new(FederationHost))
}

func GetFederationHost(ctx context.Context, ID int64) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", ID)
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return host, nil
}

func findFederationHostFromDB(ctx context.Context, searchKey, searchValue string) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where(searchKey, searchValue).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return host, nil
}

func FindFederationHostByFqdnAndPort(ctx context.Context, fqdn string, port uint16) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("host_fqdn=? AND host_port=?", fqdn, port).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return host, nil
}

func FindFederationHostByKeyID(ctx context.Context, keyID string) (*FederationHost, error) {
	return findFederationHostFromDB(ctx, "key_id=?", keyID)
}

func CreateFederationHost(ctx context.Context, host *FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return err
	}
	_, err := db.GetEngine(ctx).Insert(host)
	return err
}

func UpdateFederationHost(ctx context.Context, host *FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return err
	}
	_, err := db.GetEngine(ctx).ID(host.ID).Update(host)
	return err
}
