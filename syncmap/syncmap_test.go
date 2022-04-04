package syncmap

import (
	"fmt"
	"github.com/hq-cml/go-tools/syncmap/sync"
	"testing"
)

func Test_map(t *testing.T) {
	m := sync.Map{}
	m.Store("A", 10)
	m.Store("B", 10)
	m.Store("A", 20)
	fmt.Println(m.Load("A"))
	fmt.Println(m.Load("A"))
	m.Store("A", 30)
	m.Store("C", 40)
	m.Delete("X")
	v, ok := m.LoadAndDelete("C")
	_, _ = v, ok
}

type M struct {
	i int
	m map[string]int
}

func F(m M) {
	m.i = 200
	m.m["a"] = 100
	fmt.Println("A------", m)
}

func Test_1(t *testing.T) {
	m1 := M{
		i: 2,
		m: map[string]int{},
	}
	m1.m["a"] = 1
	fmt.Println(m1)

	F(m1)
	fmt.Println(m1)
}
