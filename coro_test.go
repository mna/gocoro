package gocoro

import (
	"fmt"
	"github.com/bmizerany/assert"
	"testing"
)

var panicAt = 0

func corofn(y Yielder) int {
	for i := 1; i <= 10; i++ {
		if i == panicAt {
			panic("gulp")
		}
		y.Yield(i)
	}
	// Just for fun...
	return 1000
}

func createCoro() Caller {
	panicAt = 0
	return New(corofn)
}

func createIter() <-chan int {
	panicAt = 0
	return NewIter(corofn)
}

func TestInitialStatus(t *testing.T) {
	c := createCoro()
	assert.Equal(t, StSuspended, c.Status())
}

func TestYieldOne(t *testing.T) {
	c := createCoro()
	i, err := c.Resume()
	assert.T(t, err == nil)
	assert.Equal(t, StSuspended, c.Status())
	assert.Equal(t, 1, i)
}

func TestYieldMany(t *testing.T) {
	c := createCoro()
	c.Resume()
	c.Resume()
	i, err := c.Resume()

	assert.T(t, err == nil)
	assert.Equal(t, StSuspended, c.Status())
	assert.Equal(t, 3, i)
}

func TestCancelBeforeStart(t *testing.T) {
	c := createCoro()
	err := c.Cancel()
	assert.T(t, err == nil)
	assert.Equal(t, StDead, c.Status())
}

func TestCancelAfterSome(t *testing.T) {
	c := createCoro()
	c.Resume()
	c.Resume()
	err := c.Cancel()

	assert.T(t, err == nil)
	assert.Equal(t, StDead, c.Status())
}

func TestYieldAll(t *testing.T) {
	var err error
	c := createCoro()
	for _, err = c.Resume(); err == nil; _, err = c.Resume() {
	}

	assert.Equal(t, ErrEndOfCoro, err)
}

func TestResumeAfterAll(t *testing.T) {
	c := createCoro()
	for _, err := c.Resume(); err == nil; _, err = c.Resume() {
	}
	_, err := c.Resume()
	assert.Equal(t, ErrEndOfCoro, err)
}

func TestCancelAfterAll(t *testing.T) {
	c := createCoro()
	for _, err := c.Resume(); err == nil; _, err = c.Resume() {
	}
	err := c.Cancel()
	assert.Equal(t, ErrInvalidState, err)
}

func TestPanicInFn(t *testing.T) {
	var err error
	c := createCoro()
	panicAt = 3
	cnt := 0
	for _, err = c.Resume(); err == nil; _, err = c.Resume() {
		cnt++
	}
	assert.Equal(t, fmt.Errorf("gulp"), err)
	assert.Equal(t, 2, cnt)
}

func TestIter(t *testing.T) {
	c := createIter()
	cnt := 0
	for _ = range c {
		cnt++
	}
	assert.Equal(t, 11, cnt)
}
