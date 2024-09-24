package main

import (
	"fmt"
	"sync"
	"time"
)

// represents a user activity event
type Event struct {
	UserID    int
	Timestamp time.Time
	Value     int // sample metric to keep track of(in reality this could be metric like "likes")
}

// represents a time window for aggregations
type Window struct {
	StartTime time.Time
	EndTime   time.Time
	Value     int
}

// handles time-windowed data aggregation
type Aggregator struct {
	mu           sync.Mutex
	windowSize   time.Duration
	userWindows  map[int][]Window
	windowTicker *time.Ticker
}

// initializes the Aggregator
func NewAggregator(windowSize time.Duration) *Aggregator {
	aggr := &Aggregator{
		windowSize:  windowSize,
		userWindows: make(map[int][]Window),
	}
	aggr.startWindowing()
	return aggr
}

// periodically advances the windows
func (a *Aggregator) startWindowing() {
	a.windowTicker = time.NewTicker(a.windowSize)
	go func() {
		for range a.windowTicker.C {
			a.advanceWindows()
		}
	}()
}

// advances the time windows and removes old data
func (a *Aggregator) advanceWindows() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// keep data for the last 24 hours
	// This makes sense say if the standard window size for aggregation is about 1 hour.
	cutoff := time.Now().Add(-24 * time.Hour)
	for userID, windows := range a.userWindows {
		var updatedWindows []Window
		for _, window := range windows {
			if window.EndTime.After(cutoff) {
				updatedWindows = append(updatedWindows, window)
			}
		}
		a.userWindows[userID] = updatedWindows
	}
}

// processes a new event and updates aggregates
func (a *Aggregator) ProcessEvent(event Event) {
	a.mu.Lock()
	defer a.mu.Unlock()

	userWindows := a.userWindows[event.UserID]
	currentWindow := getCurrentWindow(a.windowSize)

	// check if there is an existing window we can update
	var windowUpdated bool
	for i, window := range userWindows {
		if window.StartTime.Equal(currentWindow.StartTime) {
			userWindows[i].Value += event.Value
			windowUpdated = true
			break
		}
	}

	// If no existing window, create a new one
	if !windowUpdated {
		newWindow := Window{
			StartTime: currentWindow.StartTime,
			EndTime:   currentWindow.EndTime,
			Value:     event.Value,
		}
		userWindows = append(userWindows, newWindow)
	}

	a.userWindows[event.UserID] = userWindows
}

// retrieves aggregates for a user
func (a *Aggregator) GetUserAggregates(userID int) []Window {
	a.mu.Lock()
	defer a.mu.Unlock()

	// return a copy to prevent external modification
	userWindows, exists := a.userWindows[userID]
	if !exists {
		return []Window{}
	}

	aggregates := make([]Window, len(userWindows))
	copy(aggregates, userWindows)
	return aggregates
}

// calculates the current time window
func getCurrentWindow(windowSize time.Duration) Window {
	now := time.Now()
	windowStart := now.Truncate(windowSize)
	return Window{
		StartTime: windowStart,
		EndTime:   windowStart.Add(windowSize),
	}
}

func main() {
	windowSize := time.Hour
	aggregator := NewAggregator(windowSize)

	// simulate  events for some set of users
	go func() {
		users := []int{1, 2, 3}
		for {
			for _, userID := range users {
				event := Event{
					UserID:    userID,
					Timestamp: time.Now(),
					Value:     1,
				}
				aggregator.ProcessEvent(event)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// simulate requests for aggregates for one of the users above
	go func() {
		for {
			time.Sleep(30 * time.Second)
			userID := 1
			aggregates := aggregator.GetUserAggregates(userID)
			fmt.Printf("User %d aggregates:\n", userID)
			for _, window := range aggregates {
				fmt.Printf("Window %s - %s: Value = %d\n",
					window.StartTime.Format(time.RFC822),
					window.EndTime.Format(time.RFC822),
					window.Value)
			}
		}
	}()

	select {}
}
