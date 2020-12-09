package web

// Data is structure that views expect data to be in
type Data struct {
	Head
	Alert Alert
	Yield interface{}
}

// Head is the structure for the header part of the templates
type Head struct {
	Title string
}

// Alert is used to render messages in templates
type Alert struct {
	Message string
}
