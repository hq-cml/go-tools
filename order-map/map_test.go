package order_map

import (
	"fmt"
	"testing"
)

func TestOrderedMap_String(t *testing.T) {
	// Init new OrderedMap
	om := NewOrderedMap()

	// Set key
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)
	om.Set("d", 4)

	// Same interface as builtin map
	if val, ok := om.Get("b"); ok == true {
		// Found key "b"
		fmt.Println(val)
	}

	fmt.Println(om)

	revIter := om.RevIterFunc()
	fmt.Println("RevIter:")
	for kv, ok := revIter(); ok; kv, ok = revIter() {
		fmt.Print(kv, " ")
	}
	fmt.Println()
	fmt.Println()

	fmt.Println("Delete c")
	// Delete a key
	om.Delete("c")

	// Failed Get lookup becase we deleted "c"
	if _, ok := om.Get("c"); ok == false {
		// Did not find key "c"
		fmt.Println("c not found")
	}

	fmt.Println(om)
	revIter = om.RevIterFunc()
	fmt.Println("RevIter:")
	for kv, ok := revIter(); ok; kv, ok = revIter() {
		fmt.Print(kv, " ")
	}
	fmt.Println()
}

func TestOrderedMap_nil(t *testing.T) {
	// Init new OrderedMap
	om := NewOrderedMap()

	// Set key
	om.Set("a", 1)

	// Same interface as builtin map
	if val, ok := om.Get("b"); ok == true {
		// Found key "b"
		fmt.Println(val)
	} else {
		fmt.Println("b not found")
	}

	fmt.Println(om)

	revIter := om.RevIterFunc()
	fmt.Println("RevIter:")
	for kv, ok := revIter(); ok; kv, ok = revIter() {
		fmt.Print(kv, " ")
	}
	fmt.Println()
	fmt.Println()

	fmt.Println("Delete a")
	// Delete a key
	om.Delete("a")

	// Failed Get lookup becase we deleted "c"
	if _, ok := om.Get("c"); ok == false {
		// Did not find key "c"
		fmt.Println("c not found")
	}

	fmt.Println(om)
	revIter = om.RevIterFunc()
	fmt.Println("RevIter:")
	for kv, ok := revIter(); ok; kv, ok = revIter() {
		fmt.Print(kv, " ")
	}
	fmt.Println()
}
