package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	TomatoEmoji = "\U0001F345"        // Emoji representation for work status
	RestEmoji   = "\U0001F3D6"        // Emoji representation for rest status
	PauseEmoji  = "\U000023F8"        // Emoji representation for pause status
	SocketPath  = "/tmp/polybar-pomo" // Unix socket path
)

var (
	WorkDuration time.Duration
	RestDuration time.Duration
)

// PomodoroStatus represents the status of the pomodoro timer
type PomodoroStatus int

const (
	Work PomodoroStatus = iota
	Rest
)

// PomodoroState holds the state of the pomodoro timer
type PomodoroState struct {
	End    time.Time
	Paused bool
	Status PomodoroStatus
	Ticker *time.Ticker
	Timer  *time.Timer
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
	}
	return state
}

// String returns a formatted string representing the pomodoro timer status
func (state *PomodoroState) String() string {
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

// Pause toggles the paused state of the pomodoro timer
func (state *PomodoroState) Pause() {
	if state.Paused {
		state.Timer.Reset(state.End.Sub(time.Now()))
	} else {
		state.Timer.Stop()
	}
	state.Paused = !state.Paused
}

// Toggle toggles the pomodoro timer between work and rest status
func (state *PomodoroState) Toggle() {
	nextStatus := (state.Status + 1) % 2
	duration := GetDuration(nextStatus)

	state.Status = nextStatus
	state.Timer.Reset(duration)
	state.End = time.Now().Add(duration)
}

// Inc increments the pomodoro timer by the given amount
func (state *PomodoroState) Inc(increment time.Duration) {
	remainingTime := state.End.Sub(time.Now()) + increment
	state.End = state.End.Add(increment).Round(time.Second)
	if !state.Paused {
		state.Timer.Reset(remainingTime)
	}
}

// GetDuration returns the duration for the given pomodoro status
func GetDuration(status PomodoroStatus) time.Duration {
	return map[PomodoroStatus]time.Duration{
		Work: WorkDuration,
		Rest: RestDuration,
	}[status]
}

// HandleRequest handles incoming requests over the Unix socket connection
func HandleRequest(conn *net.UnixConn, pauseChannel, toggleChannel chan struct{}, incChannel chan time.Duration) {
	buffer := make([]byte, 128)

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}
	message := strings.TrimSpace(strings.ToLower(string(buffer[:n])))

	switch message {
	case "pause":
		pauseChannel <- struct{}{}
	case "toggle":
		toggleChannel <- struct{}{}
	case "inc":
		incChannel <- +5 * time.Second
	case "dec":
		incChannel <- -5 * time.Second
	}
}

func main() {
	// Parse CMD arguments
	wFlag := flag.Int("w", 25, "Work Period Duration")
	rFlag := flag.Int("r", 5, "Rest Period Duration")
	flag.Parse()

	// Set Work and Rest Time Perimeters
	WorkDuration = time.Duration(*wFlag) * time.Minute
	RestDuration = time.Duration(*rFlag) * time.Minute

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
	defer os.Remove(SocketPath)

	// Goroutine function to handle incoming Unix socket connections
	pauseChannel := make(chan struct{})
	toggleChannel := make(chan struct{})
	incChannel := make(chan time.Duration)

	go func() {
		for {
			conn, err := listener.AcceptUnix()
			if err != nil {
				fmt.Println("Error accepting connection:", err.Error())
				return
			}
			go HandleRequest(conn, pauseChannel, toggleChannel, incChannel)
		}
	}()

	// Create a new PomodoroState instance with initial status
	state := NewPomodoro(Work, true)

	var inc time.Duration

	// Main loop to update state and display pomodoro time
	for {
		select {
		case <-state.Ticker.C:
			if state.Paused {
				state.Inc(1 * time.Second)
			}
		case <-state.Timer.C:
			if !state.Paused {
				state.Toggle()
			}
		case <-pauseChannel:
			state.Pause()
		case <-toggleChannel:
			state.Toggle()
		case inc = <-incChannel:
			state.Inc(inc)
		}
		statusStr := state.String()
		fmt.Println(statusStr)
	}
}
