package main

import (
	"errors"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/manifoldco/promptui"
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

func main() {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New("Seed must have 81 characters, including the checksum")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Securely enter your seed",
		Validate: validate,
		Mask:     '*',
	}

	_, err := prompt.Run()

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
}