package shared

import (
	"html"
	"html/template"
	"strconv"

	"forgejo.org/modules/setting"
)

func ImgIcon(name string, size int) template.HTML {
	s := strconv.Itoa(size)
	src := html.EscapeString(setting.StaticURLPrefix + "/assets/img/" + name)
	return template.HTML(`<img width="` + s + `" height="` + s + `" src="` + src + `">`)
}
