/*
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package parser

import (
	"bytes"
	"fmt"

	"devt.de/common/stringutil"
)

// AST Nodes
// =========

/*
ASTNode models a node in the AST
*/
type ASTNode struct {
	Name     string     // Name of the node
	Token    *LexToken  // Lexer token of this ASTNode
	Children []*ASTNode // Child nodes
	Runtime  Runtime    // Runtime component for this ASTNode

	binding        int                                                             // Binding power of this node
	nullDenotation func(p *parser, self *ASTNode) (*ASTNode, error)                // Configure token as beginning node
	leftDenotation func(p *parser, self *ASTNode, left *ASTNode) (*ASTNode, error) // Configure token as left node
}

/*
ASTFromPlain creates an AST from a plain AST.
A plain AST is a nested map structure like this:

	{
		name     : <name of node>
		value    : <value of node>
		children : [ <child nodes> ]
	}
*/
func ASTFromPlain(plainAST map[string]interface{}) (*ASTNode, error) {
	var astChildren []*ASTNode

	name, ok := plainAST["name"]
	if !ok {
		return nil, fmt.Errorf("Found plain ast node without a name: %v", plainAST)
	}

	value, ok := plainAST["value"]
	if !ok {
		return nil, fmt.Errorf("Found plain ast node without a value: %v", plainAST)
	}

	// Create children

	if children, ok := plainAST["children"]; ok {

		if ic, ok := children.([]interface{}); ok {

			// Do a list conversion if necessary - this is necessary when we parse
			// JSON with map[string]interface{} this 

			childrenList := make([]map[string]interface{}, len(ic))
			for i := range ic {
				childrenList[i] = ic[i].(map[string]interface{})
			}

			children = childrenList
		}

		for _, child := range children.([]map[string]interface{}) {

			astChild, err := ASTFromPlain(child)
			if err != nil {
				return nil, err
			}

			astChildren = append(astChildren, astChild)
		}
	}

	return &ASTNode{fmt.Sprint(name), &LexToken{TokenGeneral, 0,
		fmt.Sprint(value), 0, 0}, astChildren, nil, 0, nil, nil}, nil
}

/*
Create a new instance of this ASTNode which is connected to a concrete lexer token.
*/
func (n *ASTNode) instance(p *parser, t *LexToken) *ASTNode {
	ret := &ASTNode{n.Name, t, make([]*ASTNode, 0, 2), nil, n.binding, n.nullDenotation, n.leftDenotation}
	if p.rp != nil {
		ret.Runtime = p.rp.Runtime(ret)
	}
	return ret
}

/*
Plain returns this ASTNode and all its children as plain AST. A plain AST
only contains map objects, lists and primitive types which can be serialized
with JSON.
*/
func (n *ASTNode) Plain() map[string]interface{} {
	ret := make(map[string]interface{})

	ret["name"] = n.Name

	lenChildren := len(n.Children)

	if lenChildren > 0 {
		children := make([]map[string]interface{}, lenChildren)
		for i, child := range n.Children {
			children[i] = child.Plain()
		}

		ret["children"] = children
	}

	// The value is what the lexer found in the source

	ret["value"] = n.Token.Val

	return ret
}

/*
String returns a string representation of this token.
*/
func (n *ASTNode) String() string {
	var buf bytes.Buffer
	n.levelString(0, &buf)
	return buf.String()
}

/*
levelString function to recursively print the tree.
*/
func (n *ASTNode) levelString(indent int, buf *bytes.Buffer) {

	// Print current level

	buf.WriteString(stringutil.GenerateRollingString(" ", indent*2))

	if n.Name == NodeVALUE || (n.Name == NodeSHOWTERM && n.Token.Val != "@") {
		buf.WriteString(fmt.Sprintf(n.Name+": %v", n.Token))
	} else {
		buf.WriteString(n.Name)
	}

	buf.WriteString("\n")

	// Print children

	for _, child := range n.Children {
		child.levelString(indent+1, buf)
	}
}

/*
Map of AST nodes corresponding to lexer tokens
*/
var astNodeMap map[LexTokenID]*ASTNode

/*
TokenSHOWTERM is an extra token which is generated by the parser
to group show terms
*/
const TokenSHOWTERM = LexTokenID(-1)

