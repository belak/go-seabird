//line expr.y:4
package plugins

import __yyfmt__ "fmt"

//line expr.y:4
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
var funcMap = map[string]interface{}{
	"abs":   math.Abs,
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"sqrt":  math.Sqrt,
	"floor": math.Floor,
	"ceil":  math.Ceil,
}

//line expr.y:35
type yySymType struct {
	yys  int
	num  float64
	str  string
	vals []float64
}

const NUM = 57346
const FUNC = 57347

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"NUM",
	"FUNC",
	"'+'",
	"'-'",
	"'*'",
	"'/'",
	"'%'",
	"'^'",
	"'('",
	"')'",
	"','",
}
var yyStatenames = [...]string{}

const yyEOFCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line expr.y:115

// The parser expects the lexer to return 0 on EOF.  Give it a name
// for clarity.
const EOF = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type yyLex struct {
	line string
	peek rune
	val  float64
	err  error
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

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 16
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 48

var yyAct = [...]int{

	3, 26, 27, 15, 13, 2, 14, 1, 16, 17,
	18, 19, 20, 21, 22, 23, 24, 8, 9, 10,
	11, 12, 13, 0, 25, 0, 0, 0, 28, 8,
	9, 10, 11, 12, 13, 4, 6, 0, 5, 0,
	0, 0, 0, 7, 10, 11, 12, 13,
}
var yyPact = [...]int{

	31, -1000, -1000, 23, -1000, 31, -9, 31, 31, 31,
	31, 31, 31, 31, 36, 31, 11, 36, 36, -7,
	-7, -7, -1000, -12, 23, -1000, -1000, 31, 23,
}
var yyPgo = [...]int{

	0, 0, 15, 7, 5,
}
var yyR1 = [...]int{

	0, 3, 3, 4, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 2, 2,
}
var yyR2 = [...]int{

	0, 1, 0, 1, 1, 2, 3, 3, 3, 3,
	3, 3, 4, 3, 1, 3,
}
var yyChk = [...]int{

	-1000, -3, -4, -1, 4, 7, 5, 12, 6, 7,
	8, 9, 10, 11, -1, 12, -1, -1, -1, -1,
	-1, -1, -1, -2, -1, 13, 13, 14, -1,
}
var yyDef = [...]int{

	2, -2, 1, 3, 4, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 5, 0, 0, 6, 7, 8,
	9, 10, 11, 0, 14, 13, 12, 0, 15,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 10, 3, 3,
	12, 13, 8, 6, 14, 7, 3, 9, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 11,
}
var yyTok2 = [...]int{

	2, 3, 4, 5,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lookahead func() int
}

func (p *yyParserImpl) Lookahead() int {
	return p.lookahead()
}

func yyNewParser() yyParser {
	p := &yyParserImpl{
		lookahead: func() int { return -1 },
	}
	return p
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
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

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yytoken := -1 // yychar translated into internal numbering
	yyrcvr.lookahead = func() int { return yychar }
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yychar = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
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
		yychar, yytoken = yylex1(yylex, &yylval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yychar = -1
		yytoken = -1
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
			yychar, yytoken = yylex1(yylex, &yylval)
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
			if yyn < 0 || yyn == yytoken {
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
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
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
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEOFCode {
				goto ret1
			}
			yychar = -1
			yytoken = -1
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
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
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
		yyDollar = yyS[yypt-1 : yypt+1]
		//line expr.y:61
		{
			lex := yylex.(*yyLex)
			lex.val = yyDollar[1].num
		}
	case 5:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line expr.y:69
		{
			yyVAL.num = -yyDollar[2].num
		}
	case 6:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:73
		{
			yyVAL.num = yyDollar[1].num + yyDollar[3].num
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:77
		{
			yyVAL.num = yyDollar[1].num - yyDollar[3].num
		}
	case 8:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:81
		{
			yyVAL.num = yyDollar[1].num * yyDollar[3].num
		}
	case 9:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:85
		{
			yyVAL.num = yyDollar[1].num / yyDollar[3].num
		}
	case 10:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:89
		{
			yyVAL.num = math.Mod(yyDollar[1].num, yyDollar[3].num)
		}
	case 11:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:93
		{
			yyVAL.num = math.Pow(yyDollar[1].num, yyDollar[3].num)
		}
	case 12:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line expr.y:97
		{
			yyVAL.num = callFunc(yyDollar[1].str, yyDollar[3].vals)
		}
	case 13:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:101
		{
			yyVAL.num = yyDollar[2].num
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line expr.y:107
		{
			yyVAL.vals = append(yyVAL.vals, yyDollar[1].num)
		}
	case 15:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line expr.y:111
		{
			yyVAL.vals = append(yyDollar[1].vals, yyDollar[3].num)
		}
	}
	goto yystack /* stack new state and value */
}
