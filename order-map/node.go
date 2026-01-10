package order_map

import "fmt"

type node struct {
	Prev  *node
	Next  *node
	Value interface{}
}

func newRootNode() *node {
	root := &node{}
	root.Prev = root
	root.Next = root
	return root
}

func newNode(prev *node, next *node, key interface{}) *node {
	return &node{Prev: prev, Next: next, Value: key}
}

type KVPair struct {
	Key   interface{}
	Value interface{}
}

func (k *KVPair) String() string {
	return fmt.Sprintf("%v:%v", k.Key, k.Value)
}

func (kv1 *KVPair) Compare(kv2 *KVPair) bool {
	return kv1.Key == kv2.Key && kv1.Value == kv2.Value
}
