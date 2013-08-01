package gocoro

import (
	"bytes"
	"github.com/bmizerany/assert"
	"strings"
	"testing"
)

// This one receives 2, yields 1
func corofng(y Yielder, args ...interface{}) interface{} {
	for len(args) > 0 {
		s := args[0].(string)
		n := args[1].(int)
		args = y.Yield(strings.Repeat(s, n)).([]interface{})
	}
	return "done"
}

// This one receives 1, yields 2 (except for last return)
func corofng2(y Yielder, args ...interface{}) interface{} {
	s := args[0].(string)
	for len(s) > 0 {
		s = y.Yield(s, len(s)).(string)
	}
	return "done"
}

func TestYieldOneG(t *testing.T) {
	c := New(corofng)
	res, err := c.Resume("a", 3)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "aaa", res)
	assert.Equal(t, StSuspended, c.Status())
}

func TestYieldManyG(t *testing.T) {
	seeds := []string{"a", "b", "c"}
	buf := bytes.NewBuffer(nil)
	c := New(corofng)
	for i, s := range seeds {
		res, err := c.Resume(s, i+1)
		if err != nil {
			panic(err)
		}
		buf.WriteString(res.(string))
	}
	assert.Equal(t, "abbccc", buf.String())
	assert.Equal(t, StSuspended, c.Status())
}

func TestIterG(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	for v := range NewIter(corofng) {
		buf.WriteString(v.(string))
	}
	assert.Equal(t, "done", buf.String())
}

func TestFn2YieldOneG(t *testing.T) {
	c := New(corofng2)
	res, err := c.Resume("test")
	if err != nil {
		panic(err)
	}
	assert.Equal(t, StSuspended, c.Status())
	assert.Equal(t, "test", res.([]interface{})[0])
	assert.Equal(t, 4, res.([]interface{})[1])
}

func TestFn2YieldAllG(t *testing.T) {
	seeds := []string{"test", "tes", "t", ""}
	c := New(corofng2)
	for _, s := range seeds {
		res, err := c.Resume(s)
		if err != nil {
			panic(err)
		}
		switch v := res.(type) {
		case []interface{}:
			assert.Equal(t, len(s), v[1])
		case string:
			assert.Equal(t, "done", v)
		}
	}
	assert.Equal(t, StDead, c.Status())
}
