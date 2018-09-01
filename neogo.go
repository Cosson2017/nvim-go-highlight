// Copyright 2014 Paul Jolly <paul@myitcv.org.uk>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO this is very alpha
// look how we ignore all the errors below
// very bad. very very bad

package main

import (
	"fmt"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"
	"github.com/neovim/go-client/nvim"
)

type Neogo struct {
	*nvim.Nvim
	ch chan struct{}
}

var neogo *Neogo
var fDebugAST bool
var fDebug bool

func Serve(c *nvim.Nvim) error {

	neogo = &Neogo{}
	neogo.Nvim = c
	neogo.ch = make(chan struct{})
	go neogo.parseBuffer()

	return nil
}

func (n *Neogo) Shutdown() error {
	return nil
}

func (n *Neogo) BufferUpdate(o *neovim.MethodOptionParams) error {
	n.ch <- struct{}{}
	return nil
}

func getUint64(i interface{}) uint64 {
	switch i := i.(type) {
	case int64:
		return uint64(i)
	case int:
		return uint64(i)
	case uint64:
		return i
	default:
		panic("Type not supported")
	}
}

func (n *Neogo) parseBuffer() {
	// Consume events, parse and send back commands to highlight
	sg := NewSynGenerator()
	for {
		select {
		case <-n.ch:
			cb, _ := n.c.CurrentBuffer()
			bn, _ := n.c.BufferName(cb)
			bc, _ := n.c.BufferLines(cb, 0, -1, false)
			src := []byte(strings.Join(bc, "\n"))

			viewPortI, err := n.c.Eval("[winsaveview()['topline'], winsaveview()['topline'] + winheight('%'), winsaveview()['leftcol'], winsaveview()['leftcol'] + winwidth('%')]")
			viewPort := viewPortI.([]interface{})
			sg.lStart = getUint64(viewPort[0])
			sg.lEnd = getUint64(viewPort[1])
			sg.cStart = getUint64(viewPort[2])
			sg.cEnd = getUint64(viewPort[3])

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, bn, src, parser.AllErrors|parser.ParseComments)
			if f == nil && err != nil {
				fmt.Println("We got an error on the parse")
			}

			if fDebugAST {
				ast.Print(fset, f)
			}

			// TODO better way? Do we really need to reparse each time?
			sg.fset = fset
			sg.f = f

			// generate our highlight positions
			ast.Walk(sg, f)

			for _, c := range f.Comments {
				ast.Walk(sg, c)
			}

			// set the highlights
			sg.sweepMap(n)
		}
	}
}

type position struct {
	l    int
	line int
	col  int
	t    nodeType
}

type action uint32

type nodeType uint32

const (
	_ADD action = iota
	_KEEP
	_DELETE
)

const (
	_KEYWORD nodeType = iota
	_STATEMENT
	_STRING
	_TYPE
	_CONDITIONAL
	_FUNCTION
	_COMMENT
	_LABEL
	_REPEAT
)

func (n nodeType) String() string {
	switch n {
	case _KEYWORD:
		return "Keyword"
	case _STATEMENT:
		return "Statement"
	case _STRING:
		return "String"
	case _TYPE:
		return "Type"
	case _CONDITIONAL:
		return "Conditional"
	case _FUNCTION:
		return "Function"
	case _COMMENT:
		return "Comment"
	case _LABEL:
		return "Label"
	case _REPEAT:
		return "Repeat"
	default:
		panic("Unknown const mapping")
	}
	return ""
}

type match struct {
	id uint64
	a  action
}

type viewport struct {
	lStart, lEnd, cStart, cEnd uint64
}

type synGenerator struct {
	fset  *token.FileSet
	f     *ast.File
	nodes map[position]*match
	viewport
}

func NewSynGenerator() *synGenerator {
	res := &synGenerator{
		nodes: make(map[position]*match),
	}
	return res
}

func (s *synGenerator) sweepMap(n *Neogo) {
	buff, _ :=  n.c.CurrentBuffer()
	for pos, m := range s.nodes {
		switch m.a {
		case _ADD:
			//com := fmt.Sprintf("matchaddpos('%v', [[%v,%v,%v]])", pos.t, pos.line, pos.col, pos.l)
			//id, _ := n.c.Eval(com)
			n.c.AddBufferHighlight(buff, 0, pos.t.String(), pos.line, pos.col, pos.col + pos.l)
			switch id := id.(type) {
			case uint64:
				m.id = id
			case int64:
				m.id = uint64(id)
			}
			m.a = _DELETE
		case _DELETE:
			if m.id == 0 {
				// this match never got added
				continue
			}
			com := fmt.Sprintf("matchdelete(%v)", m.id)
			n.c.Eval(com)
			if fDebug {
				fmt.Printf("%v\n", com)
			}
			delete(s.nodes, pos)
		case _KEEP:
			m.a = _DELETE
		}
	}
}

