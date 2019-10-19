package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

var app = cli.NewApp()

func info() {
	app.Name = "Tollgate IO CLI Client"
	app.Author = "Michael Yankelev"
	app.Version = "1.0.0"
}

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
	info()
	var endpoints = getDataSources()
	fmt.Println("Please select an endpoint:")
	for _, element := range endpoints {
		fmt.Println(element.Id, " - ", element.Url)
	}

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	fmt.Printf("You selected %s", text)
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
