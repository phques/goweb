// Very simple wrapper classes to help use http.HandleFunc & cie.
// The 'handler' funcs receive a Context and return a go error.
// Handlers can call Render to render templates from go's html/template.
// This uses go 1.22> http.HandleFunc("GET "+pattern, http.HandleFunc)
package webhelper

import (
	"html/template"
	"net/http"
)

// ---------------

// Used to return an (type) error from the handlers
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// this makes Error usable as a 'error'
func (e Error) Error() string {
	return e.Message
}

// Creates an instance of Error
func NewError(code int, message string) Error {
	return Error{
		Code:    code,
		Message: message,
	}
}

// ---------------

// Handler functions are of this type
type Handler func(c Context) error

type Context struct {
	w         http.ResponseWriter
	r         *http.Request
	templates *template.Template
}

// This gives access to .Get, .Post to register handlers,
// also gives access to the templates for Render
type Server struct {
	Templates      *template.Template
	currentContext *Context
}

func (s *Server) handleError(err error) {
	// default to StatusInternalServerError,
	// try to get code from a Error if it is one
	code := http.StatusInternalServerError
	if myError, ok := err.(Error); ok {
		code = myError.Code
	}

	// :( too little knowledge to know what to do / how to correctly handle this!
	// so just call Error (others seem to send JSON, I guess that is then used on the client side)
	s.currentContext.Error(err.Error(), code)
}

// Creates an http.HandlerFunc that wraps our own Handlers.
// It creates the context that is passed to the handler,
// and will process the returned error if any.
func (s *Server) makeHandler(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		context := Context{
			w:         w,
			r:         r,
			templates: s.Templates,
		}
		s.currentContext = &context

		if err := handler(context); err != nil {
			s.handleError(err)
		}
	}
}

// Register a GET handler
func (s *Server) Get(pattern string, handler Handler) {
	wrappedHandler := s.makeHandler(handler)
	http.HandleFunc("GET "+pattern, wrappedHandler)
}

// Register a POST handler
func (s *Server) Post(pattern string, handler Handler) {
	wrappedHandler := s.makeHandler(handler)
	http.HandleFunc("POST "+pattern, wrappedHandler)
}

//--------

// Renders the templates
func (c *Context) Render(name string, data any) error {
	if err := c.templates.ExecuteTemplate(c.w, name, data); err != nil {
		//c.Error(err.Error(), http.StatusInternalServerError)
		return NewError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// Wrappers around http.Request's PathValue, FormValue etc
// and http. Redirect, Error
func (c *Context) PathValue(name string) string {
	return c.r.PathValue(name)
}

func (c *Context) FormValue(key string) string {
	return c.r.FormValue(key)
}

func (c *Context) Redirect(url string) {
	c.RedirectWithStatus(url, http.StatusFound)
}

func (c *Context) RedirectWithStatus(url string, status int) {
	http.Redirect(c.w, c.r, url, status)
}

func (c *Context) Error(error string, code int) {
	http.Error(c.w, error, code)
}