func (s *synGenerator) addNode(t nodeType, l int, _p token.Pos) {
	p := s.fset.Position(_p)
	if uint64(p.Line) < s.lStart || uint64(p.Line) > s.lEnd {
		return
	}
	pos := position{t: t, l: l, line: p.Line, col: p.Column}
	if m, ok := s.nodes[pos]; ok {
		// when we call add, we mark the match as delete
		// for efficiency next time around, hence the need
		// to mark this as keep
		m.a = _KEEP
	} else {
		// we leave anything that needs to be deleted
		// and add a new match, with action == _ADD
		s.nodes[pos] = &match{a: _ADD}
	}
}

func (s *synGenerator) Visit(node ast.Node) ast.Visitor {
	var handleType func(ast.Expr)
	handleType = func(t ast.Expr) {
		switch node := t.(type) {
		case *ast.Ident:
			s.addNode(_TYPE, len(node.Name), node.NamePos)
		case *ast.FuncType:
			s.addNode(_KEYWORD, 4, node.Func)
		case *ast.ChanType:
			s.addNode(_TYPE, 4, node.Begin)
			// TODO add highligthing of chan arrow?
			handleType(node.Value)
		case *ast.MapType:
			s.addNode(_TYPE, 3, node.Map)
			handleType(node.Key)
			handleType(node.Value)
		}
	}
	switch node := node.(type) {
	case *ast.File:
		s.addNode(_STATEMENT, 7, node.Package)
	case *ast.BasicLit:
		if node.Kind == token.STRING {
			s.addNode(_STRING, len(node.Value), node.ValuePos)
		}
	case *ast.Comment:
		s.addNode(_COMMENT, len(node.Text), node.Slash)
	case *ast.GenDecl:
		switch node.Tok {
		case token.VAR:
			s.addNode(_KEYWORD, 3, node.TokPos)
		case token.IMPORT:
			s.addNode(_STATEMENT, 6, node.TokPos)
		case token.CONST:
			s.addNode(_KEYWORD, 5, node.TokPos)
		case token.TYPE:
			s.addNode(_KEYWORD, 4, node.TokPos)
		}
	case *ast.StructType:
		s.addNode(_KEYWORD, 6, node.Struct)
	case *ast.InterfaceType:
		s.addNode(_KEYWORD, 9, node.Interface)
	case *ast.ReturnStmt:
		s.addNode(_KEYWORD, 6, node.Return)
	case *ast.BranchStmt:
		s.addNode(_KEYWORD, len(node.Tok.String()), node.TokPos)
	case *ast.ForStmt:
		s.addNode(_REPEAT, 3, node.For)
	case *ast.GoStmt:
		s.addNode(_STATEMENT, 2, node.Go)
	case *ast.DeferStmt:
		s.addNode(_STATEMENT, 5, node.Defer)
	case *ast.FuncDecl:
		s.addNode(_FUNCTION, len(node.Name.Name), node.Name.NamePos)
		handleType(node.Type)
	case *ast.Field:
		handleType(node.Type)
	case *ast.ValueSpec:
		handleType(node.Type)
	case *ast.SwitchStmt:
		s.addNode(_CONDITIONAL, 6, node.Switch)
	case *ast.SelectStmt:
		s.addNode(_CONDITIONAL, 6, node.Select)
	case *ast.CaseClause:
		s.addNode(_LABEL, 4, node.Case)
	case *ast.RangeStmt:
		// TODO is this always safe to do?
		s.addNode(_REPEAT, 3, node.For)
		key := node.Key.(*ast.Ident)
		ass := key.Obj.Decl.(*ast.AssignStmt)
		rhs := ass.Rhs[0].(*ast.UnaryExpr)
		s.addNode(_REPEAT, 5, rhs.OpPos)
	case *ast.IfStmt:
		s.addNode(_CONDITIONAL, 2, node.If)
		// TODO need to find a way to add else highlighting
	}
	return s
}
