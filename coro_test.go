package gocoro

import (
	"github.com/bmizerany/assert"
	"testing"
)

func createCoro() Caller {
	return New(func(y Yielder) int {
		for i := 1; i <= 10; i++ {
			y.Yield(i)
		}
		// Just for fun...
		return 1000
	})
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

	assert.T(t, err == ErrEndOfCoro)
}
