package main

import (
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

//----------------------

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := "wikis/" + p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := "wikis/" + title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

//----------------------

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if err := t.templates.ExecuteTemplate(w, name+".html", data); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

//----------------------

func viewHandler(c echo.Context) error {
	title := c.Param("title")

	page, err := loadPage(title)
	if err != nil {
		return c.Redirect(http.StatusFound, "/edit/"+title)
	}

	return c.Render(http.StatusOK, "view", page)
}

func viewHandler2(c echo.Context) error {
	title := c.QueryParam("title")
	// can still do /view?title=toto/tata
	// and we will get title = "toto/tata"
	if title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing 'title' parameter")
	}

	page, err := loadPage(title)
	if err != nil {
		return c.Redirect(http.StatusFound, "/edit/"+title)
	}

	return c.Render(http.StatusOK, "view", page)
}

func editHandler(c echo.Context) error {
	title := c.Param("title")

	page, err := loadPage(title)
	if err != nil {
		page = &Page{Title: title}
	}

	return c.Render(http.StatusOK, "edit", page)
}

func saveHandler(c echo.Context) error {
	title := c.Param("title")
	body := c.FormValue("body")

	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.Redirect(http.StatusFound, "/view/"+title)
}

func main() {
	e := echo.New()
	e.Renderer = &Template{
		template.Must(template.ParseGlob("templates/*.html")),
	}

	//## equivalent go 1.22 http.HandleFunc("GET /view/{title}".
	//  would give "404 page not found" for /view/toto/tata
	//  but these accept the request with title=toto/tata
	//  (same as older http.HandleFunc("/view"..)
	//  (fiber seems to behave as go 1.22)
	e.GET("/view/:title", viewHandler)
	e.GET("/view2", viewHandler2)
	e.GET("/edit/:title", editHandler)
	e.POST("/save/:title", saveHandler)
	e.Logger.Fatal(e.Start(":8081"))
}