func init() {
	astNodeMap = map[LexTokenID]*ASTNode{
		TokenEOF:           &ASTNode{NodeEOF, nil, nil, nil, 0, ndTerm, nil},
		TokenVALUE:         &ASTNode{NodeVALUE, nil, nil, nil, 0, ndTerm, nil},
		TokenNODEKIND:      &ASTNode{NodeVALUE, nil, nil, nil, 0, ndTerm, nil},
		TokenTRUE:          &ASTNode{NodeTRUE, nil, nil, nil, 0, ndTerm, nil},
		TokenFALSE:         &ASTNode{NodeFALSE, nil, nil, nil, 0, ndTerm, nil},
		TokenNULL:          &ASTNode{NodeNULL, nil, nil, nil, 0, ndTerm, nil},
		TokenAT:            &ASTNode{NodeFUNC, nil, nil, nil, 0, ndFunc, nil},
		TokenORDERING:      &ASTNode{NodeORDERING, nil, nil, nil, 0, ndWithFunc, nil},
		TokenFILTERING:     &ASTNode{NodeFILTERING, nil, nil, nil, 0, ndWithFunc, nil},
		TokenNULLTRAVERSAL: &ASTNode{NodeNULLTRAVERSAL, nil, nil, nil, 0, ndWithFunc, nil},

		// Special tokens - always handled in a denotation function

		TokenCOMMA:  &ASTNode{NodeCOMMA, nil, nil, nil, 0, nil, nil},
		TokenGROUP:  &ASTNode{NodeGROUP, nil, nil, nil, 0, nil, nil},
		TokenEND:    &ASTNode{NodeEND, nil, nil, nil, 0, nil, nil},
		TokenAS:     &ASTNode{NodeAS, nil, nil, nil, 0, nil, nil},
		TokenFORMAT: &ASTNode{NodeFORMAT, nil, nil, nil, 0, nil, nil},

		// Keywords

		TokenGET:    &ASTNode{NodeGET, nil, nil, nil, 0, ndGet, nil},
		TokenLOOKUP: &ASTNode{NodeLOOKUP, nil, nil, nil, 0, ndLookup, nil},
		TokenFROM:   &ASTNode{NodeFROM, nil, nil, nil, 0, ndFrom, nil},
		TokenWHERE:  &ASTNode{NodeWHERE, nil, nil, nil, 0, ndPrefix, nil},

		TokenUNIQUE:      &ASTNode{NodeUNIQUE, nil, nil, nil, 0, ndPrefix, nil},
		TokenUNIQUECOUNT: &ASTNode{NodeUNIQUECOUNT, nil, nil, nil, 0, ndPrefix, nil},
		TokenISNOTNULL:   &ASTNode{NodeISNOTNULL, nil, nil, nil, 0, ndPrefix, nil},
		TokenASCENDING:   &ASTNode{NodeASCENDING, nil, nil, nil, 0, ndPrefix, nil},
		TokenDESCENDING:  &ASTNode{NodeDESCENDING, nil, nil, nil, 0, ndPrefix, nil},

		TokenTRAVERSE: &ASTNode{NodeTRAVERSE, nil, nil, nil, 0, ndTraverse, nil},
		TokenPRIMARY:  &ASTNode{NodePRIMARY, nil, nil, nil, 0, ndPrefix, nil},
		TokenSHOW:     &ASTNode{NodeSHOW, nil, nil, nil, 0, ndShow, nil},
		TokenSHOWTERM: &ASTNode{NodeSHOWTERM, nil, nil, nil, 0, ndShow, nil},
		TokenWITH:     &ASTNode{NodeWITH, nil, nil, nil, 0, ndWith, nil},
		TokenLIST:     &ASTNode{NodeLIST, nil, nil, nil, 0, nil, nil},

		// Boolean operations

		TokenNOT: &ASTNode{NodeNOT, nil, nil, nil, 20, ndPrefix, nil},
		TokenOR:  &ASTNode{NodeOR, nil, nil, nil, 30, nil, ldInfix},
		TokenAND: &ASTNode{NodeAND, nil, nil, nil, 40, nil, ldInfix},

		TokenGEQ: &ASTNode{NodeGEQ, nil, nil, nil, 60, nil, ldInfix},
		TokenLEQ: &ASTNode{NodeLEQ, nil, nil, nil, 60, nil, ldInfix},
		TokenNEQ: &ASTNode{NodeNEQ, nil, nil, nil, 60, nil, ldInfix},
		TokenEQ:  &ASTNode{NodeEQ, nil, nil, nil, 60, nil, ldInfix},
		TokenGT:  &ASTNode{NodeGT, nil, nil, nil, 60, nil, ldInfix},
		TokenLT:  &ASTNode{NodeLT, nil, nil, nil, 60, nil, ldInfix},

		TokenLIKE:        &ASTNode{NodeLIKE, nil, nil, nil, 60, nil, ldInfix},
		TokenIN:          &ASTNode{NodeIN, nil, nil, nil, 60, nil, ldInfix},
		TokenCONTAINS:    &ASTNode{NodeCONTAINS, nil, nil, nil, 60, nil, ldInfix},
		TokenBEGINSWITH:  &ASTNode{NodeBEGINSWITH, nil, nil, nil, 60, nil, ldInfix},
		TokenENDSWITH:    &ASTNode{NodeENDSWITH, nil, nil, nil, 60, nil, ldInfix},
		TokenCONTAINSNOT: &ASTNode{NodeCONTAINSNOT, nil, nil, nil, 60, nil, ldInfix},
		TokenNOTIN:       &ASTNode{NodeNOTIN, nil, nil, nil, 60, nil, ldInfix},

		// Simple arithmetic expressions

		TokenPLUS:   &ASTNode{NodePLUS, nil, nil, nil, 110, ndPrefix, ldInfix},
		TokenMINUS:  &ASTNode{NodeMINUS, nil, nil, nil, 110, ndPrefix, ldInfix},
		TokenTIMES:  &ASTNode{NodeTIMES, nil, nil, nil, 120, nil, ldInfix},
		TokenDIV:    &ASTNode{NodeDIV, nil, nil, nil, 120, nil, ldInfix},
		TokenMODINT: &ASTNode{NodeMODINT, nil, nil, nil, 120, nil, ldInfix},
		TokenDIVINT: &ASTNode{NodeDIVINT, nil, nil, nil, 120, nil, ldInfix},

		// Brackets

		TokenLPAREN: &ASTNode{NodeLPAREN, nil, nil, nil, 150, ndInner, nil},
		TokenRPAREN: &ASTNode{NodeRPAREN, nil, nil, nil, 0, nil, nil},
		TokenLBRACK: &ASTNode{NodeLBRACK, nil, nil, nil, 150, ndList, nil},
		TokenRBRACK: &ASTNode{NodeRBRACK, nil, nil, nil, 0, nil, nil},
	}
}

