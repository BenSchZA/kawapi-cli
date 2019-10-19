package main

import(
	"os"
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

func main() {
	remote, err := url.Parse("https://alpha-api-nightly.mol.ai")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	router := mux.NewRouter()
	
	router.HandleFunc("/balance/{address}", get_balance)
	router.PathPrefix("/endpoint/{id}/{path:.*}").HandlerFunc(handler(proxy))

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

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		limiter := getVisitor(r.RemoteAddr)
		if limiter.Allow() == false {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
				return
		}
		
		vars := mux.Vars(r)
		id := vars["id"]
		path := vars["path"]

		log.Println("Getting", path, "from API with ID", id)

		r.Host = ""
		r.URL.Path = path

		p.ServeHTTP(w, r)
	}
}

func must(err error) {
	if err != nil {
			panic(err)
	}
}
