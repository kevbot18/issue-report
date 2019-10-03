package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/labstack/echo"
)

var (
	port     string
	baseURL  string
	issues   map[string]Issue
	insecure bool
)

func init() {
	flag.StringVar(&port, "port", "443", "port to run server on")
	flag.BoolVar(&insecure, "insecure", false, "allow non-https connections")
	flag.StringVar(&baseURL, "url", "127.0.0.1", "base URL of the server")

	flag.Parse()

	if !((strings.HasPrefix("http://", baseURL) && insecure) || strings.HasPrefix("https://", baseURL)) {
		if insecure {
			baseURL = "http://" + baseURL
		} else {
			baseURL = "https://" + baseURL
		}
	}

	if !(strings.HasSuffix(baseURL, "/")) {
		baseURL = baseURL + "/"
	}

	issues = make(map[string]Issue)

	if insecure {
		fmt.Println("Running in Insecure Mode.")
	}
	fmt.Printf("Server Root URL at %s using Port %s\n", baseURL, port)
}

type Issue struct {
	User        string
	Title       string
	Description string
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {

	t := &Template{
		templates: template.Must(template.ParseGlob("web/*.html")),
	}

	e := echo.New()
	e.Renderer = t
	e.HideBanner = true

	e.GET("/report/:id", editReport)
	e.POST("/report/:id", updateReport)
	e.POST("/report", newReport)
	e.GET("/", mainPage)
	e.Logger.Fatal(e.Start(":" + port))
}

func mainPage(c echo.Context) error {
	return c.Render(http.StatusOK, "list", issues)
}

func editReport(c echo.Context) error {
	id := c.Param("id")
	page, ok := issues[id]
	if ok {
		return c.Render(http.StatusOK, "report", page)
	}
	return c.Redirect(http.StatusSeeOther, "/")
}

func updateReport(c echo.Context) error {
	id := c.Param("id")
	title := c.FormValue("title")
	description := c.FormValue("description")
	page, ok := issues[id]
	if ok {
		page = Issue{
			User:        page.User,
			Title:       title,
			Description: description,
		}

		issues[id] = page

		return c.Render(http.StatusOK, "report", page)
	}
	return c.NoContent(http.StatusNotFound)
}

func newReport(c echo.Context) error {
	data, err := c.FormParams()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error")
	}

	cTime := time.Now().UTC().Format(time.ANSIC)

	issueHash := sha1.Sum([]byte(cTime + data["user_id"][0] + data["text"][0]))

	issueID := hex.EncodeToString(issueHash[:])

	issue := Issue{
		User:        data["user_id"][0],
		Title:       data["text"][0],
		Description: "Describe issue here",
	}

	issues[issueID] = issue

	return c.String(http.StatusOK, baseURL+"report/"+issueID)
}
