package web

import (
	"html/template"
	"net/http"
)

const (
	// AlertErrorMsgGeneric is displayed when any random error
	// is encountered by our backend.
	AlertErrorMsgGeneric = "Sorry; something went wrong."
	AlertFormParseErrorGeneric = "Sorry; there was a problem parsing your form."
)

func newView(layout string, files ...string) *view {
	commonFiles := []string{"../../web/template/common/base.html", "../../web/template/alert.html"}
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
	w.Header().Set("Content-Type", "text/html")
	switch data.(type) {
	case Data:
		// do nothing
	default:
		data = Data{
			Yield: data,
		}
	}
	return v.Template.ExecuteTemplate(w, v.Layout, data)
}
