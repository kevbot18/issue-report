package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var (
	port     string
	baseURL  string
	insecure bool

	setupSQL string
	tickets  TicketsDB

	dbType string
	dbPath string
)

func setVars() {
	flag.StringVar(&port, "port", "443", "port to run server on")
	flag.BoolVar(&insecure, "insecure", false, "allow non-https connections")
	flag.StringVar(&baseURL, "url", "127.0.0.1", "base URL of the web server (for Angular frontend)")
	flag.StringVar(&setupSQL, "sql", "setup.sql", "SQL setup script")
	flag.StringVar(&dbType, "db", "sqlite3", "database engine to use")
	flag.StringVar(&dbPath, "dbURL", "tickets.db", "path to DB (file or URL)")

	flag.Parse()

	if port[0] != ':' {
		port = ":" + port
	}

	if !((strings.HasPrefix("http://", baseURL) && insecure) || strings.HasPrefix("https://", baseURL)) {
		if insecure {
			baseURL = "http://" + baseURL
		} else {
			baseURL = "https://" + baseURL
		}
	}

	if strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL[:len(baseURL)-1] + port + "/"
	} else {
		baseURL = baseURL + port + "/"
	}

	if insecure {
		fmt.Println("Running in Insecure Mode.")
	}
	fmt.Printf("Server Root URL at %s using Port %s\n", baseURL, port)
}

func setup() {

	// sets up global variables based on CLI args
	setVars()

	// set up database
	var err error
	if tickets, err = NewTicketsDB(dbPath, setupSQL); err != nil {
		panic("Could Not Create Database")
	}
}

func main() {

	// sets up database
	setup()

	// set up webserver
	router := httprouter.New()

	router.GET("/ticket/:id", getTicket)
	router.POST("/ticket/:id", updateTicket)
	router.POST("/ticket", newTicket)
	router.GET("/tickets", getAllTickets)

	log.Fatal(http.ListenAndServe(port, router))
}
