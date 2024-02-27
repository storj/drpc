package main

import (
	"reflect"
	"testing"
	"unsafe"
)

var C any

type CookiePool struct {
	Cookie *Cookie
}

type Cookie struct {
	A, B, C, D, E, F, G, H, I string
	Z                         [10]byte
}

func BenchmarkAllocate(b *testing.B) {
	var cookie *CookiePool
	in1 := reflect.TypeOf(cookie)

	for j := 0; j < b.N; j++ {
		for i := 0; i < 1000; i++ {
			C = reflect.New(in1.Elem()).Interface()
		}
	}

	b.Logf("size %d", in1.Size())
	b.Logf("unsafe size %d", unsafe.Sizeof(cookie))
	b.Logf("unsafe size %d", unsafe.Sizeof(reflect.New(in1.Elem())))
	b.Logf("Cookie: %+v, %t", C, C)
}
