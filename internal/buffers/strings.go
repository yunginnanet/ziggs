package buffers

import (
	"strings"
	"sync"
)

type pool struct {
	*sync.Pool
}

var Stringers = pool{&sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	}}}

func (p *pool) Get() *strings.Builder {
	return p.Pool.Get().(*strings.Builder)
}

func (p *pool) Put(sb *strings.Builder) {
	sb.Reset()
	p.Pool.Put(sb)
}
