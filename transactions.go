package main

import (
	"database/sql"
	"io/ioutil"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// TicketsDB stores pointer to DB and name of DB
// Path is name of database (path)
type TicketsDB struct {
	Path string
	DB   *sql.DB
}

// NewTicketsDB creates a new TicketsDB
func NewTicketsDB(dbPath string, setupSQL ...string) (TicketsDB, error) {
	tmpDB, err := sql.Open(dbType, dbPath)
	if err != nil {
		panic("Cannot open database " + dbPath)
	}
	tDB := TicketsDB{
		Path: dbPath,
		DB:   tmpDB,
	}

	if err := tDB.migrateDB(setupSQL...); err != nil {
		return tDB, err
	}

	return tDB, nil
}

// MigrateDB runs SQL files on database
func (t TicketsDB) migrateDB(sqlFiles ...string) error {
	for _, sqlFile := range sqlFiles {
		setupScript, err := ioutil.ReadFile(sqlFile)
		if err != nil {
			return err
		}

		_, err = t.DB.Exec(string(setupScript))
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAllTickets returns array of all tickets
// Only Title and ID are filled in
func (t TicketsDB) getAllTickets() ([]Ticket, error) {

	sql := "SELECT id, title FROM tickets"

	rows, err := t.DB.Query(sql)
	if err != nil {
		panic(err)
	}

	defer rows.Close()

	ticketList := make([]Ticket, 0)
	for rows.Next() {
		ticket := Ticket{}
		if err := rows.Scan(&ticket.ID, &ticket.Title); err != nil {
			return nil, err
		}
		ticketList = append(ticketList, ticket)
	}

	return ticketList, nil
}

// returns a ticket object
// if ticket was not found, Ticket will be missing it's ID
// returns error if something goes wrong
func (t TicketsDB) getTicketByID(ID string) (Ticket, error) {
	sql := "SELECT title, description, createdBy, createdAt FROM tickets WHERE id = (?)"

	stmt, err := t.DB.Prepare(sql)

	ticket := Ticket{}
	if err != nil {
		return ticket, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(ID)
	if err != nil {
		panic(err)
	}

	// Only care about first result, searching by key anyway
	isFirst := true
	for rows.Next() && isFirst {
		isFirst = false
		err = rows.Scan(&ticket.Title, &ticket.Description, &ticket.User, &ticket.Created)
		if err != nil {
			return ticket, err
		}
		rows.Close()
		ticket.ID = ID
	}

	return ticket, nil
}

func (t TicketsDB) updateTicket(ticket *Ticket) (int64, error) {
	sql := "UPDATE tickets SET title = (?), description = (?) WHERE id = (?)"

	stmt, err := t.DB.Prepare(sql)

	if err != nil {
		return -1, err
	}

	defer stmt.Close()

	result, err := stmt.Exec(ticket.Title, ticket.Description, ticket.ID)

	if err != nil {
		return -1, err
	}

	return result.RowsAffected()
}

func (t TicketsDB) addTicket(ticket *Ticket) (int64, error) {
	sql := "INSERT INTO tickets(id, title, description, createdAt, createdBy) VALUES(?, ?, ?, ?, ?)"

	stmt, err := t.DB.Prepare(sql)

	if err != nil {
		return -1, err
	}

	defer stmt.Close()

	result, err := stmt.Exec(ticket.ID, ticket.Title, ticket.Description, ticket.Created, ticket.User)

	if err != nil {
		return -1, err
	}

	return result.LastInsertId()
}
