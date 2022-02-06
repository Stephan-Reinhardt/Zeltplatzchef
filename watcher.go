package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	createIfNotExists = `CREATE TABLE IF NOT EXISTS watcher (
email varchar(300) NOT NULL,
created timestamp NOT NULL,
PRIMARY KEY (email)
)`
)

type Watcher struct {
	Email   string    `json:"email"`
	Created time.Time `json:"created"`
}

func CreateWatcher(watcher *Watcher) error {
	_, err := db.Query(
		"INSERT INTO watcher(email,created) VALUES ($1,$2)",
		watcher.Email, watcher.Created)
	return err
}

func GetWatcher() ([]*Watcher, error) {
	rows, err := db.Query("SELECT email, created FROM watcher")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var watchers []*Watcher
	for rows.Next() {
		watcher := &Watcher{}
		if err := rows.Scan(&watcher.Email, &watcher.Created); err != nil {
			return nil, err
		}
		watchers = append(watchers, watcher)
	}
	return watchers, nil
}

func NotifyMeHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Println(fmt.Errorf("Error: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	watcher := Watcher{}
	watcher.Email = r.Form.Get("email")
	watcher.Created = time.Now().UTC()

	dbErr := CreateWatcher(&watcher)
	if dbErr != nil {
		fmt.Println(dbErr)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func CreateWatcherTable(db *sql.DB) error {
	infoLogger := log.New(os.Stdout, "db: ", log.LstdFlags)
	errorLogger := log.New(os.Stderr, "db: ", log.LstdFlags)
	result, err := db.Exec(createIfNotExists)
	if err != nil {
		errorLogger.Printf("Error creating watcher table", err)
	} else {
		infoLogger.Println("Table created", result)
	}

	return err
}
