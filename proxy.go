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

	. "github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/trinary"
	"github.com/gorilla/mux"
	// "github.com/didip/tollbooth"
	// "github.com/throttled/throttled"
	"golang.org/x/time/rate"
	"github.com/boltdb/bolt"
)

var endpoint = os.Getenv("API")

//https://www.alexedwards.net/blog/how-to-rate-limit-http-requests

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the the map to hold values of the type visitor.
var visitors = make(map[string]*visitor)
var mtx sync.Mutex

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	go cleanupVisitors()
}

func addVisitor(ip string) *rate.Limiter {
	limiter := rate.NewLimiter(2, 5)
	mtx.Lock()
	// Include the current time when creating a new visitor.
	visitors[ip] = &visitor{limiter, time.Now()}
	mtx.Unlock()
	return limiter
}

func getVisitor(ip string) *rate.Limiter {
	mtx.Lock()
	v, exists := visitors[ip]
	if !exists {
		mtx.Unlock()
		return addVisitor(ip)
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	mtx.Unlock()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		mtx.Lock()
		for ip, v := range visitors {
			if time.Now().Sub(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
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

var db *bolt.DB
var router *mux.Router

func main() {
	var err error
	db, err = bolt.Open("store.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("APIS"))
		if err != nil {
			return fmt.Errorf("Create bucket: %s", err)
		}
		return nil
	})
	defer db.Close()
	seed_db(db)

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("APIS"))
		v := b.Get([]byte("a"))
		fmt.Printf("Value for key 'a': %s\n", v)
		return nil
	})

	router = mux.NewRouter()
	router.HandleFunc("/balance/{address}", get_balance)
	router.HandleFunc("/endpoint/{id}/{path:.*}", proxy_handler)

	err = http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}

func get_balance(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	api, err := ComposeAPI(HTTPClientSettings{URI: endpoint})
  must(err)
    
	// GetNewAddress retrieves the first unspent from address through IRI
	// The 100 argument represents only fully confirmed transactions
	address := trinary.Trytes(vars["address"])
	log.Println(address)

	balances, err := api.GetBalances(trinary.Hashes{address}, 100)
	must(err)
	log.Println("\nBalance:", balances.Balances[0], " - According to milestone", balances.MilestoneIndex)
}

func proxy_handler(w http.ResponseWriter, req *http.Request) {
	limiter := getVisitor(req.RemoteAddr)
	if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
	}
	
	vars := mux.Vars(req)
	id := vars["id"]
	path := vars["path"]

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

	log.Println("Getting", path, "from API with ID", id)

	req.Host = ""
	req.URL.Path = path

	p.ServeHTTP(w, req)
}

func must(err error) {
	if err != nil {
			panic(err)
	}
}
