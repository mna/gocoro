// Package gocoro implements the same API and features as the Lua coroutines in pure
// go.
//
// See http://www.lua.org/pil/9.html for details on the Lua coroutines.
package gocoro

import (
	"errors"
	"fmt"
)

var (
	// Common errors returned by the coroutine
	ErrEndOfCoro    = errors.New("coroutine terminated")
	ErrInvalidState = errors.New("coroutine is in invalid state")
	ErrCancel       = errors.New("coroutine canceled")
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
	fn      Fn               // The function to run as a coro
	rsm     chan interface{} // The resume synchronisation channel
	yld     chan interface{} // The yield synchronisation channel
	status  Status           // The current status of the coro
	started bool             // Whether or not the coro has started
	err     error            // The last error
}

// The Yielder interface is to be used only from inside a coroutine's
// function.
type Yielder interface {
	// Yield sends the specified values to the caller of the coro, and
	// returns any values sent to the next call to Resume().
	// This is the equivalent of `coroutine.yield()` in Lua.
	Yield(...interface{}) interface{}
}

// The Caller interface is to be used anywhere where a coro needs to be
// called.
type Caller interface {
	// Resume (re)starts the coroutine and returns the values yielded by this run,
	// or an error. This is the equivalent of `coroutine.resume()` in Lua.
	Resume(...interface{}) (interface{}, error)
	// Status returns the current status of the coro. This is the equivalent of
	// `coroutine.status()` in Lua.
	Status() Status
	// Cancel kills the coroutine. Because Go leaks dangling goroutines (and a goroutine
	// is used internally to implement the coro), it must be explicitly killed if it
	// is not to be used again, unlike in Lua where coroutines are eventually garbage-collected.
	// http://stackoverflow.com/questions/3642808/abandoning-coroutines
	Cancel() error
}

// Internal constructor for a coroutine, used to create all coroutine structs.
func newCoroutine(fn Fn) *coroutine {
	// Use as little initial memory as possible, zero value other fields
	return &coroutine{
		fn: fn,
	}
}

// Public constructor of a coroutine Caller. The matching Yielder will automatically
// be given to the function once the coro is started. This is equivalent to
// `coroutine.create()` in Lua.
func New(fn Fn) Caller {
	return newCoroutine(fn)
}

// Public constructor of an Iterator coroutine.
// Cannot be canceled, should be drained or goroutine will leak
// This is equivalent to `coroutine.wrap()` in Lua.
func NewIter(fn Fn) <-chan interface{} {
	c := newCoroutine(fn)
	ch := make(chan interface{})
	go c.iter(ch)
	return ch
}

// Implements the iterator behaviour by looping over all values returned by the coro
// and sending them over the channel used to iterate.
func (c *coroutine) iter(ch chan interface{}) {
	var (
		i   interface{}
		err error
	)
	for i, err = c.Resume(); err == nil; i, err = c.Resume() {
		ch <- i
	}
	close(ch)
	if err != ErrEndOfCoro {
		// That's the downside of the iterator version, cannot return errors
		// if we want to allow for x := range NewIter(y)
		panic(err)
	}
}

// Executes the coroutine function and catches any error, and returns the final
// return value.
func (c *coroutine) run(args ...interface{}) {
	// set status as running, now that the coro goroutine is running.
	c.status = StRunning
	// Start the goroutine that runs the actual coro function.
	go func() {
		var i interface{}
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(error); !ok {
					// Turn the panic into an error type if it isn't
					c.err = fmt.Errorf("%s", err)
				} else {
					c.err = e
				}
			}
			// Return the last value and die
			c.status = StDead
			c.Yield(i)
		}()

		// Trap the return value, and in the defer, yield it like any normally Yielded value.
		i = c.fn(c, args...)
	}()
}

// Returns the current status of the coro.
func (c *coroutine) Status() Status {
	return c.status
}

// Resumes (or starts) execution of the coro.
func (c *coroutine) Resume(args ...interface{}) (interface{}, error) {
	switch c.status {
	case StSuspended:
		if !c.started {
			// Never started, so create the channels and run the coro.
			c.started = true
			c.rsm = make(chan interface{})
			c.yld = make(chan interface{})
			c.run(args...) // run sets the status as Running
		} else {
			// Restart, so simply set status back to Running and unblock the waiting
			// goroutine by sending on the resume channel.
			c.status = StRunning
			if len(args) == 1 {
				// Slightly more convenient: send the one and only arg instead of a
				// slice of empty interfaces.
				c.rsm <- args[0]
			} else {
				c.rsm <- args
			}
		}
	case StDead:
		// Resume on a Dead coro returns an error (either EndOfCoro, or the previous error
		// that caused the coro to die).
		if c.err == nil {
			c.err = ErrEndOfCoro
		}
		return 0, c.err
	default:
		// Any other state is invalid to call Resume on.
		return 0, ErrInvalidState
	}

	// Wait for a yield
	i := <-c.yld
	return i, c.err
}

// Cancels execution of a coro. Can only be called on suspended coros,
// returns an error otherwise.
func (c *coroutine) Cancel() error {
	if c.status != StSuspended {
		return ErrInvalidState
	}
	if c.started {
		// Signal the end by closing the resume channel
		close(c.rsm)
		// Wait for confirmation
		<-c.yld
	} else {
		// Coro was never started, so simply set its status to Dead.
		c.status = StDead
	}
	return nil
}

// Yields execution to the caller, sending values along the way.
func (c *coroutine) Yield(args ...interface{}) interface{} {
	// Yield is called from within the func. It sets the status to Suspended,
	// unless the coro is dying (Yield from a call to Cancel).
	isDead := c.status == StDead
	if !isDead {
		c.status = StSuspended
	}
	// Send the value
	if len(args) == 1 {
		// Slightly more convenient: if only one arg, send the one arg instead of a slice
		// of empty interfaces. Easier for type assertions from the caller.
		c.yld <- args[0]
	} else {
		c.yld <- args
	}
	if !isDead {
		// Wait for resume
		if i, ok := <-c.rsm; ok {
			return i
		}
		// c.rsm is closed, cancel by panicking, will be caught in c.run's defer statement.
		panic(ErrCancel)
	}
	return nil
}
