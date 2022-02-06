package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

var db *sql.DB

var (
	dbHost     = "localhost"
	dbPort     = 5432
	dbUser     = "postgres"
	dbPassword = "changeme"
	dbName     = "zeltplatzchef"
)

func main() {

	infoLogger, _ := getLogger()

	port := getEnv("PORT", "5000")

	infoLogger.Println("Starting server on port: " + port)

	db = createDbConnection()

	ensureTables()

	log.Fatal(http.ListenAndServe(":"+port, noTrailingSlash(serve)))
}

func serve(w http.ResponseWriter, r *http.Request) {
	var head string
	head, r.URL.Path = shiftPath(r.URL.Path)
	switch head {
	case "":
		ServeIndex(w, r)
	case "admin":
		ServeAdminIndex(w, r)
	case "static":
		ServeStatic(w, r)
	case "notifyme":
		NotifyMeHandler(w, r)
	default:
		return
	}
}

func noTrailingSlash(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}
}

// shiftPath splits the given path into the first segment (head) and
// the rest (tail). For example, "/foo/bar/baz" gives "foo", "/bar/baz".
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func ensureTables() {
	err := CreateWatcherTable(db)
	if err != nil {
		return
	}
}

func getLogger() (*log.Logger, *log.Logger) {
	infoLog := log.New(os.Stdout, "main: ", log.LstdFlags)
	errorLog := log.New(os.Stderr, "main: ", log.LstdFlags)
	return infoLog, errorLog
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func createDbConnection() *sql.DB {

	infoLogger := log.New(os.Stdout, "db: ", log.LstdFlags)
	errorLogger := log.New(os.Stderr, "db: ", log.LstdFlags)

	psqlInfo := getEnv("DATABASE_URL", fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName))
	infoLogger.Println("Try to establish Database Connection. Details: " + psqlInfo)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		errorLogger.Fatal("Error connecting to Database", err)
	}

	err = db.Ping()
	if err != nil {
		errorLogger.Fatal("Could not ping database.", err)
	}

	fmt.Println("Successfully connected!")

	return db
}
