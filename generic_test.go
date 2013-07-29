package gocoro

import (
	"bytes"
	"github.com/bmizerany/assert"
	"strings"
	"testing"
)

func corofng(y Yielder, args ...interface{}) interface{} {
	for len(args) > 0 {
		s := args[0].(string)
		n := args[1].(int)
		args = y.Yield(strings.Repeat(s, n)).([]interface{})
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
