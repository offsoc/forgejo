package misc

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/services/context"
)

// Get the relative time constant definitions for the current language.
func RelativeTimeConstants(ctx *context.Context) {
	lang := ctx.Locale
	text := fmt.Sprintf(`
		DATETIMESTRINGS = {
			'future': '%s',
			'now': '%s',
			'1min': '%s',
			'mins': (minutes) => %s%s%s,
			'1hour': '%s',
			'hour': (hours) => %s%s%s,
			'1day': '%s',
			'days': (days) => %s%s%s,
			'1week': '%s',
			'weeks': (weeks) => %s%s%s,
			'1month': '%s',
			'months': (months) => %s%s%s,
			'1year': '%s',
			'years': (years) => %s%s%s,
		};`, lang.TrString("tool.future"), lang.TrString("tool.now"), lang.TrString("tool.ago_1min"), "`", lang.TrString("tool.ago_mins"), "`", lang.TrString("tool.ago_1hour"), "`", lang.TrString("tool.ago_hours"), "`", lang.TrString("tool.ago_1day"), "`", lang.TrString("tool.ago_days"), "`", lang.TrString("tool.ago_1week"), "`", lang.TrString("tool.ago_weeks"), "`", lang.TrString("tool.ago_1month"), "`", lang.TrString("tool.ago_months"), "`", lang.TrString("tool.ago_1year"), "`", lang.TrString("tool.ago_years"), "`")

	ctx.PlainText(http.StatusOK, text)
}
