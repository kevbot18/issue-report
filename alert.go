package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/labstack/echo/middleware"

	"github.com/labstack/echo"
	_ "github.com/mattn/go-sqlite3"
)

var (
	port     string
	baseURL  string
	insecure bool

	setupSQL string

	db *sql.DB
)

func init() {
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

func main() {

	var err error = nil

	// set up DB
	db, err = setupDB("tickets.db")
	if err != nil {
		panic(err)
	}

	db.Stats()

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
	tickets, err := getAllTicketsFromDB(db)
	if err != nil {
		panic(err)
	}
	return c.Render(http.StatusOK, "list", tickets)
}

func editReport(c echo.Context) error {
	id := c.Param("id")
	ticket, err := getTicketFromDB(db, id)

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

	_, err := updateTicket(db, &ticket)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	return editReport(c)
}

func newReport(c echo.Context) error {
	data, err := c.FormParams()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error")
	}

	cTime := time.Now().UTC().Format(time.ANSIC)

	ticketHash := sha1.Sum([]byte(cTime + data["user_id"][0] + data["text"][0]))

	ticketID := hex.EncodeToString(ticketHash[:])

	ticket := Ticket{
		ID:          ticketID,
		User:        data["user_id"][0],
		Title:       data["text"][0],
		Description: "Describe ticket here",
		Created:     cTime,
	}

	_, err = addTicketToDB(db, &ticket)
	if err != nil {
		fmt.Printf("Error Adding Ticket to DB: %s\n", err)
	}

	return c.String(http.StatusOK, baseURL+"report/"+ticketID)
}

func setupDB(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", file)

	if err != nil || db == nil {
		return nil, err
	}

	if migrateDB(db, setupSQL) != nil {
		return nil, err
	}

	return db, nil
}

func migrateDB(db *sql.DB, file string) error {
	setupScript, err := ioutil.ReadFile(setupSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(string(setupScript))

	if err != nil {
		return err
	}

	return nil
}

func addTicketToDB(db *sql.DB, ticket *Ticket) (int64, error) {
	sql := "INSERT INTO tickets(id, title, description, createdAt, createdBy) VALUES(?, ?, ?, ?, ?)"

	stmt, err := db.Prepare(sql)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	result, err := stmt.Exec(ticket.ID, ticket.Title, ticket.Description, ticket.Created, ticket.User)

	if err != nil {
		panic(err)
	}

	return result.LastInsertId()
}

func updateTicket(db *sql.DB, ticket *Ticket) (int64, error) {
	sql := "UPDATE tickets SET title = (?), description = (?) WHERE id = (?)"

	stmt, err := db.Prepare(sql)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	result, err := stmt.Exec(ticket.Title, ticket.Description, ticket.ID)

	if err != nil {
		panic(err)
	}

	return result.RowsAffected()
}

func getTicketFromDB(db *sql.DB, id string) (Ticket, error) {
	sql := "SELECT title, description, createdBy FROM tickets WHERE id = (?)"

	stmt, err := db.Prepare(sql)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	rows, err := stmt.Query(id)

	if err != nil {
		panic(err)
	}

	ticket := Ticket{
		ID: id,
	}

	// Only care about first result, searching by key anyway
	isFirst := true
	for rows.Next() && isFirst {
		isFirst = false
		err = rows.Scan(&ticket.Title, &ticket.Description, &ticket.User)

		rows.Close()
	}

	return ticket, nil
}

func getAllTicketsFromDB(db *sql.DB) ([]Ticket, error) {
	sql := "SELECT id, title FROM tickets"
	rows, err := db.Query(sql)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	ticketList := make([]Ticket, 0)
	for rows.Next() {
		ticket := Ticket{}
		err := rows.Scan(&ticket.ID, &ticket.Title)

		if err != nil {
			panic(err)
		}
		ticketList = append(ticketList, ticket)
	}

	return ticketList, nil
}
