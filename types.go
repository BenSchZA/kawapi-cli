package main

import (
	"time"

	"golang.org/x/time/rate"
)

type Endpoint struct {
	Id  string
	Url string
	Address string
}

// Create a custom session struct which holds the rate limiter for each
// session and the last time that the session was seen.
type Session struct {
	id string
	limiter  *rate.Limiter
	lastSeen time.Time
	consumer string
	producer string
	initial_value uint64
	paid_value uint64
	expected_value uint64
}