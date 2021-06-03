//go:generate peg -inline eval.peg
package runtime

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/sorcix/irc.v2"
)

type callFrame struct {
	cmd  string
	args []string
}

func EvalCond(s string) bool {
	log.Println("start eval condition", s)
	v := eval(s)
	log.Println("eval condition", s, "->", v)
	return s2b(v)
}

func mustVar(s string) {
	if s[0] != '%' {
		panic("expected % on var " + s)
	}
}

func joinNoEmptySpaces(s []string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == "" {
			s = append(s[:i], s[i+1:]...)
			i--
		}
	}
	return strings.Join(s, " ")
}

func Stmt(values ...string) {
	// Evaluate every term to get concrete statement.
	terms := make([]string, len(values))
	for i, s := range values {
		terms[i] = eval(s)
	}
	// Run commmand.
	log.Printf("stmt terms: %+v => %+v\n", values, terms)
	switch strings.ToLower(terms[0]) {
	case "msg":
		msg := &irc.Message{
			Command: irc.PRIVMSG,
			Params:  []string{terms[1], joinNoEmptySpaces(terms[2:])},
		}
		mustPostCommand(msg)
	case "mode":
		msg := &irc.Message{Command: irc.MODE, Params: terms[1:]}
		mustPostCommand(msg)
	case "var":
		terms[1] = evalReference(values[1])
		mustVar(terms[1])
		mslVar.SetLocal(terms[1][1:], terms[2])
		log.Println("var ", terms[1], "=", terms[2])
	case "inc":
		terms[1] = evalReference(values[1])
		mustVar(terms[1])
		v := 1.0
		if len(terms) > 2 {
			v = s2f(terms[2])
		}
		if oldv, ok := mslVar.Lookup(terms[1][1:]); ok {
			v += s2f(oldv)
		}
		mslVar.SetOverride(terms[1][1:], f2s(v))
	case "dec":
		terms[1] = evalReference(values[1])
		mustVar(terms[1])
		v := 1.0
		if len(terms) > 2 {
			v = s2f(terms[2])
		}
		if oldv, ok := mslVar.Lookup(terms[1][1:]); ok {
			v = s2f(oldv) - v
		}
		mslVar.SetOverride(terms[1][1:], f2s(v))
	case "set":
		terms[1] = evalReference(values[1])
		mustVar(terms[1])
		mslVar.SetGlobal(terms[1][1:], terms[2])
	case "unset":
		terms[1] = evalReference(values[1])
		mustVar(terms[1])
		if strings.Contains(terms[1], "*") {
			restr := "^" + strings.Replace(terms[1][1:], "*", ".*", -1) + "$"
			re, err := regexp.Compile(restr)
			if err != nil {
				panic(err)
			}
			for k := range mslVar.Globals {
				if re.MatchString(k) {
					log.Println("unsetting variable", k)
					delete(mslVar.Globals, k)
				}
			}
		} else {
			delete(mslVar.Globals, terms[1][1:])
		}
	default:
		panic(fmt.Sprintf("evaled: %+v -> %+v", values, terms))
	}
}

func f2s(f float64) string {
	s := fmt.Sprintf("%f", f)
	ss := strings.Split(s, ".")
	if len(ss) == 1 {
		return s
	}
	for len(ss[1]) > 0 {
		if ss[1][len(ss[1])-1] != '0' {
			break
		}
		ss[1] = ss[1][:len(ss[1])-1]
	}
	if len(ss[1]) == 0 {
		return ss[0]
	}
	return strings.Join(ss, ".")
}

func evalReference(s string) string {
	return evalAny(s, true)
}

func eval(s string) string {
	log.Printf("start eval %q", s)
	return evalAny(s, false)
}

func evalAny(s string, keepReferences bool) string {
	g := EvalGrammar{Buffer: s, keepReferences: keepReferences}
	g.Init()
	if err := g.Parse(); err != nil {
		panic(err)
	}
	// g.PrintSyntaxTree()
	g.Execute()
	return g.v
}

func (g *EvalGrammar) startCall(s string) {
	g.frames = append(g.frames, callFrame{cmd: s})
}

func (g *EvalGrammar) addCallArg(s string) {
	i := len(g.frames) - 1
	g.frames[i].args = append(g.frames[i].args, s)
}

