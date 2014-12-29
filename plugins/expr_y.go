//line expr.y:4
package plugins

import __yyfmt__ "fmt"

//line expr.y:4
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
var funcMap = map[string]func(float64) float64{
	"abs": math.Abs,
	"sin": math.Sin,
	"cos": math.Cos,
	"tan": math.Tan,
}

//line expr.y:27
type yySymType struct {
	yys int
	num float64
	str string
}

const NUM = 57346
const FUNC = 57347

var yyToknames = []string{
	"NUM",
	"FUNC",
	"'+'",
	"'-'",
	"'*'",
	"'/'",
	"'%'",
	"'^'",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line expr.y:103

// The parser expects the lexer to return 0 on EOF.  Give it a name
// for clarity.
const eof = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type yyLex struct {
	line string
	peek rune
	val  float64
	err  error
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

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 14
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 48

var yyAct = []int{

	3, 10, 11, 12, 13, 13, 14, 15, 16, 17,
	18, 19, 20, 21, 22, 2, 23, 8, 9, 10,
	11, 12, 13, 1, 25, 8, 9, 10, 11, 12,
	13, 0, 24, 8, 9, 10, 11, 12, 13, 4,
	6, 0, 5, 0, 0, 0, 0, 7,
}
var yyPact = []int{

	35, -1000, -1000, 27, -1000, 35, -5, 35, 35, 35,
	35, 35, 35, 35, -7, 35, 19, -7, -7, -6,
	-6, -6, -1000, 11, -1000, -1000,
}
var yyPgo = []int{

	0, 0, 23, 15,
}
var yyR1 = []int{

	0, 2, 2, 3, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1,
}
var yyR2 = []int{

	0, 1, 0, 1, 1, 2, 3, 3, 3, 3,
	3, 3, 4, 3,
}
var yyChk = []int{

	-1000, -2, -3, -1, 4, 7, 5, 12, 6, 7,
	8, 9, 10, 11, -1, 12, -1, -1, -1, -1,
	-1, -1, -1, -1, 13, 13,
}
var yyDef = []int{

	2, -2, 1, 3, 4, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 5, 0, 0, 6, 7, 8,
	9, 10, 11, 0, 13, 12,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 10, 3, 3,
	12, 13, 8, 6, 3, 7, 3, 9, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 11,
}
var yyTok2 = []int{

	2, 3, 4, 5,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(yyToknames) {
		if yyToknames[c-4] != "" {
			return yyToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(c), uint(char))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 3:
		//line expr.y:51
		{
			lex, ok := yylex.(*yyLex)
			if !ok {
				fmt.Println("Wrong lexer format")
			} else {
				lex.val = yyS[yypt-0].num
			}
		}
	case 4:
		yyVAL.num = yyS[yypt-0].num
	case 5:
		//line expr.y:63
		{
			yyVAL.num = -yyS[yypt-0].num
		}
	case 6:
		//line expr.y:67
		{
			yyVAL.num = yyS[yypt-2].num + yyS[yypt-0].num
		}
	case 7:
		//line expr.y:71
		{
			yyVAL.num = yyS[yypt-2].num - yyS[yypt-0].num
		}
	case 8:
		//line expr.y:75
		{
			yyVAL.num = yyS[yypt-2].num * yyS[yypt-0].num
		}
	case 9:
		//line expr.y:79
		{
			yyVAL.num = yyS[yypt-2].num / yyS[yypt-0].num
		}
	case 10:
		//line expr.y:83
		{
			yyVAL.num = math.Mod(yyS[yypt-2].num, yyS[yypt-0].num)
		}
	case 11:
		//line expr.y:87
		{
			yyVAL.num = math.Pow(yyS[yypt-2].num, yyS[yypt-0].num)
		}
	case 12:
		//line expr.y:91
		{
			f, ok := funcMap[yyS[yypt-3].str]
			if !ok {

			}
			yyVAL.num = f(yyS[yypt-1].num)
		}
	case 13:
		//line expr.y:99
		{
			yyVAL.num = yyS[yypt-1].num
		}
	}
	goto yystack /* stack new state and value */
}
