package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
)

const (
	dbHost     = "localhost"
	dbPort     = 5432
	dbUser     = "postgres"
	dbPassword = "changeme"
	dbName     = "zeltplatzchef"

	createIfNotExists = `CREATE TABLE IF NOT EXISTS watcher (
email varchar(300) NOT NULL,
created timestamp NOT NULL,
PRIMARY KEY (email)
)`
)

func OpenDbConnection() *sql.DB {

	infoLogger := log.New(os.Stdout, "db: ", log.LstdFlags)
	errorLogger := log.New(os.Stderr, "db: ", log.LstdFlags)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	infoLogger.Println("Try to establish Database Connection. Details: " + psqlInfo)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		errorLogger.Fatal("Error connecting to Database", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			errorLogger.Fatal("Error closing DB connection", err)
		}
	}(db)

	err = db.Ping()
	if err != nil {
		errorLogger.Fatal("Could not ping database.", err)
	}

	fmt.Println("Successfully connected!")

	createWatcherTable(db)
	return db
}

func createWatcherTable(db *sql.DB) error {
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
