package main

import (
	"errors"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/manifoldco/promptui"
	"github.com/common-nighthawk/go-figure"
)

func getDataSources() []Endpoint {
	resp, err := http.Get("http://localhost:8080/endpoint")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}
	var results []Endpoint
	jsonerr := json.Unmarshal(body, &results)

	if jsonerr != nil {
		log.Fatal(jsonerr)
	}
	return results
}

// func getAPIKey(provider: string) string {

// }

// func getMIOTAValue(address) uint64 {

// }

func main() {
	splash := figure.NewFigure("KawAPI", "", true)
	splash.Print()
	fmt.Printf("\n")

	validate := func(input string) error {
		if len(input) != 81 {
			return errors.New("Seed must have 81 characters, including the checksum")
		}
		return nil
	}

	seed_prompt := promptui.Prompt{
		Label:    "Securely enter your seed",
		Validate: validate,
		Mask:     '*',
	}

	seed, err := seed_prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	var endpoints []Endpoint = getDataSources()
	var endpoint_ids []string

	for _, element := range endpoints {
		endpoint_ids = append(endpoint_ids, fmt.Sprintf("%s ~ %s", element.Id,  element.Url))
	}

	prompt_endpoint := promptui.Select{
		Label: "Select dataset endpoint",
		Items: endpoint_ids,
	}

	_, result_endpoint, err := prompt_endpoint.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("Connecting to endpoint %q...\n", result_endpoint)

	address := GetAddress(seed)
	fmt.Println("\nUsing address:", address)
}

func must(err error) {
    if err != nil {
        panic(err)
    }
}