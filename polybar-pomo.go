package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	TomatoEmoji  = "\U0001F345"        // Emoji representation for work status
	RestEmoji    = "\U0001F3D6"        // Emoji representation for rest status
	PauseEmoji   = "\U000023F8"        // Emoji representation for pause status
	WorkDuration = 25 * time.Minute    // Duration for work period
	RestDuration = 5 * time.Minute     // Duration for rest period
	SocketPath   = "/tmp/polybar-pomo" // Unix socket path
)

// PomodoroStatus represents the status of the pomodoro timer
type PomodoroStatus int

const (
	Work PomodoroStatus = iota
	Rest
)

// PomodoroState holds the state of the pomodoro timer
type PomodoroState struct {
	Status PomodoroStatus
	Timer  *time.Timer
	End    time.Time
	Ticker *time.Ticker
	Paused bool
	mu     sync.Mutex
}

// NewPomodoro initializes a new PomodoroState instance with given status and pause state
func NewPomodoro(status PomodoroStatus, paused bool) *PomodoroState {
	duration := GetDuration(status)
	state := &PomodoroState{
		Status: status,
		Timer:  time.NewTimer(duration),
		Ticker: time.NewTicker(time.Second),
		End:    time.Now().Add(duration),
		Paused: paused,
	}

	if paused {
		state.Timer.Stop()
	} else {
		state.Ticker.Stop()
	}
	return state
}

// String returns a formatted string representing the pomodoro timer status
func (state *PomodoroState) String() string {
	state.mu.Lock()
	defer state.mu.Unlock()

	var suffix string
	if state.Paused {
		suffix = PauseEmoji
	} else if state.Status == Work {
		suffix = TomatoEmoji
	} else {
		suffix = RestEmoji
	}

	elapsedTime := state.End.Sub(time.Now()).Round(time.Second)
	minutes := int(elapsedTime.Minutes())
	seconds := int(elapsedTime.Seconds()) - 60*minutes

	return fmt.Sprintf("%s %02d:%02d", suffix, minutes, seconds)
}

// Update continuously monitors the PomodoroState, responding to signals from the Timer and Ticker channels.
func (state *PomodoroState) Update() {
	for {
		select {
		case <-state.Timer.C:
			state.Toggle()
		case <-state.Ticker.C:
			state.mu.Lock()
			state.End = state.End.Add(1 * time.Second)
			state.mu.Unlock()
		}
	}
}

// Pause toggles the paused state of the pomodoro timer
func (state *PomodoroState) Pause() {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Paused {
		state.Ticker.Stop()
		state.Timer.Reset(state.End.Sub(time.Now()))
	} else {
		state.Ticker.Reset(1 * time.Second)
		state.Timer.Stop()
	}
	state.Paused = !state.Paused
}

// Toggle toggles the pomodoro timer between work and rest status
func (state *PomodoroState) Toggle() {
	state.mu.Lock()
	defer state.mu.Unlock()

	nextStatus := (state.Status + 1) % 2
	duration := GetDuration(nextStatus)

	state.Status = nextStatus
	state.Timer.Reset(duration)
	state.End = time.Now().Add(duration)
}

// GetDuration returns the duration for the given pomodoro status
func GetDuration(status PomodoroStatus) time.Duration {
	return map[PomodoroStatus]time.Duration{
		Work: WorkDuration,
		Rest: RestDuration,
	}[status]
}

// HandleRequest handles incoming requests over the Unix socket connection
func HandleRequest(conn *net.UnixConn, state *PomodoroState) {
	buffer := make([]byte, 128)

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}
	message := strings.TrimSpace(strings.ToLower(string(buffer[:n])))

	switch message {
	case "pause":
		state.Pause()
	case "toggle":
		state.Toggle()
	}
}

func main() {
	// Create a new PomodoroState instance with initial status
	state := NewPomodoro(Work, false)

	// Start a new goroutine to continuously update the PomodoroState.
	go state.Update()

	// Remove existing socket file if it exists
	if err := os.RemoveAll(SocketPath); err != nil {
		fmt.Println("Error removing socket file:", err.Error())
		return
	}

	// Attempt to listen to the Unix socket
	listener, err := net.ListenUnix("unix", &net.UnixAddr{SocketPath, "unix"})
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer listener.Close()

	// Goroutine function to handle incoming Unix socket connections
	go func() {
		for {
			conn, err := listener.AcceptUnix()
			if err != nil {
				fmt.Println("Error accepting connection:", err.Error())
				return
			}
			go HandleRequest(conn, state)
		}
	}()

	// Main loop to display pomodoro timer status
	for {
		statusStr := state.String()
		fmt.Println(statusStr)
		time.Sleep(1 * time.Second)
	}
}
