package injector

import (
	"fmt"
	"reflect"
	"testing"
)

type MyT struct {
}

func Test_getTypeName(t *testing.T) {
	refType := reflect.TypeOf(MyT{})
	fmt.Println(getTypeName(refType))
	refType = reflect.TypeOf(&MyT{})
	fmt.Println(getTypeName(refType))
}
