package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

type key int

var db *sql.DB

const (
	requestIDKey key = 0
)

var (
	listenAddr string
	healthy    int32

	dbHost     = "localhost"
	dbPort     = 5432
	dbUser     = "postgres"
	dbPassword = "changeme"
	dbName     = "zeltplatzchef"
)

func main() {

	infoLogger, errorLogger := getLogger()

	listenAddr = ":" + getEnv("PORT", "5000")

	infoLogger.Println("Zeltplatz Server startet jetzt.... hoffentlich")

	db = createDbConnection()

	ensureTables()

	router := router()

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      tracing(nextRequestID)(logging(infoLogger)(router)),
		ErrorLog:     errorLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		infoLogger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		err := db.Close()
		if err != nil {
			errorLogger.Fatal("Error closing DB connection", err)
		} else {
			infoLogger.Println("Database Connection is closed")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			errorLogger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	infoLogger.Println("Chef is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errorLogger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	infoLogger.Println("Server stopped")
}

func ensureTables() {
	err := CreateWatcherTable(db)
	if err != nil {
		return
	}
}

func router() *http.ServeMux {
	router := http.NewServeMux()

	router.Handle("/", static())
	router.Handle("/notifyme", NotifyMeHandler())
	router.Handle("/metric", metric())
	return router
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

func static() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path)
		p := "./static" + r.URL.Path
		http.ServeFile(w, r, p)
	})
}

func metric() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "status: OK\n")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "status: ERROR\n")
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
