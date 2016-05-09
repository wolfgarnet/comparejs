package main

import (
	"bufio"
	"fmt"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/parser"
	"github.com/wolfgarnet/walker"
	"os"
	"reflect"
	"strings"
	"time"
)

var snippetLength int = 50
var useNewLines bool = true
var proceedOnBlocks = true
var displayIntermediate = false

func main() {
	args := os.Args[1:]

	var first string
	var second string

	for _, a := range args {
		if strings.HasPrefix(a, "--first=") {
			first = a[8:]
		}

		if strings.HasPrefix(a, "--second=") {
			second = a[9:]
		}
	}

	program1, err := parser.ParseFile(nil, first, nil, parser.StoreComments)
	if err != nil {
		panic("Error while reading first: " + err.Error())
	}

	program2, err := parser.ParseFile(nil, second, nil, parser.StoreComments)
	if err != nil {
		panic("Error while reading second: " + err.Error())
	}

	c1 := make(chan int)
	c2 := make(chan int)

	typeChannel1 := make(chan ast.Node)
	typeChannel2 := make(chan ast.Node)

	heartBeat1 := make(chan bool)
	heartBeat2 := make(chan bool)

	compare1 := NewCompareVisitor("1", c1, typeChannel1, heartBeat1)
	compare2 := NewCompareVisitor("2", c2, typeChannel2, heartBeat2)

	walker1 := walker.NewWalker(compare1)
	walker2 := walker.NewWalker(compare2)

	go func() {
		walker1.Begin(program1)
		heartBeat1 <- false
	}()

	go func() {
		walker2.Begin(program2)
		heartBeat2 <- false
	}()

	var node1 ast.Node
	var node2 ast.Node
	var parent1 ast.Node
	var parent2 ast.Node
	var last1 ast.Node
	var last2 ast.Node
	var poll1 bool = true
	var poll2 bool = true

	// Get first
	if !checkHeartBeat(heartBeat1, heartBeat2) {
		panic("What the....?")
	}
	c1 <- 1
	c2 <- 2

	running := true
	for running {
		noAccidents := true

		for noAccidents {
			if poll1 {
				last1 = node1
				node1 = <-typeChannel1
				parent1 = <-typeChannel1
			}
			if poll2 {
				last2 = node2
				node2 = <-typeChannel2
				parent2 = <-typeChannel2
			}

			e, t := Comparator(node1, node2)

			if !checkHeartBeat(heartBeat1, heartBeat2) {
				running = false
				break
			}

			switch e {
			case Same:
				c1 <- 1
				c2 <- 2

				poll1 = true
				poll2 = true

			case NotSameType:
				v := OnNotSameType(node1, node2)
				switch v {
				case Node1:
					c1 <- 1
					poll1 = true
					poll2 = false

				case Node2:
					c2 <- 2
					poll1 = false
					poll2 = true

				default:
					fmt.Printf("Not the same type: %v\n", t)
					noAccidents = false
				}

			case NotSameValue:
				fmt.Printf("Not the same value: %v\n", t)
				noAccidents = false
			}
		}

		if displayIntermediate && noAccidents {
			fmt.Printf("Snippet 1 %v: %v\n", reflect.TypeOf(node1), DisplaySnippet(program1, node1))
			fmt.Printf("Snippet 2 %v: %v\n", reflect.TypeOf(node2), DisplaySnippet(program2, node2))
			fmt.Print("---------------------------------------------------------------------\n")
		}

		// Only do selection if still running
		if running {
			fmt.Print("\n")
			if parent1 != nil && parent2 != nil && false {
				fmt.Printf("Parent 1: %v: %v\n", reflect.TypeOf(parent1), DisplaySnippet(program1, parent1))
				fmt.Printf("Parent 2: %v: %v\n", reflect.TypeOf(parent2), DisplaySnippet(program2, parent2))
			}

			fmt.Printf("Snippet 1: %v: %v\n", reflect.TypeOf(node1), DisplaySnippet(program1, node1))
			fmt.Printf("Snippet 2: %v: %v\n", reflect.TypeOf(node2), DisplaySnippet(program2, node2))

			done := false
			for !done {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Select: ")
				text, _ := reader.ReadString('\n')
				text = strings.TrimSpace(text)

				switch text {
				case "1":
					c1 <- 1
					poll1 = true
					poll2 = false
					done = true

				case "2":
					c2 <- 2
					poll1 = false
					poll2 = true
					done = true

				case "3":
					c1 <- 3
					c2 <- 4
					poll1 = true
					poll2 = true
					done = true

				case "d1":
					fmt.Printf("[1] Snippet from %v - %v\n", reflect.TypeOf(node1), first)
					fmt.Printf("%v\n\n", getSnippet(program1, node1))

				case "l1":
					fmt.Printf("[1] Snippet from last node %v - %v\n", reflect.TypeOf(last1), first)
					fmt.Printf("%v\n\n", getSnippet(program1, last1))

				case "p1":
					fmt.Printf("[1] Parent snippet %v - %v\n", reflect.TypeOf(parent1), first)
					fmt.Printf("%v\n\n", getSnippet(program1, parent1))

				case "d2":
					fmt.Printf("[2] Snippet from %v - %v\n", reflect.TypeOf(node2), second)
					fmt.Printf("%v\n\n", getSnippet(program2, node2))

				case "l2":
					fmt.Printf("[2] Snippet from last node %v - %v\n", reflect.TypeOf(last2), second)
					fmt.Printf("%v\n\n", getSnippet(program2, last2))

				case "p2":
					fmt.Printf("[2] Parent snippet %v - %v\n", reflect.TypeOf(parent2), second)
					fmt.Printf("%v\n\n", getSnippet(program2, parent2))

				default:
					fmt.Printf("Please select one of the following:\n")
					fmt.Printf(" '1' - Advance node 1\n")
					fmt.Printf(" '2' - Advance node 2\n")
					fmt.Printf(" '3' - Advance node 1 AND node 2\n")
					fmt.Printf(" 'd1' - Display node 1\n")
					fmt.Printf(" 'd2' - Display node 2\n")
					fmt.Printf(" 'p1' - Display parent 1\n")
					fmt.Printf(" 'p2' - Display parent 2\n")
					fmt.Printf(" 'l1' - Display last node 1\n")
					fmt.Printf(" 'l2' - Display last node 2\n")
				}
			}

			fmt.Printf("\n")
		}
	}
}

