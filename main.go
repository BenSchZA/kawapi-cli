package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
	"context"

	"github.com/gorilla/mux"
	"github.com/iotaledger/iota.go/trinary"
	// "github.com/didip/tollbooth"
	// "github.com/throttled/throttled"
	"github.com/boltdb/bolt"
	"github.com/common-nighthawk/go-figure"
	"golang.org/x/time/rate"

	"github.com/paulbellamy/ratecounter"
	"google.golang.org/grpc"
	pb "helloworld"
)

var txPrice uint64 = 1
var txBuffer uint64 = 10
var txRate rate.Limit = 2
var txBurst int = 5

const (
	address     = "0.0.0.0:50051"
	defaultName = "world"
)

//https://www.alexedwards.net/blog/how-to-rate-limit-http-requests

// Change the the map to hold values of the type Session.
var sessions = make(map[string]*Session)
var mtx sync.Mutex

// Run a background goroutine to remove old entries from the sessions map.
func init() {
	go cleanupSessions()
}

func addSession(ip string, consumer string, producer string) *Session {
	limiter := rate.NewLimiter(txRate, txBurst)
	mtx.Lock()
	// Include the current time when creating a new Session.
	value := Session{
		id:             ip,
		limiter:        limiter,
		lastSeen:       time.Now(),
		consumer:       consumer,
		producer:       producer,
		initial_value:  GetTagValue(consumer, producer, "VALTEST"), //TODO: set tag
		paid_value:     0,
		expected_value: 0,
	}
	sessions[ip] = &value
	mtx.Unlock()
	log.Println("New Session:", ip)
	return &value
}

func getSession(ip string, consumer string, producer string) *Session {
	mtx.Lock()
	v, exists := sessions[ip]
	if !exists {
		mtx.Unlock()
		return addSession(ip, consumer, producer)
	}

	// Update the last seen time for the Session.
	v.lastSeen = time.Now()
	mtx.Unlock()
	return v
}

// Every minute check the map for sessions that haven't been seen for
// more than 10 minutes and delete the entries.
func cleanupSessions() {
	for {
		time.Sleep(time.Minute)
		mtx.Lock()
		for ip, v := range sessions {
			if time.Now().Sub(v.lastSeen) > 10*time.Minute {
				delete(sessions, ip)
			}
		}
		mtx.Unlock()
	}
}

func validateTransaction(session *Session) bool {
	session.expected_value = session.expected_value + txPrice
	session.paid_value = GetTagValue(session.consumer, session.producer, "VALTEST") - session.initial_value //TODO: set tag
	sessions[session.id] = session
	if session.expected_value-session.paid_value > txBuffer {
		return false
	} else {
		return true
	}
}

var seeds = []Endpoint{
	Endpoint{
		Id:      "molecule",
		Url:     "https://alpha-api-nightly.mol.ai",
		Address: "FMYHLHBSJJMJZNPVUOKDCUSFOPQAGPBSPOPMFVBGXUUDFPEWPXREZFQKGKSNHZWDMODRDYWIXQT9CLVBXGPANCSYBW",
	},
	Endpoint{
		Id:      "google",
		Url:     "https://google.com",
		Address: "FMYHLHBSJJMJZNPVUOKDCUSFOPQAGPBSPOPMFVBGXUUDFPEWPXREZFQKGKSNHZWDMODRDYWIXQT9CLVBXGPANCSYBW",
	},
}

func seed_db(db *bolt.DB) {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("APIS"))
		var err error
		for _, api := range seeds {
			encoded, err_json := json.Marshal(api)
			must(err_json)
			log.Println("Seeding:", api.Id, api)

			err = b.Put([]byte(api.Id), encoded)
			must(err)
		}
		return err
	})
	must(err)
}

func create_buckets(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Sessions"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("APIS"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
}

var db *bolt.DB
var router *mux.Router

func determineListenAddress() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return "", fmt.Errorf("$PORT not set")
	}
	return ":" + port, nil
}

var counter = ratecounter.NewRateCounter(60 * time.Second)

func startGrpc() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
	// Record an event happening
	counter.Incr(1)
}

func testGrpc() {
	goTicker := time.NewTicker(1 * time.Nanosecond)
	logTicker := time.NewTicker(10 * time.Second)
	for {
		select {
			case <-goTicker.C:
				go startGrpc()
			case <-logTicker.C:
				rate := counter.Rate() / 60
				fmt.Printf("%d\n", rate)
		} 
	}
}

func main() {
	splash := figure.NewFigure("KawAPI", "", true)
	splash.Print()

	addr, err_port := determineListenAddress()
	must(err_port)

	var err error
	db, err = bolt.Open("store.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize datastore
	create_buckets(db)
	seed_db(db)

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("APIS"))
		v := b.Get([]byte("a"))
		var data *Endpoint
		json.Unmarshal(v, &data)
		log.Println("Successfully seeded APIs:", data)
		return nil
	})

	router = mux.NewRouter()

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	router.Handle("/", http.FileServer(http.Dir("./static")))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.HandleFunc("/endpoint", get_endpoints_handler)
	router.HandleFunc("/balance/{address}", get_balance_handler)
	router.HandleFunc("/{token}/endpoint/{id}/{path:.*}", proxy_handler)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	server.ListenAndServe()
}

func get_balance_handler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	address := trinary.Trytes(vars["address"])
	balance := GetBalance(address)
	log.Println("Balance:", balance)
}

func get_endpoints_handler(w http.ResponseWriter, req *http.Request) {
	var endpoints []Endpoint
	log.Println("getting endpoints")

	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte("APIS"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var apiData Endpoint
			if err := json.Unmarshal(v, &apiData); err != nil {
				return err
			}
			endpoints = append(endpoints, apiData)
		}
		return nil
	})

	body, err := json.Marshal(endpoints)

	if err != nil {

	}
	w.Write([]byte(body))
}

func proxy_handler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	// Use IP for now
	// key := vars["apiKey"] //TODO: we need to generate an API key with consumer seed for Session
	token := vars["token"]
	id := vars["id"]
	path := vars["path"]

	var p *httputil.ReverseProxy
	var apiData *Endpoint

	err_endpoint := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("APIS"))
		v := b.Get([]byte(id))

		if err := json.Unmarshal(v, &apiData); err != nil {
			return err
		}

		log.Println("Endpoint:", apiData)

		remote, err := url.Parse(string(apiData.Url))
		must(err)
		p = httputil.NewSingleHostReverseProxy(remote)

		return nil
	})
	if err_endpoint != nil {
		http.Error(w, http.StatusText(404), http.StatusNotFound)
		return
	}

	session := getSession(
		token, //req.RemoteAddr,
		"JXBIEWEBYCZOKBHIGDXT9VNLUTGCZGXJLCSAUTCRGEEHFETHRIVMTBNKGPQUXNVSCLIWEKHWFBASGYFLWZOGJE9YPX",
		apiData.Address,
	)

	diff := int64(session.paid_value - session.expected_value)
	log.Println("Session", token, "outstanding value:", diff)

	validTX := validateTransaction(session)
	if validTX {
		log.Println("Valid TX:", session.id)
	} else {
		http.Error(w, http.StatusText(402), http.StatusPaymentRequired)
		log.Println("Payment required:", session.id)
		return
	}

	limiter := session.limiter
	if limiter.Allow() == false {
		http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
		log.Println("Rate limit exceeded:", session.id)
		return
	}

	log.Println("Getting path", path, "from Endpoint with ID", id)
	req.Host = ""
	req.URL.Path = path
	p.ServeHTTP(w, req)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
