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

type Issue struct {
	// TODO: Add ID to Issue
	// ID          string
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
	db, err = setupDB("issues.db")
	if err != nil {
		panic(err)
	}

	db.Stats()

	// set up webserver

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
	issues, err := getAllIssuesFromDB(db)
	if err != nil {
		panic(err)
	}
	return c.Render(http.StatusOK, "list", issues)
}

func editReport(c echo.Context) error {
	id := c.Param("id")
	issue, err := getIssueFromDB(db, id)
	if err != nil {
		return c.Redirect(http.StatusNotFound, "/")
	}
	return c.Render(http.StatusOK, "report", issue)
}

func updateReport(c echo.Context) error {
	id := c.Param("id")
	title := c.FormValue("title")
	description := c.FormValue("description")

	issue := Issue{
		Title:       title,
		Description: description,
		User:        "Unchanged",
	}

	_, err := updateIssue(db, id, issue)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	return c.Render(http.StatusOK, "report", issue)
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
		Created:     cTime,
	}

	_, err = addIssueToDB(db, issueID, issue)
	if err != nil {
		fmt.Errorf("Error Adding Issue to DB: %s", err)
	}

	return c.String(http.StatusOK, baseURL+"report/"+issueID)
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

func addIssueToDB(db *sql.DB, id string, issue Issue) (int64, error) {
	sql := "INSERT INTO issues(id, title, description, createdAt, createdBy) VALUES(?, ?, ?, ?, ?)"

	stmt, err := db.Prepare(sql)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	result, err := stmt.Exec(id, issue.Title, issue.Description, issue.Created, issue.User)

	if err != nil {
		panic(err)
	}

	return result.LastInsertId()
}

func updateIssue(db *sql.DB, id string, issue Issue) (int64, error) {
	sql := "UPDATE issues SET title = (?), description = (?) WHERE id = (?)"

	stmt, err := db.Prepare(sql)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	result, err := stmt.Exec(issue.Title, issue.Description, id)

	if err != nil {
		panic(err)
	}

	return result.RowsAffected()
}

func getIssueFromDB(db *sql.DB, id string) (Issue, error) {
	sql := "SELECT title, description, createdBy FROM issues WHERE id = (?)"

	stmt, err := db.Prepare(sql)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(id)

	issue := Issue{}

	err = row.Scan(&issue.Title, &issue.Description, &issue.User)

	if err != nil {
		panic(err)
	}

	return issue, nil
}

func getAllIssuesFromDB(db *sql.DB) (map[string]Issue, error) {
	sql := "SELECT id, title FROM issues"
	rows, err := db.Query(sql)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	issueMap := make(map[string]Issue)
	var ID string
	for rows.Next() {
		issue := Issue{}
		err := rows.Scan(&ID, &issue.Title)

		if err != nil {
			panic(err)
		}
		issueMap[ID] = issue
	}

	return issueMap, nil
}