// Parser
// ======

/*
Parser data structure
*/
type parser struct {
	name   string          // Name to identify the input
	node   *ASTNode        // Current ast node
	tokens chan LexToken   // Channel which contains lex tokens
	rp     RuntimeProvider // Runtime provider which creates runtime components
}

/*
Parse parses a given input string and returns an AST.
*/
func Parse(name string, input string) (*ASTNode, error) {
	return ParseWithRuntime(name, input, nil)
}

/*
ParseWithRuntime parses a given input string and returns an AST decorated with
runtime components.
*/
func ParseWithRuntime(name string, input string, rp RuntimeProvider) (*ASTNode, error) {
	p := &parser{name, nil, Lex(name, input), rp}

	node, err := p.next()

	if err != nil {
		return nil, err
	}

	p.node = node

	return p.run(0)
}

/*
run models the main parser function.
*/
func (p *parser) run(rightBinding int) (*ASTNode, error) {
	var err error

	n := p.node

	p.node, err = p.next()
	if err != nil {
		return nil, err
	}

	// Start with the null denotation of this statement / expression

	if n.nullDenotation == nil {
		return nil, p.newParserError(ErrImpossibleNullDenotation,
			n.Token.String(), *n.Token)
	}

	left, err := n.nullDenotation(p, n)
	if err != nil {
		return nil, err
	}

	// Collect left denotations as long as the left binding power is greater
	// than the initial right one

	for rightBinding < p.node.binding {
		var nleft *ASTNode

		n = p.node

		p.node, err = p.next()

		if err != nil {
			return nil, err
		}

		if n.leftDenotation == nil {
			return nil, p.newParserError(ErrImpossibleLeftDenotation,
				n.Token.String(), *n.Token)
		}

		// Get the next left denotation

		nleft, err = n.leftDenotation(p, n, left)

		left = nleft

		if err != nil {
			return nil, err
		}
	}

	return left, nil
}

/*
next retrieves the next lexer token.
*/
func (p *parser) next() (*ASTNode, error) {

	token, more := <-p.tokens

	if !more {

		// Unexpected end of input - the associated token is an empty error token

		return nil, p.newParserError(ErrUnexpectedEnd, "", token)

	} else if token.ID == TokenError {

		// There was a lexer error wrap it in a parser error

		return nil, p.newParserError(ErrLexicalError, token.Val, token)

	} else if node, ok := astNodeMap[token.ID]; ok {

		return node.instance(p, &token), nil
	}

	return nil, p.newParserError(ErrUnknownToken, fmt.Sprintf("id:%v (%v)", token.ID, token), token)
}

