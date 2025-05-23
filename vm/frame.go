package vm

import (
	"github.com/ioanzicu/monkeyd/code"
	"github.com/ioanzicu/monkeyd/object"
)

type Frame struct {
	cl          *object.Closure
	ip          int
	basePointer int // frame pointer - for reference while executing a function
	// 1. rest button (clean up the stack) - get rid of a just-executed function
	// 2. serve as a reference for local bindings
}

func NewFrame(cl *object.Closure, basePointer int) *Frame {
	return &Frame{
		cl:          cl,
		ip:          -1,
		basePointer: basePointer,
	}
}

func (f *Frame) Instructions() code.Instructions {
	return f.cl.Fn.Instructions
}
