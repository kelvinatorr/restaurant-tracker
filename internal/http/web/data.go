package web

const (
	// AlertErrorMsgGeneric is displayed when any random error
	// is encountered by our backend.
	AlertErrorMsgGeneric = "Sorry; something went wrong."
)

// Data is structure that views expect data to be in
type Data struct {
	Header Header
	Alert  *Alert
	Yield  interface{}
}

// Header is the structure for the header part of the templates
type Header struct {
	Title string
}

// Alert is used to render messages in templates
type Alert struct {
	Message string
}
