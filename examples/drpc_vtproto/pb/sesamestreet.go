package pb

import (
	"storj.io/drpc"
)

var _ drpc.VTProtoMessage = &Cookie{}

func (m *Cookie) FromVTPool() interface{} {
	mm := CookieFromVTPool()
	m = mm
	return m
}
