package ui

import (
	"fmt"
	"sync"
	"time"
)

// Spinner provides a simple progress indicator for long-running operations
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	active   bool
	mu       sync.Mutex
	done     chan bool
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 100 * time.Millisecond,
		done:     make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				// Clear the line
				fmt.Print("\r\033[K")
				return
			default:
				fmt.Printf("\r%s %s", s.frames[i%len(s.frames)], s.message)
				i++
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.done <- true
}

// Success stops the spinner and shows success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Printf("\r\033[K✓ %s\n", message)
}

// Fail stops the spinner and shows error message
func (s *Spinner) Fail(message string) {
	s.Stop()
	fmt.Printf("\r\033[K✗ %s\n", message)
}

// UpdateMessage changes the spinner message while it's running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}
