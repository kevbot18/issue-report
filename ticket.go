package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

var (
	port     string
	baseURL  string
	insecure bool

	setupSQL string
	tickets  TicketsDB

	tmpl *template.Template
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
	ID          string `json:"id" db:"id"`
	User        string `json:"user" db:"createdBy"`
	Title       string `json:"title" db:"title"`
	Description string `json:"description" db:"title"`
	Created     string `json:"created" db:"createdAt"`
}

func setup() {

	// sets up global variables based on CLI args
	setVars()

	// set up database
	var err error
	if tickets, err = NewTicketsDB("tickets.db", "setup.sql"); err != nil {
		panic("Could Not Create Database")
	}

	tmpl = template.Must(template.ParseGlob("web/*.html"))
}

func main() {

	// sets up database
	setup()

	// set up webserver
	router := httprouter.New()

	router.GET("/ticket/:id", editTicket)
	router.POST("/ticket/:id", updateTicket)
	// router.GET("/ticket", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	http.Redirect(w, r, "/", http.StatusMovedPermanently)
	// })
	router.POST("/ticket", newTicket)
	router.GET("/", mainPage)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func mainPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tickets, err := tickets.getAllTickets()
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(w, "list", tickets)
}

func editTicket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ticket, err := tickets.getTicketByID(id)

	// Empty title means no results
	if ticket.Title == "" || err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404: Issue Not Found"))
		return
	}

	ticket.ID = id

	w.WriteHeader(http.StatusOK)
	tmpl.ExecuteTemplate(w, "ticket", ticket)
}

func updateTicket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	title := r.FormValue("title")
	description := r.FormValue("description")

	ticket := Ticket{
		ID:          id,
		Title:       title,
		Description: description,
	}

	_, err := tickets.updateTicket(&ticket)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	editTicket(w, r, ps)
}

type SlackMessage struct {
	Response    string        `json:"response_type"`
	Text        string        `json:"text"`
	Attachments []interface{} `json:"attachments"`
}

func newTicket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Print("Error")
		return
	}

	cTime := time.Now().UTC().Format(time.ANSIC)

	user := r.Form["user_id"][0]

	title := r.Form["text"][0]

	ticketHash := sha1.Sum([]byte(cTime + user + title))

	ticketID := hex.EncodeToString(ticketHash[:])

	ticket := Ticket{
		ID:          ticketID,
		User:        r.Form["user_id"][0],
		Title:       r.Form["text"][0],
		Description: "Describe ticket here",
		Created:     cTime,
	}

	_, err = tickets.addTicket(&ticket)
	if err != nil {
		fmt.Printf("Error Adding Ticket to DB: %s\n", err)
	}

	go sendTicketCreatedMessage(r.Form["response_url"][0], &ticket)

	w.WriteHeader(http.StatusOK)
}

func sendTicketCreatedMessage(msgURL string, ticket *Ticket) {

	responseText := "Ticket \"" + ticket.Title + "\" created by <@" + ticket.User + ">."

	ticketURL := baseURL + port + "ticket/" + ticket.ID

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