// Standard null denotation functions
// ==================================

/*
ndTerm is used for terminals.
*/
func ndTerm(p *parser, self *ASTNode) (*ASTNode, error) {
	return self, nil
}

/*
ndInner returns the inner expression of an enclosed block and discard the
block token. This method is used for brackets.
*/
func ndInner(p *parser, self *ASTNode) (*ASTNode, error) {

	// Get the inner expression

	exp, err := p.run(0)
	if err != nil {
		return nil, err
	}

	// We return here the inner expression - discarding the bracket tokens

	return exp, skipToken(p, TokenRPAREN)
}

/*
ndPrefix is used for prefix operators.
*/
func ndPrefix(p *parser, self *ASTNode) (*ASTNode, error) {

	// Make sure a prefix will only prefix the next item

	val, err := p.run(self.binding + 20)
	if err != nil {
		return nil, err
	}

	self.Children = append(self.Children, val)

	return self, nil
}

// Null denotation functions for specific expressions
// ==================================================

/*
ndGet is used to parse lookup expressions.
*/
func ndGet(p *parser, self *ASTNode) (*ASTNode, error) {

	// Must specify a node kind

	if err := acceptChild(p, self, TokenNODEKIND); err != nil {
		return nil, err
	}

	// Parse the rest and add it as children

	for p.node.Token.ID != TokenEOF {
		exp, err := p.run(0)
		if err != nil {
			return nil, err
		}

		self.Children = append(self.Children, exp)
	}

	return self, nil
}

/*
ndLookup is used to parse lookup expressions.
*/
func ndLookup(p *parser, self *ASTNode) (*ASTNode, error) {

	// Must specify a node kind

	if err := acceptChild(p, self, TokenNODEKIND); err != nil {
		return nil, err
	}

	// Must have at least on node key

	if err := acceptChild(p, self, TokenVALUE); err != nil {
		return nil, err
	}

	// Read all commas and accept further values as additional node keys

	for skipToken(p, TokenCOMMA) == nil {
		if err := acceptChild(p, self, TokenVALUE); err != nil {
			return nil, err
		}
	}

	// Parse the rest and add it as children

	for p.node.Token.ID != TokenEOF {
		exp, err := p.run(0)
		if err != nil {
			return nil, err
		}

		self.Children = append(self.Children, exp)
	}

	return self, nil
}

/*
ndFrom is used to parse from group ... expressions.
*/
func ndFrom(p *parser, self *ASTNode) (*ASTNode, error) {

	// Must be followed by a group keyword

	if err := acceptChild(p, self, TokenGROUP); err != nil {
		return nil, err
	}

	// Must have a group name

	return self, acceptChild(p, self.Children[0], TokenVALUE)
}

/*
ndTraverse is used to parse traverse expressions.
*/
func ndTraverse(p *parser, self *ASTNode) (*ASTNode, error) {

	// Must be followed by traversal spec

	if err := acceptChild(p, self, TokenVALUE); err != nil {
		return nil, err
	}

	// Parse the rest and add it as children - must end with "end" if
	// further clauses are given

	for p.node.Token.ID != TokenEOF && p.node.Token.ID != TokenEND {
		exp, err := p.run(0)
		if err != nil {
			return nil, err
		}

		self.Children = append(self.Children, exp)
	}

	if p.node.Token.ID == TokenEND {
		skipToken(p, TokenEND)
	}

	return self, nil
}

/*
ndFunc is used to parse functions.
*/
func ndFunc(p *parser, self *ASTNode) (*ASTNode, error) {

	// Must specify a name

	if err := acceptChild(p, self, TokenVALUE); err != nil {
		return nil, err
	}

	// Must have an opening bracket

	if err := skipToken(p, TokenLPAREN); err != nil {
		return nil, err
	}

	// Read in the first attribute

	if p.node.Token.ID == TokenVALUE {

		// Next call cannot fail since we just checked for it. Value is optional.

		acceptChild(p, self, TokenVALUE)

		// Read all commas and accept further values as parameters until the end

		for skipToken(p, TokenCOMMA) == nil {
			if err := acceptChild(p, self, TokenVALUE); err != nil {
				return nil, err
			}
		}
	}

	// Must have a closing bracket

	return self, skipToken(p, TokenRPAREN)
}

