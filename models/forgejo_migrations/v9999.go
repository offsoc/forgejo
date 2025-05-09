package forgejo_migrations

import (
	"xorm.io/xorm"
)

func addMirrorBranchFilter(x *xorm.Engine) error {
	type Mirror struct {
		BranchFilter string `xorm:"VARCHAR(255)"`
	}
	return x.Sync2(new(Mirror))
}
