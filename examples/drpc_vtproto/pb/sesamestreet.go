package pb

import (
	"storj.io/drpc"
)

var _ drpc.VTProtoMessage = &Cookie{}
var _ drpc.VTProtoMessage = &CookiePool{}

func (m *Cookie) FromVTPool() interface{} {
	mm := CookieFromVTPool()
	m = mm
	return m
}

func (m *CookiePool) FromVTPool() interface{} {
	mm := CookiePoolFromVTPool()
	m = mm
	return m
}