func (g *EvalGrammar) endCall() {
	i := len(g.frames) - 1
	top := g.frames[i]
	v := ""
	log.Println("issuing call", top.cmd, top.args)
	switch strings.ToLower(top.cmd) {
	case "chan":
		if v = mslEv.Chan; v == "" {
			panic("no channel")
		}
	case "nick":
		if v = mslEv.Nick; v == "" {
			panic("no nick")
		}
	case "upper":
		v = strings.ToUpper(top.args[0])
	case "rand":
		lo, err := strconv.Atoi(top.args[0])
		if err != nil {
			panic(err)
		}
		hi, err := strconv.Atoi(top.args[1])
		if err != nil {
			panic(err)
		}
		v = fmt.Sprintf("%d", rand.Int63n((int64(hi-lo))+1)+int64(lo))
	case "calc":
		v = top.args[0]
		vv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			// TODO: handle expressions
			panic(err)
		}
		v = f2s(vv)
	case "replace":
		v = top.args[0]
		log.Println("args", top.args)
		for i := 1; i < len(top.args); i += 2 {
			v = strings.Replace(v, top.args[i], top.args[i+1], -1)
		}
		log.Println("replaced", top.args[0], "to", v)
	case "bytes":
		vv := s2f(top.args[0])
		if top.args[1] != "b" {
			panic("unexpected byte format " + top.args[1])
		}
		v = message.NewPrinter(language.English).Sprintf("%d", uint(vv))
	case "len":
		v = fmt.Sprintf("%d", len(top.args[0]))
	case "right":
		vv := int(s2f(top.args[1]))
		if vv <= 0 {
			panic("negative right")
		}
		v, l := top.args[0], len(top.args[0])
		if vv < l {
			v = v[l-vv:]
		}
	case "int":
		v = fmt.Sprintf("%d", int(s2f(top.args[0])))
	case "nopnick":
		nn := NopNicks(top.args[0])
		n, err := strconv.Atoi(top.args[1])
		if err != nil {
			// nick mode
			v = "$null"
			for i, nick := range nn {
				if nick == top.args[1] {
					v = fmt.Sprintf("%d", i)
					break
				}
			}
		} else if n == 0 {
			// count mode
			v = fmt.Sprintf("%d", len(nn))
		} else if n > 0 && n <= len(nn) {
			v = nn[n-1]
		} else {
			v = ""
		}
	default:
		panic(top.cmd + fmt.Sprintf("(%+v)", g.frames[i].args))
	}
	g.v, g.frames = v, g.frames[:i]
}

func (g *EvalGrammar) startValueAppend() {
	g.pastes, g.v = append(g.pastes, g.v), ""
}

func (g *EvalGrammar) endValueAppend() {
	g.pastes = append(g.pastes, g.v)
	l := len(g.pastes)
	g.v = strings.Join(g.pastes[l-2:l], "")
	g.pastes = g.pastes[:l-2]
}

func (g *EvalGrammar) evalVar() (ret string) {
	name, suffix := g.v, ""
	ret = "$null"
	for len(name) > 0 {
		if g.keepReferences && g.varDepth < 2 {
			ret = "%" + name
		} else if g.exprDepth == 0 {
			ret = g.v
		} else if v, ok := mslVar.Lookup(name); ok {
			ret = v
		} else {
			// Hack for $+; search for match then append suffix.
			suffix = name[len(name)-1:] + suffix
			name = name[:len(name)-1]
			continue
		}
		break
	}
	if len(name) != 0 {
		ret += suffix
	}
	log.Printf("evaluated %q -> %q => %q", name, g.v, ret)
	return ret
}

type binOp func(a, b string) string

func (g *EvalGrammar) pushBinOp(f binOp) {
	g.binOps = append(g.binOps, f)
	g.exprStack = append(g.exprStack, g.v)
}

func s2f(s string) float64 {
	if s == "$null" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func s2b(s string) bool {
	switch s {
	case "true":
		return true
	case "false":
		return false
	default:
		panic("not bool: " + s)
	}
}

func b2s(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Case insensitive.
func eqOp(a, b string) string { return b2s(strings.ToUpper(a) == strings.ToUpper(b)) }
func neOp(a, b string) string { return b2s(strings.ToUpper(a) != strings.ToUpper(b)) }

func geOp(a, b string) string { return b2s(s2f(a) >= s2f(b)) }
func gtOp(a, b string) string { return b2s(s2f(a) > s2f(b)) }
func leOp(a, b string) string { return b2s(s2f(a) <= s2f(b)) }
func ltOp(a, b string) string { return b2s(s2f(a) < s2f(b)) }

func orOp(a, b string) string  { return b2s(s2b(a) || s2b(b)) }
func andOp(a, b string) string { return b2s(s2b(a) && s2b(b)) }

func isIn(a, b string) string    { return b2s(strings.Contains(b, a)) }
func isNotIn(a, b string) string { return b2s(!strings.Contains(b, a)) }

func (g *EvalGrammar) applyBinOp() {
	i, j := len(g.binOps)-1, len(g.exprStack)
	lhs, rhs := g.exprStack[j-2], g.exprStack[j-1]
	// log.Printf("before bin op (%+v, %+v); g.v=%+v exprs=%+v", lhs, rhs, g.v, g.exprStack)
	ret := g.binOps[i](lhs, rhs)
	g.binOps, g.exprStack = g.binOps[:i], g.exprStack[:j-2]
	// log.Printf("after bin op exprs=%+v ret=%+v", g.exprStack, ret)
	g.v = ret
}

func (g *EvalGrammar) msgTokens(s string) string {
	isList := s[len(s)-1] == '-'
	if isList {
		s = s[:len(s)-1]
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return "$" + s
	}
	v--
	fs := strings.Fields(mslEv.Msg)
	if (v >= len(fs) || v < 0) && !isList {
		return "$" + s
	}
	if isList {
		return strings.Join(fs[v:], " ")
	}
	return fs[v]
}
