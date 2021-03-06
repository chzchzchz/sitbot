package msl

import (
	"fmt"
	"strings"
)

type Command interface {
	Emit() string
}

// used in stack
type commandUnion struct {
	IfCmd
	WhileCmd
	Block
	Statement

	Result Command
	Line   int
}

type IfCmd struct {
	ifCond string
	ifCmd  Command

	elseConds []string
	elseCmds  []Command
}

func (c *IfCmd) Emit() string {
	emitCondAndCmd := func(cond string, cmd Command) string {
		ret := "if runtime.EvalCond(\"" + cond + "\") {\n"
		ret += cmd.Emit()
		ret += "}"
		return ret
	}
	ret := emitCondAndCmd(c.ifCond, c.ifCmd)
	for i, ec := range c.elseConds {
		ret += " else " + emitCondAndCmd(ec, c.elseCmds[i])
	}
	if len(c.elseCmds) > len(c.elseConds) {
		ret += " else {\n"
		ret += c.elseCmds[len(c.elseCmds)-1].Emit()
		ret += "}"
	}
	return ret + "\n"
}

type WhileCmd struct {
	cond string
	cmd  Command
}

func (w *WhileCmd) Emit() string {
	return "for runtime.EvalCond(\"" + w.cond + "\") {\n" + w.cmd.Emit() + "}\n"
}

type Block struct {
	commands []Command
}

func (b *Block) Add(c Command) {
	if c == nil {
		panic("undefined command")
	}
	b.commands = append(b.commands, c)
}

func (b *Block) Emit() (ret string) {
	for _, c := range b.commands {
		if c == nil {
			ret += "???\n"
			continue
		}
		ret += c.Emit() + "\n"
	}
	return ret
}

type Statement struct {
	Values []string
}

func (m *Statement) Emit() string {
	qvals := make([]string, len(m.Values))
	for i := range m.Values {
		qvals[i] = fmt.Sprintf("%q", m.Values[i])
	}
	return "runtime.Stmt(" + strings.Join(qvals, ", ") + ")"
}
