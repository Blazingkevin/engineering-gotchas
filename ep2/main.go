package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

/**
  More Doc To Come ******
*/

// Rate limit settings
const (
	RequestLimit = 5           // the max requests per time window
	TimeWindow   = time.Minute // time window for rate limiting
)

// to hold the visitor's rate limit data
type RateLimiter struct {
	// to ensure thread safe acces to the the `visitors` map
	mu sync.Mutex
	// a map with user-id as key and value as Visitor data
	visitors map[string]*Visitor
	// to simulate storage availability (In reality, central storage like redis can be unavailable. Please check main function to see how this simulation works)
	storageEnabled bool
}

// to track number of requests and last seen time
type Visitor struct {
	lastSeen time.Time
	requests int
}

// initializes the RateLimiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		// storage initially available
		storageEnabled: true,
	}

	// very important!
	// having 100,000 one-time user that never come back to our platform.
	// what is the point of keeping their rate data in our system especially knowing that their next request will surely be outside the time window.
	// It is then necessary to clean up this kind of records otherwise, our central storage can grow infitely large
	// in reality, this can be a separate workload on a different node tasked with just running this sort of clean up
	go rl.cleanupVisitors()
	return rl
}

// toggles the storage availability
func (rl *RateLimiter) SimulateStorageFailure(enable bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.storageEnabled = enable
	if !enable {
		log.Println("Simulating storage unavailability")
	} else {
		log.Println("Storage is now available")
	}
}

// helper function to remove visitors that have not been seen within the time window
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for id, visitor := range rl.visitors {
			if time.Since(visitor.lastSeen) > TimeWindow {
				delete(rl.visitors, id)
			}
		}
		rl.mu.Unlock()
	}
}

// core rate limit checker to check if a user has exceeded the rate limit
func (rl *RateLimiter) Limit(userID string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.storageEnabled {
		// Simulate storage failure ->  Allow request (as a fallback)
		return false, fmt.Errorf("storage unavailable")
	}

	visitor, exists := rl.visitors[userID]
	if !exists {
		rl.visitors[userID] = &Visitor{
			lastSeen: time.Now(),
			requests: 1,
		}
		return false, nil // Not exceeded
	}

	if time.Since(visitor.lastSeen) > TimeWindow {
		visitor.lastSeen = time.Now()
		visitor.requests = 1
		return false, nil // Not exceeded
	}

	visitor.requests++
	visitor.lastSeen = time.Now()
	if visitor.requests > RequestLimit {
		return true, nil //  exceeded
	}

	return false, nil
}

// applies rate limiting to incoming requests
func rateLimiterMiddleware(rl *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			http.Error(w, "X-User-ID header is required", http.StatusBadRequest)
			return
		}

		limited, err := rl.Limit(userID)
		if err != nil {
			// central storage is unavailable; implement graceful degradation
			log.Printf("Storage error: %v", err)
			// allow the request but in an actual system, we should also log the incident
			next.ServeHTTP(w, r)
			return
		}

		if limited {
			retryAfter := int(TimeWindow.Seconds())

			//  It's so important to give the client-side a way to handle this rate limit
			// set Retry-After header to show the the start of the next available time window, set the appropriate error code(429)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// dummy handler to simulate an API endpoint
func apiHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Request successful")
}

func main() {
	rateLimiter := NewRateLimiter()

	mux := http.NewServeMux()
	mux.HandleFunc("/api", apiHandler)

	// apply the rate limiter middleware
	handler := rateLimiterMiddleware(rateLimiter, mux)

	// Below we simulate multiple nodes by running more than one server in a separate go routine
	// you can add as much server as you want. The whole point is to test the behavior of the rate limiter
	// across multiple servers(by making requests, alternating between the ports below).

	// Start the server in a separate goroutine
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
	go func() {
		log.Println("Server1 is running on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server1 failed: %v", err)
		}
	}()

	server2 := &http.Server{
		Addr:    ":8081",
		Handler: handler,
	}
	go func() {
		log.Println("Server2 is running on port 8081")
		if err := server2.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server2 failed: %v", err)
		}
	}()

	// simulating storage unavailability after some time
	go func() {
		for {
			time.Sleep(60 * time.Second)
			rateLimiter.SimulateStorageFailure(false) // disable storage

			// enable storage after some time
			time.Sleep(10 * time.Second)
			rateLimiter.SimulateStorageFailure(true) // enable storage
		}
	}()

	// keeep the main function running
	select {}
}
