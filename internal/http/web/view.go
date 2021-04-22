package web

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
)

const (
	// AlertErrorMsgGeneric is displayed when any random error
	// is encountered by our backend.
	AlertErrorMsgGeneric       = "Sorry; something went wrong."
	AlertFormParseErrorGeneric = "Sorry; there was a problem parsing your form."
)

func newView(layout string, files ...string) *view {
	commonFiles := []string{"../../web/template/common/base.html", "../../web/template/alert.html"}
	files = append(files, commonFiles...)
	// Add a genCSRFField function to the template so we can change it in the render function
	t, err := template.New("").Funcs(template.FuncMap{
		"genCSRFField": func() (template.HTML, error) {
			// This function is generated in the when the view is rendered
			return "", errors.New("csrfField is not implemented")
		},
	}).ParseFiles(files...)
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

func (v *view) render(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	var viewData Data
	switch d := data.(type) {
	case Data:
		viewData = d
	default:
		viewData = Data{
			Yield: data,
		}
	}

	user, ok := r.Context().Value(contextKeyUser).(lister.User)
	if ok {
		viewData.User.ID = user.ID
		viewData.User.FirstName = user.FirstName
	}

	csrfField := csrf.TemplateField(r)
	tpl := v.Template.Funcs(template.FuncMap{
		"genCSRFField": func() template.HTML {
			return csrfField
		},
	})

	// Execute the template into a buffer 1st to see if there was a problem. This prevents rendering a incomplete
	// template to the user.
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, v.Layout, viewData)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "There was a problem rendering the html template", http.StatusInternalServerError)
		return
	}
	io.Copy(w, &buf)
}
