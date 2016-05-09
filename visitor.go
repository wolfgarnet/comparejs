package main

import (
	"github.com/robertkrimen/otto/ast"
	"github.com/wolfgarnet/walker"
)

type CompareVisitor struct {
	walker.VisitorImpl
	name string
}

func NewCompareVisitor(name string, signal chan int, tc chan ast.Node, heartBeat chan bool) *CompareVisitor {
	vh := &walker.Hook{}
	vh.OnNode = func(node ast.Node, metadata []walker.Metadata) error {
		heartBeat <- true
		<-signal

		tc <- node
		parent := walker.ParentMetadata(metadata).Node()
		tc <- parent

		return nil
	}
	visitor := &CompareVisitor{}
	visitor.name = name

	visitor.AddHook(vh)

	return visitor
}
