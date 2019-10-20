package main

import (
	. "github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/converter"
	"github.com/iotaledger/iota.go/trinary"
)

var endpoint = "https://nodes.devnet.thetangle.org"

func GetBalance(address trinary.Trytes) uint64 {
	// GetNewAddress retrieves the first unspent from address through IRI
	// The 100 argument represents only fully confirmed transactions
	api, err := ComposeAPI(HTTPClientSettings{URI: endpoint})
	must(err)

	balances, err := api.GetBalances(trinary.Hashes{address}, 100)
	must(err)

	return balances.Balances[0]
}

func GetTagValue(consumer string, producer string, tag string) uint64 {
	var query = FindTransactionsQuery{
		Tags: []trinary.Trytes{
			bundle.PadTag(trinary.Trytes(tag)),
		},
		Addresses: trinary.Hashes{
			trinary.Trytes(consumer),
			trinary.Trytes(producer),
		},
	}

	api, err := ComposeAPI(HTTPClientSettings{URI: endpoint})
	must(err)

	transactions, err := api.FindTransactionObjects(query)
	must(err)

	var tx_sum int64 = 0
	for _, tx := range transactions {
		// To get our message back we need to convert the signatureMessageFragment to ASCII
		// We should strip all suffix 9's from the signatureMessageFragment, we use a
		// custom function to do this.
		if len(removeSuffixNine(tx.SignatureMessageFragment))%2 == 0 {
			_, err := converter.TrytesToASCII(removeSuffixNine(tx.SignatureMessageFragment))
			must(err)
			//fmt.Println(tx.Address, " / ", tx.Value) tx.Value tx.Tag msg tx.Hash
			tx_sum = tx_sum + tx.Value
		}
	}
	return uint64(tx_sum)
}

func removeSuffixNine(frag string) string {
	fraglen := len(frag)
	var firstNonNineAt int
	for i := fraglen - 1; i > 0; i-- {
		if frag[i] != '9' {
			firstNonNineAt = i
			break
		}
	}
	return frag[:firstNonNineAt+1]
}