/*
ndShow is used to parse a show clauses.
*/
func ndShow(p *parser, self *ASTNode) (*ASTNode, error) {

	acceptShowTerm := func() error {
		st := astNodeMap[TokenSHOWTERM].instance(p, p.node.Token)

		if p.node.Token.ID == TokenAT {

			// Parse a function

			exp, err := p.run(0)
			if err != nil {
				return err
			}

			st.Children = append(st.Children, exp)

		} else {

			// Skip the value token from which we just created an AST node

			skipToken(p, TokenVALUE)
		}

		// Parse an "as" definition if given

		if p.node.Token.ID == TokenAS {

			current := p.node
			acceptChild(p, st, TokenAS)

			if err := acceptChild(p, current, TokenVALUE); err != nil {
				return err
			}
		}

		// Parse a "format" definition if given

		if p.node.Token.ID == TokenFORMAT {

			current := p.node
			acceptChild(p, st, TokenFORMAT)

			if err := acceptChild(p, current, TokenVALUE); err != nil {
				return err
			}
		}

		self.Children = append(self.Children, st)

		return nil
	}

	// Read in the first node attribute

	if p.node.Token.ID == TokenVALUE || p.node.Token.ID == TokenAT {
		if err := acceptShowTerm(); err != nil {
			return nil, err
		}

		// Read further show entries

		for skipToken(p, TokenCOMMA) == nil {
			if err := acceptShowTerm(); err != nil {
				return nil, err
			}
		}
	}

	return self, nil
}

/*
ndWith is used to parse a with clauses.
*/
func ndWith(p *parser, self *ASTNode) (*ASTNode, error) {

	// Parse the rest and add it as children

	for p.node.Token.ID != TokenEOF {
		exp, err := p.run(0)
		if err != nil {
			return nil, err
		}

		self.Children = append(self.Children, exp)

		if p.node.Token.ID == TokenCOMMA {
			skipToken(p, TokenCOMMA)
		}
	}

	return self, nil
}

/*
ndWithFunc is used to parse directives in with clauses.
*/
func ndWithFunc(p *parser, self *ASTNode) (*ASTNode, error) {

	// Must have an opening bracket

	if err := skipToken(p, TokenLPAREN); err != nil {
		return nil, err
	}

	for p.node.Token.ID != TokenRPAREN {

		// Parse all the expressions inside the directives

		exp, err := p.run(0)
		if err != nil {
			return nil, err
		}

		self.Children = append(self.Children, exp)

		if p.node.Token.ID == TokenCOMMA {
			skipToken(p, TokenCOMMA)
		}
	}

	// Must have a closing bracket

	return self, skipToken(p, TokenRPAREN)
}

/*
ndList is used to collect elements of a list.
*/
func ndList(p *parser, self *ASTNode) (*ASTNode, error) {

	// Create a list token

	st := astNodeMap[TokenLIST].instance(p, self.Token)

	// Get the inner expression

	for p.node.Token.ID != TokenRBRACK {

		// Parse all the expressions inside the directives

		exp, err := p.run(0)
		if err != nil {
			return nil, err
		}

		st.Children = append(st.Children, exp)

		if p.node.Token.ID == TokenCOMMA {
			skipToken(p, TokenCOMMA)
		}
	}

	// Must have a closing bracket

	return st, skipToken(p, TokenRBRACK)
}

// Standard left denotation functions
// ==================================

/*
ldInfix is used for infix operators.
*/
func ldInfix(p *parser, self *ASTNode, left *ASTNode) (*ASTNode, error) {

	right, err := p.run(self.binding)
	if err != nil {
		return nil, err
	}

	self.Children = append(self.Children, left)
	self.Children = append(self.Children, right)

	return self, nil
}

// Helper functions
// ================

/*
skipToken skips over a given token.
*/
func skipToken(p *parser, ids ...LexTokenID) error {
	var err error

	canSkip := func(id LexTokenID) bool {
		for _, i := range ids {
			if i == id {
				return true
			}
		}
		return false
	}

	if !canSkip(p.node.Token.ID) {
		if p.node.Token.ID == TokenEOF {
			return p.newParserError(ErrUnexpectedEnd, "", *p.node.Token)
		}
		return p.newParserError(ErrUnexpectedToken, p.node.Token.Val, *p.node.Token)
	}

	// This should never return an error unless we skip over EOF or complex tokens
	// like values

	p.node, err = p.next()

	return err
}

/*
acceptChild accepts the current token as a child.
*/
func acceptChild(p *parser, self *ASTNode, id LexTokenID) error {
	var err error

	current := p.node

	p.node, err = p.next()
	if err != nil {
		return err
	}

	if current.Token.ID == id {
		self.Children = append(self.Children, current)
		return nil
	}

	return p.newParserError(ErrUnexpectedToken, current.Token.Val, *current.Token)
}
