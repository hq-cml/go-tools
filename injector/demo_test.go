package injector

import (
	"fmt"
	"testing"
)

type Test struct {
	Target int `inject:"target"`
}

func (t *Test) Start() error {
	fmt.Println("start", t.Target)
	return nil
}

func (t *Test) Close() {
	fmt.Println("close", t.Target)
}

type Dep struct {
	Test *Test `inject:"test"`
}

func (d *Dep) Close() {
	fmt.Println("close Dep", d.Test)
}

func Test_Demo1(t *testing.T) {
	InitDefault()
	//dep.Close, test.Close will be called orderly
	defer Close()

	Reg("target", 123)

	//test will be auto created, test.Start will be called, then dep.Start(if any)
	dep := Reg("dep", (*Dep)(nil)).(*Dep)
	fmt.Println("find dep", dep)
}