func checkHeartBeat(hb1, hb2 chan bool) bool {
	b1 := false
	b2 := false
	check := 0
	for check < 2 {
		select {
		case beat := <-hb1:
			if beat {
				b1 = true
			}
		case beat := <-hb2:
			if beat {
				b2 = true
			}
		case <-time.After(time.Millisecond * 100):
		}

		check++
	}

	if !b1 {
		fmt.Printf("Program 1 is done!\n")
	}
	if !b2 {
		fmt.Printf("Program 2 is done!\n")
	}

	return b1 && b2
}

type Proceed int

const (
	Node1 Proceed = iota
	Node2
	None
)

func OnNotSameType(node1, node2 ast.Node) Proceed {
	switch node1.(type) {
	case *ast.EmptyStatement:
		return Node1

	case *ast.BlockStatement:
		if proceedOnBlocks {
			return Node1
		}
	}

	switch node2.(type) {
	case *ast.EmptyStatement:
		return Node2

	case *ast.BlockStatement:
		if proceedOnBlocks {
			return Node2
		}
	}

	return None
}

func getSnippet(program *ast.Program, node ast.Node) string {
	snippet := program.File.Source()[node.Idx0()-1 : node.Idx1()-1]
	position := program.File.Position(node.Idx0())
	return fmt.Sprintf("%v,%v: \"%v\"", position.Line, position.Column, snippet)
}

