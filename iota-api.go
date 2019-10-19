package main

import (
	"os"

	. "github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/trinary"
)

var endpoint = os.Getenv("API")

func GetBalance(address trinary.Trytes) uint64 {
	// GetNewAddress retrieves the first unspent from address through IRI
	// The 100 argument represents only fully confirmed transactions
	api, err := ComposeAPI(HTTPClientSettings{URI: endpoint})
	must(err)

	balances, err := api.GetBalances(trinary.Hashes{address}, 100)
	must(err)

	return balances.Balances[0]
}