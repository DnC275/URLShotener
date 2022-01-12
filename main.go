package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"hash/fnv"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type UrlsRelation struct {
	ID         uint32 `json:"id,omitempty"`
	ShortUrlID uint32 `json:"short_url_id,omitempty"`
	LongUrl    string `json:"longUrl,omitempty"`
	ShortUrl   string `json:"shortUrl,omitempty"`
}

type MyErrorType int

const (
	NonExistent MyErrorType = iota + 1
)

type MyError struct {
	Type         MyErrorType `json:"-"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

type Storage int

const (
	InMemory Storage = iota + 1
	Postgres
)

type UrlType int

const (
	Short UrlType = iota + 1
	Long
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789"

var db *sql.DB
var err error
var storage Storage
var urlsRelationArr []UrlsRelation

func (s UrlType) String() string {
	types := [...]string{"shortUrlID", "id"}
	if s < Short || s > Long {
		return fmt.Sprintf("UrlType(%d)", int(s))
	}
	return types[s-1]
}

func MakeShortUrlPK() string {
	b := make([]byte, 10)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func getFromDB(urlType UrlType, value uint32) (UrlsRelation, MyError) {
	sqlQuery := fmt.Sprintf("select id, shortUrlId, longUrl, shortUrl from urlrelations where %s = %d",
		fmt.Sprint(urlType), value)

	var result UrlsRelation
	var myErr MyError
	err = db.QueryRow(sqlQuery).
		Scan(&result.ID, &result.ShortUrlID, &result.LongUrl, &result.ShortUrl)
	if err != nil && err != sql.ErrNoRows {
		panic(err)
	} else if err == sql.ErrNoRows {
		myErr.Type = NonExistent
	}
	return result, myErr
}

func getFromMemory(urlType UrlType, id uint32) (UrlsRelation, MyError) {
	var result UrlsRelation
	var myErr MyError
	switch urlType {
	case Long:
		for _, url := range urlsRelationArr {
			if url.ID == id {
				result = url
				break
			}
		}
	case Short:
		for _, url := range urlsRelationArr {
			if url.ShortUrlID == id {
				result = url
				break
			}
		}
	}
	if result.ID == 0 {
		myErr.Type = NonExistent
	}
	return result, myErr
}

func GetById(urlString string, urlType UrlType) (UrlsRelation, MyError) {
	var result UrlsRelation
	var myErr MyError
	h := fnv.New32a()
	h.Write([]byte(urlString))
	switch storage {
	case Postgres:
		result, myErr = getFromDB(urlType, h.Sum32())
	case InMemory:
		result, myErr = getFromMemory(urlType, h.Sum32())
	}
	if myErr.Type == NonExistent {
		if urlType == Short {
			myErr.ErrorMessage = "Non-existent shortUrl"
		} else {
			myErr.ErrorMessage = "Non-existent longUrl"
		}
		return UrlsRelation{}, myErr
	} else if err != nil {
		panic(err)
	}
	return result, MyError{}
}

func RedirectEndpoint(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	res, err := GetById(params["shortUrlPK"], Short)
	if len(err.ErrorMessage) != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(err)
		return
	}

	http.Redirect(w, r, res.LongUrl, 301)
}

func ExpandEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := r.URL.Query()
	res, err := GetById(params.Get("shortUrlPK"), Short)
	if len(err.ErrorMessage) != 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	var response UrlsRelation
	response.LongUrl = res.LongUrl
	json.NewEncoder(w).Encode(response)
}

func CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var tmp UrlsRelation
	_ = json.NewDecoder(r.Body).Decode(&tmp)
	longUrl := tmp.LongUrl

	h := fnv.New32a()
	h.Write([]byte(longUrl))
	id := h.Sum32()
	url, err := GetById(longUrl, Long)
	if len(err.ErrorMessage) != 0 && err.Type != NonExistent {
		panic(err)
	}

	if len(url.ShortUrl) == 0 {
		check := false
		for !check {
			shortUrlPK := MakeShortUrlPK()
			h := fnv.New32a()
			h.Write([]byte(shortUrlPK))

			url.LongUrl = longUrl
			url.ID = id
			url.ShortUrl = fmt.Sprintf("http://127.0.0.1:8000/r/%s", shortUrlPK)
			url.ShortUrlID = h.Sum32()

			_, myErr := GetById(shortUrlPK, Short)
			if len(myErr.ErrorMessage) != 0 && myErr.Type != NonExistent {
				panic(myErr)
			} else if myErr.Type == NonExistent {
				check = true
			}
		}

		switch storage {
		case Postgres:
			_, err := db.Exec("insert into urlrelations (id, shortUrlID, longUrl, shortUrl) values ($1, $2, $3, $4)",
				url.ID, url.ShortUrlID, url.LongUrl, url.ShortUrl)
			if err != nil {
				panic(err)
			}
		case InMemory:
			urlsRelationArr = append(urlsRelationArr, url)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	var response UrlsRelation
	response.ShortUrl = url.ShortUrl
	json.NewEncoder(w).Encode(response)
}

func makeInit(dbHost, dbPort, dbUser, dbPassword, dbName string) {
	if storage == Postgres {
		fmt.Println(dbHost, dbPort, dbName)
		fmt.Println(fmt.Sprint(storage))
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost,
			dbPort,
			dbUser,
			dbPassword,
			dbName)
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			panic(err)
		}

		var tName sql.NullString
		err = db.QueryRow("select table_name from information_schema.tables where table_name = 'urlrelations'").
			Scan(&tName)
		if err != nil && err != sql.ErrNoRows {
			panic(err)
		}

		if !tName.Valid {
			_, err := db.Exec("create table urlrelations (id bigint primary key, shortUrlID bigint, longUrl varchar(200), shortUrl varchar(100))")
			if err != nil {
				log.Fatal("Error creating table in database")
			}
		}
	} else {
		urlsRelationArr = []UrlsRelation{}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var s, dbHost, dbPort, dbUser, dbPassword, dbName string
	flag.StringVar(&s, "storage", "in-memory", "Storage type")
	flag.StringVar(&dbHost, "db_host", os.Getenv("DB_HOST"), "Database host")
	flag.StringVar(&dbPort, "db_port", os.Getenv("DB_PORT"), "Database port")
	flag.StringVar(&dbUser, "db_user", os.Getenv("DB_USER"), "Database user")
	flag.StringVar(&dbPassword, "db_pwd", os.Getenv("DB_PASSWORD"), "Database password for user *user*")
	flag.StringVar(&dbName, "db_name", os.Getenv("DB_NAME"), "Database name")
	flag.Parse()

	if s == "in-memory" {
		storage = InMemory
	} else if s == "postgres" {
		storage = Postgres
	} else {
		log.Fatal("Invalid storage type")
	}

	makeInit(dbHost, dbPort, dbUser, dbPassword, dbName)
	defer db.Close()

	serverPort := os.Getenv("SERVER_PORT")
	router := mux.NewRouter()
	router.HandleFunc("/expand", ExpandEndpoint).Methods("GET")
	router.HandleFunc("/r/{shortUrlPK}", RedirectEndpoint).Methods("GET")
	router.HandleFunc("/create/", CreateEndpoint).Methods("POST")

	log.Printf("Server running on localhost:%s", serverPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", serverPort), router))
}
