// go tool yacc math.y

%{
package plugins

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Math functions we want to be able to use
var funcMap = map[string]func(float64) float64 {
	"abs": math.Abs,
	"sin": math.Sin,
	"cos": math.Cos,
	"tan": math.Tan,
}

%}

%union {
	num float64
	str string
}

// Set what expr returns
%type  <num> expr

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
		lex, ok := yylex.(*yyLex)
		if !ok {
			fmt.Println("Wrong lexer format")
		} else {
			lex.val = $1
		}
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
	| FUNC '(' expr ')'
	{
		f, ok := funcMap[$1]
		if !ok {

		}
		$$ = f($3)
	}
	| '(' expr ')'
	{
		$$ = $2
	}
	;
%%

// The parser expects the lexer to return 0 on EOF.  Give it a name
// for clarity.
const eof = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type yyLex struct {
	line string
	peek rune
	val float64
	err error
}

// The parser calls this method to get each new token.  This
// implementation returns operators and NUM.
func (x *yyLex) Lex(yylval *yySymType) int {
	for {
		c := x.next()
		switch c {
		// If we're at the end
		case eof:
			return eof

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
			} else if unicode.IsLetter(c) {
				return x.str(c, yylval)
			} else if unicode.IsNumber(c) {
				return x.num(c, yylval)
			} else {
				log.Printf("unrecognized character %q", c)
			}
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

	if c != eof {
		x.peek = c
	}

	tmp, err := strconv.ParseFloat(b.String(), 64)
	if err != nil {
		log.Printf("bad number %q", b.String())
		return eof
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

	if c != eof {
		x.peek = c
	}

	yylval.str = b.String()
	return FUNC
}

// Return the next rune for the lexer.
func (x *yyLex) next() rune {
	if x.peek != eof {
		r := x.peek
		x.peek = eof
		return r
	}

	if len(x.line) == 0 {
		return eof
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
