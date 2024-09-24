package utils

import "code.gitea.io/gitea/services/context"

// check if api token contains `public-only` scope
func PublicOnlyToken(ctx *context.APIContext, scopeKey string) bool {
	publicScope, _ := ctx.Data[scopeKey].(bool)
	return publicScope
}
