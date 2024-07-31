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
	R         *http.Request
	W         http.ResponseWriter
	templates *template.Template
}

// This is the server, create one with CreateServer
// It is then used to register handlers and 'run' the server
type Server struct {
	templates      *template.Template
	currentContext *Context // only valid during a call to Handler
	address        string
	router         *http.ServeMux
	server         *http.Server
	middlewares    []Handler // these are called before the registerer Handler
}

// Creates a Server.
// Register handlers with Post, Get ..
// Call Run to start the server
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

// Starts the server, accepting requests and dispatching them
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
			W:         w,
			R:         r,
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

// Register a middleware function,
// the regitered mdw functions will be executed before the registered handler.
// Can be used for example to log requests etc
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
	err := c.templates.ExecuteTemplate(c.W, name, data)
	if err != nil {
		return NewError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// Wrappers around http.Request's PathValue, FormValue etc
// and http. Redirect, Error
func (c *Context) PathValue(name string) string {
	return c.R.PathValue(name)
}

func (c *Context) FormValue(key string) string {
	return c.R.FormValue(key)
}

func (c *Context) Path() string {
	return c.R.URL.Path
}

func (c *Context) Method() string {
	return c.R.Method
}

func (c *Context) RedirectWithStatus(url string, status int) {
	http.Redirect(c.W, c.R, url, status)
}

func (c *Context) Redirect(url string) {
	c.RedirectWithStatus(url, http.StatusFound)
}

func (c *Context) WriteString(data string) {
	c.W.Write([]byte(data))
}

func (c *Context) Write(data []byte) {
	c.W.Write(data)
}

func (c *Context) Error(error string, code int) {
	http.Error(c.W, error, code)
}