func DisplaySnippet(program *ast.Program, node ast.Node) string {
	var snippet string
	switch node.(type) {
	case *ast.Program, *ast.FunctionLiteral, *ast.FunctionStatement, *ast.BlockStatement, *ast.ExpressionStatement, *ast.CallExpression:
		diff := node.Idx1() - node.Idx0()
		end := node.Idx1() - 1
		if int(diff) > snippetLength {
			end = node.Idx0() + file.Idx(snippetLength)
		}
		snippet = program.File.Source()[node.Idx0()-1:end] + " ..."

	default:
		snippet = program.File.Source()[node.Idx0()-1 : node.Idx1()-1]
	}

	position := program.File.Position(node.Idx0())
	if !useNewLines {
		snippet = strings.Replace(snippet, "\n", " ", -1)
	}
	return fmt.Sprintf("%v,%v: \"%v\"", position.Line, position.Column, snippet)
}

type Equality int

const (
	NotSameType Equality = iota
	NotSameValue
	Same
)

func (e Equality) String() string {
	switch e {
	case NotSameType:
		return "Not the same type"

	case NotSameValue:
		return "Not the same value"

	case Same:
		return "The same"
	}

	return "Waat!?"
}

func Comparator(node1, node2 ast.Node) (Equality, string) {
	//fmt.Printf("Comparing %v and %v\n", reflect.TypeOf(node1), reflect.TypeOf(node2))
	if reflect.TypeOf(node1) != reflect.TypeOf(node2) {
		return NotSameType, fmt.Sprintf("%v and %v", reflect.TypeOf(node1), reflect.TypeOf(node2))
	}

	switch n1 := node1.(type) {
	case *ast.AssignExpression:
		n2, _ := node2.(*ast.AssignExpression)
		if n1.Operator != n2.Operator {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Operator, n2.Operator)
		}

	case *ast.BinaryExpression:
		n2, _ := node2.(*ast.BinaryExpression)
		if n1.Operator != n2.Operator {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Operator, n2.Operator)
		}

	case *ast.BooleanLiteral:
		n2, _ := node2.(*ast.BooleanLiteral)
		if n1.Literal != n2.Literal {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Literal, n2.Literal)
		}

	case *ast.Identifier:
		n2, _ := node2.(*ast.Identifier)
		if n1.Name != n2.Name {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Name, n2.Name)
		}

	case *ast.NumberLiteral:
		n2, _ := node2.(*ast.NumberLiteral)
		if n1.Value != n2.Value {
			if n1.Literal != n2.Literal {
				return NotSameValue, fmt.Sprintf("%v and %v", n1.Literal, n2.Literal)
			}
		}

	case *ast.ObjectLiteral:
		n2, _ := node2.(*ast.ObjectLiteral)
		if len(n1.Value) != len(n2.Value) {
			return NotSameValue, fmt.Sprintf("%v and %v", len(n1.Value), len(n2.Value))
		}
		for i, _ := range n1.Value {
			p1 := n1.Value[i]
			p2 := n2.Value[i]
			if p1.Key != p2.Key {
				return NotSameValue, fmt.Sprintf("KEY %v and %v", p1.Key, p2.Key)
			}
			if p1.Kind != p2.Kind {
				return NotSameValue, fmt.Sprintf("KIND %v and %v", p1.Kind, p2.Kind)
			}
		}

	case *ast.Program:
		n2, _ := node2.(*ast.Program)

		if len(n1.Body) != len(n2.Body) {
			return NotSameValue, fmt.Sprintf("%v and %v", len(n1.Body), len(n2.Body))
		}

	case *ast.RegExpLiteral:
		n2, _ := node2.(*ast.RegExpLiteral)
		if n1.Literal != n2.Literal {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Literal, n2.Literal)
		}

	case *ast.StringLiteral:
		n2, _ := node2.(*ast.StringLiteral)
		if n1.Value != n2.Value {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Value, n2.Value)
		}

	case *ast.VariableExpression:
		n2, _ := node2.(*ast.VariableExpression)
		if n1.Name != n2.Name {
			return NotSameValue, fmt.Sprintf("%v and %v", n1.Name, n2.Name)
		}

	default:

	}

	return Same, ""
}
