package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/middleware"

	"github.com/labstack/echo"
)

var (
	port     string
	baseURL  string
	insecure bool

	setupSQL string
	tickets  TicketsDB
)

func setVars() {
	flag.StringVar(&port, "port", "443", "port to run server on")
	flag.BoolVar(&insecure, "insecure", false, "allow non-https connections")
	flag.StringVar(&baseURL, "url", "127.0.0.1", "base URL of the server")
	flag.StringVar(&setupSQL, "sql", "setup.sql", "SQL setup script")

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

	if insecure {
		fmt.Println("Running in Insecure Mode.")
	}
	fmt.Printf("Server Root URL at %s using Port %s\n", baseURL, port)
}

type Ticket struct {
	ID          string
	User        string
	Title       string
	Description string
	Created     string
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func setup() {

	// sets up global variables based on CLI args
	setVars()

	// set up database
	var err error
	if tickets, err = NewTicketsDB("tickets.db", "setup.sql"); err != nil {
		panic("Could Not Create Database")
	}
}

func main() {

	// sets up database
	setup()

	// set up webserver

	t := &Template{
		templates: template.Must(template.ParseGlob("web/*.html")),
	}

	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Renderer = t
	e.HideBanner = true

	e.GET("/report/:id", editReport)
	e.POST("/report/:id", updateReport)
	e.GET("/report", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, baseURL)
	})
	e.POST("/report", newReport)
	e.GET("*/*", mainPage)
	e.Logger.Fatal(e.Start(":" + port))
}

func mainPage(c echo.Context) error {
	fmt.Println("Tickets DB: ", tickets.DB)
	tickets, err := tickets.getAllTickets()
	if err != nil {
		panic(err)
	}
	return c.Render(http.StatusOK, "list", tickets)
}

func editReport(c echo.Context) error {
	id := c.Param("id")
	ticket, err := tickets.getTicketByID(id)

	// Empty title means no results
	if ticket.Title == "" || err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, baseURL)
	}

	ticket.ID = id

	return c.Render(http.StatusOK, "report", ticket)
}

func updateReport(c echo.Context) error {
	id := c.Param("id")
	title := c.FormValue("title")
	description := c.FormValue("description")

	ticket := Ticket{
		ID:          id,
		Title:       title,
		Description: description,
	}

	_, err := tickets.updateTicket(&ticket)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	return editReport(c)
}

type SlackMessage struct {
	Response    string        `json:"response_type"`
	Text        string        `json:"text"`
	Attachments []interface{} `json:"attachments"`
}

func newReport(c echo.Context) error {
	data, err := c.FormParams()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error")
	}

	cTime := time.Now().UTC().Format(time.ANSIC)

	user := data["user_id"][0]

	title := data["text"][0]

	ticketHash := sha1.Sum([]byte(cTime + user + title))

	ticketID := hex.EncodeToString(ticketHash[:])

	ticket := Ticket{
		ID:          ticketID,
		User:        data["user_id"][0],
		Title:       data["text"][0],
		Description: "Describe ticket here",
		Created:     cTime,
	}

	_, err = tickets.addTicket(&ticket)
	if err != nil {
		fmt.Printf("Error Adding Ticket to DB: %s\n", err)
	}

	go sendTicketCreatedMessage(data["response_url"][0], &ticket)

	return c.NoContent(http.StatusOK)
}

func sendTicketCreatedMessage(msgURL string, ticket *Ticket) {

	responseText := "Ticket \"" + ticket.Title + "\" created by <@" + ticket.User + ">."

	ticketURL := baseURL + "report/" + ticket.ID

	var attachments []interface{}
	attachments = append(attachments, map[string]string{"text": ticketURL})

	response := &SlackMessage{
		Response:    "in_channel",
		Text:        responseText,
		Attachments: attachments,
	}

	jsonData, err := json.Marshal(response)

	if err != nil {
		panic(err)
	}

	http.Post(msgURL, "application/json", bytes.NewBuffer(jsonData))
}
