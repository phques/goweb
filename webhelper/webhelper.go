// Very simple wrapper classes to help use http.HandleFunc & cie.
// The 'handler' funcs receive a Context and return a go error.
// Handlers can call Render to render templates from go's html/template.
// This uses go 1.22> http.HandleFunc("GET "+pattern, http.HandleFunc)
package webhelper

import (
	"html/template"
	"log"
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
	templates      *template.Template
	currentContext *Context
	address        string
	router         *http.ServeMux
	server         *http.Server
	middlewares    []Handler
}

func CreateServer(address string, templates *template.Template) *Server {
	router := http.NewServeMux()
	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	return &Server{
		templates:   templates,
		address:     address,
		router:      router,
		server:      server,
		middlewares: make([]Handler, 0, 10),
	}
}

func (s *Server) Run() error {
	log.Printf("server running on %s\n", s.address)
	return s.server.ListenAndServe()
}

func (s *Server) handleError(err error) {
	// default to StatusInternalServerError,
	// try to get code from a Error if it is one
	code := http.StatusInternalServerError
	if myError, ok := err.(Error); ok {
		code = myError.Code
	}

	// just call Error (others seem to send JSON, I guess that is then used on the client side)
	s.currentContext.Error(err.Error(), code)
}

// Creates an http.HandlerFunc that wraps our own Handlers.
// It creates the context that is passed to the handler,
// and will process the returned error if any.
func (s *Server) makeHandler(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// create a Context
		context := Context{
			w:         w,
			r:         r,
			templates: s.templates,
		}

		s.currentContext = &context
		defer func() { s.currentContext = nil }()

		// call middlewares
		for _, middleware := range s.middlewares {
			if err := middleware(context); err != nil {
				s.handleError(err)
				return
			}
		}
		// call the handler
		if err := handler(context); err != nil {
			s.handleError(err)
		}
	}
}

// Add a middleware
func (s *Server) AddMiddlware(m Handler) {
	s.middlewares = append(s.middlewares, m)
}

// Register a GET handler
func (s *Server) Get(pattern string, handler Handler) {
	wrappedHandler := s.makeHandler(handler)
	s.router.HandleFunc("GET "+pattern, wrappedHandler)
}

// Register a POST handler
func (s *Server) Post(pattern string, handler Handler) {
	wrappedHandler := s.makeHandler(handler)
	s.router.HandleFunc("POST "+pattern, wrappedHandler)
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

func (c *Context) Path() string {
	return c.r.URL.Path
}

func (c *Context) Method() string {
	return c.r.Method
}

func (c *Context) RedirectWithStatus(url string, status int) {
	http.Redirect(c.w, c.r, url, status)
}

func (c *Context) Redirect(url string) {
	c.RedirectWithStatus(url, http.StatusFound)
}

func (c *Context) Error(error string, code int) {
	http.Error(c.w, error, code)
}
