package main

import(
	"os"
	"fmt"
	"time"
	"sync"
	"log"
	"net/url"
	"net/http"
	"net/http/httputil"

	"github.com/iotaledger/iota.go/trinary"
	"github.com/gorilla/mux"
	// "github.com/didip/tollbooth"
	// "github.com/throttled/throttled"
	"golang.org/x/time/rate"
	"github.com/boltdb/bolt"
)

//https://www.alexedwards.net/blog/how-to-rate-limit-http-requests

// Create a custom session struct which holds the rate limiter for each
// session and the last time that the session was seen.
type session struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	consumer string
	producer string
	paid_value uint64
	expected_value uint64
}

// Change the the map to hold values of the type session.
var sessions = make(map[string]*session)
var mtx sync.Mutex

// Run a background goroutine to remove old entries from the sessions map.
func init() {
	go cleanupSessions()
}

func addSession(ip string) *session {
	limiter := rate.NewLimiter(2, 5)
	mtx.Lock()
	// Include the current time when creating a new session.
	value := session{
		limiter: limiter, 
		lastSeen: time.Now(),
		consumer: "consumer",
		producer: "producer",
		paid_value: 0,
		expected_value: 0,
	}
	sessions[ip] = &value
	mtx.Unlock()
	log.Println("New session:", ip)
	return &value
}

func getSession(ip string) *session {
	mtx.Lock()
	v, exists := sessions[ip]
	if !exists {
		mtx.Unlock()
		return addSession(ip)
	}

	// Update the last seen time for the session.
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

type api struct {
	id  string
	url string
}

var seeds =  []api {
	api {
		id: "a",
		url:  "https://alpha-api-nightly.mol.ai",
	},
	api {
		id: "b",
		url:  "https://google.com",
	},
}

func seed_db(db *bolt.DB) {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("APIS"))
		var err error
		for _, api := range seeds {
			err = b.Put([]byte(api.id), []byte(api.url))
			must(err)
		}
		return err
	})
	must(err)
}

func create_buckets(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("Sessions"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("APIS"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
}

var db *bolt.DB
var router *mux.Router

func main() {
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
		fmt.Printf("Value for key 'a': %s\n", v)
		return nil
	})

	router = mux.NewRouter()
	router.HandleFunc("/balance/{address}", get_balance_handler)
	router.HandleFunc("/endpoint/{id}/{path:.*}", proxy_handler)

	err = http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}

// struct session {
// 	key string
// 	expiry time.Time
// 	consumer string
// 	producer string
// 	paid_value uint64
// 	expected_value uint64
// }
// func purgeExpiredSessions(db *bolt.DB) {
// 	db.Update(func(tx *bolt.Tx) error {
// 		// Assume bucket exists and has keys
// 		b := tx.Bucket([]byte("Sessions"))
// 		c := b.Cursor()
	
// 		now := time.Now()
// 		for k, v := c.First(); k != nil; k, v = c.Next() {
// 			fmt.Printf("key=%s, value=%s\n", k, v)
// 			if v.expiry.After(now) {
// 				fmt.Printf("Purging session:", k)
// 				c.Delete()
// 			}
// 		}
		
// 		return nil
// 	})
// }

func get_balance_handler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	address := trinary.Trytes(vars["address"])
	balance := GetBalance(address)
	log.Println("Balance:", balance)
}

func proxy_handler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	// Use IP for now
	// key := vars["apiKey"] //TODO: we need to generate an API key with consumer seed for session
	id := vars["id"]
	path := vars["path"]
	
	session := getSession(req.RemoteAddr)
	limiter := session.limiter
	if limiter.Allow() == false {
		http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
		return
	}

	producer_balance := GetBalance(os.Getenv("ADDRESS_PRODUCER"))
	log.Println("Producer balance:", producer_balance)

	var p *httputil.ReverseProxy

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("APIS"))
		v := b.Get([]byte(id))
		fmt.Printf("API: %s\n", v)
		remote, err := url.Parse(string(v))
		must(err)
		p = httputil.NewSingleHostReverseProxy(remote)
		return nil
	})

	log.Println("Getting path", path, "from API with ID", id)

	req.Host = ""
	req.URL.Path = path

	p.ServeHTTP(w, req)
}

func must(err error) {
	if err != nil {
			panic(err)
	}
}
