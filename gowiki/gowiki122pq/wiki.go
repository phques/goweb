package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/phques/goweb/webhelper"
)

//----------

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

//----------

func viewHandler(c webhelper.Context) error {
	title := c.PathValue("title")

	if page, err := loadPage(title); err == nil {
		return c.Render("view.html", page)
	}

	c.Redirect("/edit/" + title)
	return nil
}

func editHandler(c webhelper.Context) error {
	title := c.PathValue("title")

	// load page, create new one with title if not found
	page, err := loadPage(title)
	if err != nil {
		page = &Page{Title: title}
	}

	return c.Render("edit.html", page)
}

func saveHandler(c webhelper.Context) error {
	title := c.PathValue("title")
	body := c.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}

	if err := p.save(); err != nil {
		return webhelper.NewError(http.StatusInternalServerError, err.Error())
	}

	c.Redirect("/view/" + title)
	return nil
}

//-----------

func denyOops(c webhelper.Context) error {
	// dummy mdw, to test that we can stop processing,
	// for example if doing authentication
	if c.Path() == "/view/oops" {
		log.Printf("denying [%s] at [%s]\n", c.Method(), c.Path())
		return webhelper.NewError(http.StatusBadRequest, "oops indeed (denied)")
	}
	return nil
}

func main() {
	templates := template.Must(template.ParseGlob("templates/*.html"))
	server := webhelper.CreateServer(":8080", templates)

	// add a mdw that logs what we receive
	server.AddMiddlware(func(c webhelper.Context) error {
		log.Printf("received [%s] at [%s]\n", c.Method(), c.Path())
		return nil
	})

	// add a dummy mdw that will deny access to "/view/oops"
	server.AddMiddlware(denyOops)

	server.Get("/view/{title}", viewHandler)
	server.Get("/edit/{title}", editHandler)
	server.Post("/save/{title}", saveHandler)

	log.Fatal(server.Run())
}
