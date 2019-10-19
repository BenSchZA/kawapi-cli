package main

import (
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

type Endpoint struct {
	Id      string
	Url     string
	Address string
}

func getDataSources() []Endpoint {
	resp, err := http.Get("http://localhost:8080/datasets")
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var endpoints []Endpoint
	json.Unmarshal(body, &endpoints)

	return endpoints
}

func main() {
	info()
	var endpoints = getDataSources()
	fmt.Println(endpoints)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
