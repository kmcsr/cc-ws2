
package protos_test

import (
	"testing"
	"reflect"

	protos "github.com/kmcsr/cc-ws2/plugin/protos"
)

func TestAnyWrapNil(t *testing.T){
	wv, err := protos.WrapValue(nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	v, err := wv.Unwrap()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if v != nil {
		t.Errorf("Unexpected value: %v, expected nil", v)
	}
}

func TestAnyWrap(t *testing.T){
	type EStruct struct{}
	protos.RegisterType((*EStruct)(nil))
	type TStruct struct {
		A int
		a int
		B float32
		C *string
	}
	protos.RegisterType((*TStruct)(nil))
	type AStruct struct {
		V int
		N any
	}
	protos.RegisterType((*AStruct)(nil))

	var nInt int = 1
	var nUint uint = 2
	var nFloat64 float64 = 3.4
	var sString string = "edf"

	var tests = []struct{ V any; Size int }{
		{int(1), 0},
		{int8(1), 0},
		{int16(1), 0},
		{int32(1), 0},
		{int64(1), 0},
		{uint(1), 0},
		{uint8(1), 0},
		{uint16(1), 0},
		{uint32(1), 0},
		{uint64(1), 0},
		{float32(1), 0},
		{float64(1), 0},
		{complex64(1+2i), 0},
		{complex128(1+2i), 0},
		{(*int)(nil), 0},
		{(*uint)(nil), 0},
		{(*float64)(nil), 0},
		{&nInt, 0},
		{&nUint, 0},
		{&nFloat64, 0},
		{"abc", 0},
		{"\x00", 0},
		{EStruct{}, 0},
		{TStruct{1, 0, 3.4, &sString}, 0},
		{&EStruct{}, 0},
		{&TStruct{1, 0, 3.4, &sString}, 0},
		{AStruct{1, nil}, 0},
		{AStruct{1, 2.3}, 0},
	}
	for _, data := range tests {
		wv, err := protos.WrapValue(data.V)
		if err != nil {
			t.Errorf("Unexpected error when wrapping %T(%v): %v", data.V, data.V, err)
			continue
		}
		t.Logf("Wrapped %T(%v): size=%d", data.V, data.V, len(wv.Value))
		v, err := wv.Unwrap()
		if err != nil {
			t.Errorf("Unexpected error when unwrapping %T(%v): %v", data.V, data.V, err)
			continue
		}
		if !reflect.DeepEqual(v, data.V) {
			t.Errorf("Value changed after unwrap: before=%T(%v), after=%T(%v)", data.V, data.V, v, v)
			continue
		}
	}
}
