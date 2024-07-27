package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
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

func viewHandler(c *fiber.Ctx) error {
	title := c.Params("title")

	page, err := loadPage(title)
	if err != nil {
		return c.Redirect("/edit/" + title)
	}

	return c.Render("view", page)
}

func editHandler(c *fiber.Ctx) error {
	title := c.Params("title")

	page, err := loadPage(title)
	if err != nil {
		page = &Page{Title: title}
	}

	return c.Render("edit", page)
}

func saveHandler(c *fiber.Ctx) error {
	title := c.Params("title")
	body := c.FormValue("body")

	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}

	return c.Redirect("/view/" + title)
}

func main() {
	// Initialize standard Go html template engine
	engine := html.New("./templates", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/view/:title", viewHandler)
	app.Get("/edit/:title", editHandler)
	app.Post("/save/:title", saveHandler)
	log.Fatal(app.Listen(":8081"))
}
