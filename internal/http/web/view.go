package web

import (
	"html/template"
	"net/http"
)

func newView(layout string, files ...string) *view {
	commonFiles := []string{"../../web/template/common/base.html"}
	files = append(files, commonFiles...)
	t, err := template.ParseFiles(files...)
	if err != nil {
		panic(err)
	}

	return &view{
		Template: t,
		Layout:   layout,
	}
}

type view struct {
	Template *template.Template
	Layout   string
}

func (v *view) render(w http.ResponseWriter, data interface{}) error {
	return v.Template.ExecuteTemplate(w, v.Layout, data)
}
