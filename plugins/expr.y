// go tool yacc expr.y

%{
package plugins

import (
	"bytes"
	"errors"
	"log"
	"math"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Math functions we want to be able to use
// I hate having to use interface, but that's the
// price we pay for making this interpreted
//
// Note that we only really accept taking multiple
// float64 values and returning a single float64
var funcMap = map[string]interface{} {
	"abs":   math.Abs,
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"sqrt":  math.Sqrt,
	"floor": math.Floor,
	"ceil":  math.Ceil,
}

%}

%union {
	num float64
	str string
	vals []float64
}

// Set what expr returns
%type <num> expr
%type <vals> exprlist

// Set the types of possible lexed values
%token <num> NUM
%token <str> FUNC

%left '+' '-'
%left '*' '/' '%'
%left '^'

%%

prog:
	stmt
	|
	;
stmt:
	expr
	{
		lex := yylex.(*yyLex)
		lex.val = $1
	}
	;
expr:
	NUM
	| '-' expr
	{
		$$ = -$2
	}
	| expr '+' expr
	{
		$$ = $1 + $3
	}
	| expr '-' expr
	{
		$$ = $1 - $3
	}
	| expr '*' expr
	{
		$$ = $1 * $3
	}
	| expr '/' expr
	{
		$$ = $1 / $3
	}
	| expr '%' expr
	{
		$$ = math.Mod($1, $3)
	}
	| expr '^' expr
	{
		$$ = math.Pow($1, $3)
	}
	| FUNC '(' exprlist ')'
	{
		$$ = callFunc($1, $3)
	}
	| '(' expr ')'
	{
		$$ = $2
	}
	;
exprlist:
	expr
	{
		$$ = append($$, $1)
	}
	| exprlist ',' expr
	{
		$$ = append($1, $3)
	}
	;
%%

// The parser expects the lexer to return 0 on EOF.  Give it a name
// for clarity.
const EOF = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type yyLex struct {
	line string
	peek rune
	val float64
	err error
}

func callFunc(name string, args []float64) float64 {
	f, ok := funcMap[name]
	if !ok {
		// TODO: Set error
		return 0
	}

	v := reflect.ValueOf(f)
	t := v.Type()
	if t.Kind() != reflect.Func || t.NumOut() != 1 || t.Out(0).Kind() != reflect.Float64 || t.NumIn() != len(args) {
		// TODO: Set error
		return 0
	}

	for i := 0; i < t.NumIn(); i++ {
		if t.In(i).Kind() != reflect.Float64 {
			// TODO: Set error
			return 0
		}
	}

	var vals []reflect.Value
	for _, arg := range args {
		vals = append(vals, reflect.ValueOf(arg))
	}

	ret := v.Call(vals)
	return ret[0].Float()
}

// The parser calls this method to get each new token.  This
// implementation returns operators and NUM.
func (x *yyLex) Lex(yylval *yySymType) int {
	for {
		c := x.next()
		switch c {
		// If we're at the end
		case EOF:
			return EOF

		// Any punctuation needed for expressions
		case '+', '-', '*', '/', '%', '^', '(', ')':
			return int(c)

		// Recognize Unicode multiplication and division
		// symbols, returning what the parser expects.
		case '×':
			return '*'
		case '÷':
			return '/'

		default:
			if unicode.IsSpace(c) {
				// Clear out whitespace
				continue
			} else if unicode.IsLetter(c) {
				return x.str(c, yylval)
			} else if unicode.IsNumber(c) {
				return x.num(c, yylval)
			}

			log.Printf("unrecognized character %q", c)
		}
	}
}

// Lex a number.
func (x *yyLex) num(c rune, yylval *yySymType) int {
	add := func(b *bytes.Buffer, c rune) {
		if _, err := b.WriteRune(c); err != nil {
			log.Fatalf("WriteRune: %s", err)
		}
	}

	var b bytes.Buffer
	add(&b, c)
	for {
		c = x.next()
		if !unicode.IsNumber(c) && c != '.' && c != 'e' && c != 'E' {
			break
		}
		add(&b, c)
	}

	if c != EOF {
		x.peek = c
	}

	tmp, err := strconv.ParseFloat(b.String(), 64)
	if err != nil {
		log.Printf("bad number %q", b.String())
		return EOF
	}

	yylval.num = tmp
	return NUM
}

func (x *yyLex) str(c rune, yylval *yySymType) int {
	add := func(b *bytes.Buffer, c rune) {
		if _, err := b.WriteRune(c); err != nil {
			log.Fatalf("WriteRune: %s", err)
		}
	}

	var b bytes.Buffer
	add(&b, c)
	for {
		c = x.next()
		if !unicode.IsLetter(c) {
			break
		}
		add(&b, c)
	}

	if c != EOF {
		x.peek = c
	}

	yylval.str = b.String()
	return FUNC
}

// Return the next rune for the lexer.
func (x *yyLex) next() rune {
	if x.peek != EOF {
		r := x.peek
		x.peek = EOF
		return r
	}

	if len(x.line) == 0 {
		return EOF
	}

	c, size := utf8.DecodeRuneInString(x.line)
	x.line = x.line[size:]
	if c == utf8.RuneError && size == 1 {
		log.Print("invalid utf8")
		return x.next()
	}
	return c
}

// The parser calls this method on a parse error.
func (x *yyLex) Error(s string) {
	x.err = errors.New(s)
}

func parseExpr(line string) (float64, error) {
	lex := &yyLex{line: line}
	yyParse(lex)
	return lex.val, lex.err
}
