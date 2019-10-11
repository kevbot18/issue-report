package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Ticket object template
type Ticket struct {
	ID          string `json:"id,omitempty" db:"id"`
	User        string `json:"user,omitempty" db:"createdBy"`
	Title       string `json:"title,omitempty" db:"title"`
	Description string `json:"description,omitempty" db:"description"`
	Created     string `json:"created,omitempty" db:"createdAt"`
}

// getTicket
// returns json object with:
// @returns json:
//	id: string id of ticket
//  user: string name of user
//	title: string title of ticket
//	description: description of title
//	created: string of date submitted
func getTicket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ticket, err := tickets.getTicketByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}
	jsonData, err := json.Marshal(ticket)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}

// updateTicket
// POST request
// request:
// id: string
// title: string
// description: string
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}

// SlackMessage contains the fields used to respond to the slack message.
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

func getAllTickets(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	list, err := tickets.getAllTickets()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	jsonData, err := json.Marshal(list)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
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
