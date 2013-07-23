// Package gocoro implements the same API and features as the Lua coroutines in pure
// go.
//
// See http://www.lua.org/pil/9.html for details on the Lua coroutines.
package gocoro

import (
	"fmt"
)

var (
	// Common errors returned by the coroutine
	ErrEndOfCoro    = fmt.Errorf("coroutine terminated")
	ErrInvalidState = fmt.Errorf("coroutine is in invalid state")
	ErrCancel       = fmt.Errorf("coroutine canceled")
)

// The status of the coroutine is an "enum"
type Status int

const (
	// Possible values of the status of the coroutine
	StDead      Status = iota - 1
	StSuspended        // Zero value when a coro is created
	StRunning
)

var (
	// Lookup map to pretty-print the status
	statusNms = map[Status]string{
		StDead:      "Dead",
		StSuspended: "Suspended",
		StRunning:   "Running",
	}
)

// Stringer interface implementation
func (s Status) String() string {
	return statusNms[s]
}

// The generic signature of a coro-ready function, in Lua this is built into
// the language via the global coroutine variable, here the Yielder is passed
// as a parameter.
type Fn func(Yielder, ...interface{}) interface{}

// The coroutine struct is private, the outside world only see the contextually
// relevant portions of it, via the Yielder or Caller interfaces.
type coroutine struct {
	fn      Fn            // The function to run as a coro
	rsm     chan struct{} // The resume synchronisation channel
	yld     chan int      // The yield synchronisation channel
	status  Status        // The current status of the coro
	started bool          // Whether or not the coro has started
	err     error         // The last error
}

// The Yielder interface is to be used only from inside a coroutine's
// function.
type Yielder interface {
	// Yield sends the specified values to the caller of the coro, and
	// returns any values sent to the next call to Resume().
	// This is the equivalent of `coroutine.yield()` in Lua.
	Yield(...interface{}) []interface{}
}

// The Caller interface is to be used anywhere where a coro needs to be
// called.
type Caller interface {
	// Resume (re)starts the coroutine and returns the values yielded by this run,
	// or an error. This is the equivalent of `coroutine.resume()` in Lua.
	Resume(...interface{}) ([]interface{}, error)
	// Status returns the current status of the coro. This is the equivalent of
	// `coroutine.status()` in Lua.
	Status() Status
	// Cancel kills the coroutine. Because Go leaks dangling goroutines (and a goroutine
	// is used internally to implement the coro), it must be explicitly killed if it
	// is not to be used again, unlike in Lua where coroutines are eventually garbage-collected.
	// http://stackoverflow.com/questions/3642808/abandoning-coroutines
	Cancel() error
}
