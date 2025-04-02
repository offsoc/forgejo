// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/attachment"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
	"github.com/google/uuid"
)

var _ f3_tree.ForgeDriverInterface = &issue{}

type asset struct {
	common

	forgejoAsset *repo_model.Attachment
	sha          string
	contentType  string
	downloadFunc f3.DownloadFuncType
}

func (o *asset) SetNative(asset any) {
	o.forgejoAsset = asset.(*repo_model.Attachment)
}

func (o *asset) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoAsset.ID)
}

func (o *asset) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *asset) ToFormat() f3.Interface {
	if o.forgejoAsset == nil {
		return o.NewFormat()
	}

	return &f3.ReleaseAsset{
		Common:        f3.NewCommon(o.GetNativeID()),
		Name:          o.forgejoAsset.Name,
		ContentType:   o.contentType,
		Size:          o.forgejoAsset.Size,
		DownloadCount: o.forgejoAsset.DownloadCount,
		Created:       o.forgejoAsset.CreatedUnix.AsTime(),
		SHA256:        o.sha,
		DownloadURL:   o.forgejoAsset.DownloadURL(),
		DownloadFunc:  o.downloadFunc,
	}
}

func (o *asset) FromFormat(content f3.Interface) {
	asset := content.(*f3.ReleaseAsset)
	o.forgejoAsset = &repo_model.Attachment{
		ID:                f3_util.ParseInt(asset.GetID()),
		Name:              asset.Name,
		Size:              asset.Size,
		DownloadCount:     asset.DownloadCount,
		CreatedUnix:       timeutil.TimeStamp(asset.Created.Unix()),
		CustomDownloadURL: asset.DownloadURL,
	}
	o.contentType = asset.ContentType
	o.sha = asset.SHA256
	o.downloadFunc = asset.DownloadFunc
}

func (o *asset) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	asset, err := repo_model.GetAttachmentByID(ctx, id)
	if repo_model.IsErrAttachmentNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("asset %v %w", id, err))
	}

	o.forgejoAsset = asset

	path := o.forgejoAsset.RelativePath()

	{
		f, err := storage.Attachments.Open(path)
		if err != nil {
			panic(err)
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			panic(fmt.Errorf("io.Copy to hasher: %v", err))
		}
		o.sha = hex.EncodeToString(hasher.Sum(nil))
	}

	o.downloadFunc = func() io.ReadCloser {
		o.Trace("download %s from copy stored in temporary file %s", o.forgejoAsset.DownloadURL, path)
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		return f
	}
	return true
}

func (o *asset) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoAsset.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoAsset.ID).Cols("name").Update(o.forgejoAsset); err != nil {
		panic(fmt.Errorf("UpdateAssetCols: %v %v", o.forgejoAsset, err))
	}
}

func (o *asset) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	uploader, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}

	o.forgejoAsset.UploaderID = uploader.ID
	o.forgejoAsset.RepoID = f3_tree.GetProjectID(o.GetNode())
	o.forgejoAsset.ReleaseID = f3_tree.GetReleaseID(o.GetNode())
	o.forgejoAsset.UUID = uuid.New().String()

	download := o.downloadFunc()
	defer download.Close()

	_, err = attachment.NewAttachment(ctx, o.forgejoAsset, download, o.forgejoAsset.Size)
	if err != nil {
		panic(err)
	}

	o.Trace("asset created %d", o.forgejoAsset.ID)
	return f3_id.NewNodeID(o.forgejoAsset.ID)
}

func (o *asset) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := repo_model.DeleteAttachment(ctx, o.forgejoAsset, true); err != nil {
		panic(err)
	}
}

func newAsset() generic.NodeDriverInterface {
	return &asset{}
}
