package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

/**
While there are two common solutions to handling third party API rate limiting,

I have only demonstrated throttling
*/

const MaxRequestsPerMinute = 1000 // Ttird-party rate limit

// controls the rate of outgoing requests
type RateLimiter struct {
	requests     int
	requestChan  chan *UserRequest
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// represents a user's request to the third-party API
type UserRequest struct {
	UserID   string
	Data     string
	Response chan *APIResponse
}

// represents the response from the third-party API
type APIResponse struct {
	Data string
	Err  error
}

// initializes the RateLimiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		// buffered channel to handle 10,000 requests. We know each reqeust can't stay more than 5 secs in the channel
		//(which is the worst case i.e our internal timeout as set in the http handler below)
		// so we can be sure no request will be left in channel indefinitely.
		requestChan: make(chan *UserRequest, 10000),
		// shutdownChan -> carries signal to gracefully shutdown the rate limiter(not really necessary for our usecase
		// but I think it's standard for cleanup for instance say we want to perform some operations on user requests left in requestChan)
		shutdownChan: make(chan struct{}),
	}
	rl.wg.Add(1)
	go rl.processQueue()
	return rl
}

// handles sending requests to the third-party API
func (rl *RateLimiter) processQueue() {
	defer rl.wg.Done()
	ticker := time.NewTicker(time.Minute / time.Duration(MaxRequestsPerMinute))
	defer ticker.Stop()

	// We keep processing requests until signaled by the calling go routine to close queue(i.e shutdownChan).
	for {
		select {
		case <-rl.shutdownChan:
			return
		case req := <-rl.requestChan:
			<-ticker.C
			rl.sendRequest(req)
		}
	}
}

// sends the request to the third-party API with retry logic
func (rl *RateLimiter) sendRequest(req *UserRequest) {
	var (
		maxRetries = 5
		backoff    = time.Millisecond * 500
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// simulate third-party API call
		resp, err := callThirdPartyAPI(req)

		if err == nil {
			// successful response
			req.Response <- resp
			return
		}

		if err == ErrRateLimited {
			// wait for backoff before retrying
			log.Printf("Retry in %f seconds", backoff.Seconds())
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * math.Pow(2, float64(attempt)))
			continue
		} else {
			// Other errors
			req.Response <- &APIResponse{Err: err}
			return
		}
	}

	// If all retries failed
	req.Response <- &APIResponse{Err: fmt.Errorf("request failed after %d retries", maxRetries)}
}

// csimulates the third-party API call
func callThirdPartyAPI(req *UserRequest) (*APIResponse, error) {
	// simulate rate limiting error randomly (30% chance of error)

	if rand.Float32() < 0.3 {
		return nil, ErrRateLimited
	}

	// simulate successful response
	return &APIResponse{Data: fmt.Sprintf("Processed data for user %s", req.UserID)}, nil
}

// Error to indicate that the request was rate-limited
var ErrRateLimited = fmt.Errorf("rate limited by third-party API")

// allows users to submit requests to the RateLimiter
func (rl *RateLimiter) SubmitRequest(req *UserRequest) {
	rl.requestChan <- req
}

// Shutdown gracefully shuts down the RateLimiter
func (rl *RateLimiter) Shutdown() {
	close(rl.shutdownChan)
	rl.wg.Wait()
}

func main() {
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Shutdown()

	// simulate incoming user requests
	http.HandleFunc("/api/request", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// create a UserRequest
		req := &UserRequest{
			UserID:   userID,
			Data:     "Some data",
			Response: make(chan *APIResponse, 1),
		}

		// submit the request to the RateLimiter
		rateLimiter.SubmitRequest(req)

		// Wait for the response or timeout
		select {
		case resp := <-req.Response:
			if resp.Err != nil {
				// handle errors gracefully
				if resp.Err == ErrRateLimited {
					http.Error(w, "Service is busy, please try again later.", http.StatusTooManyRequests)
				} else {
					http.Error(w, resp.Err.Error(), http.StatusInternalServerError)
				}
			} else {
				// Successful response
				fmt.Fprintf(w, "Success: %s", resp.Data)
			}
		case <-time.After(5 * time.Second):
			// Timeout
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		}
	})

	// Start the HTTP server
	log.Println("Server is running on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
