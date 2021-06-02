//go:generate peg -inline msl.peg
package msl

func (g *Grammar) Line() int {
	return g.line + 1
}

func (g *Grammar) closeEvent() {
	e := &g.Events[len(g.Events)-1]
	e.Type, e.Location, e.Level, e.Pattern = g.evType, g.location, g.level, g.pattern
	g.evType, g.location, g.level, g.pattern = "", "", "", ""

	e.Command = g.topCmd().Result
	g.cmdStack, g.values = nil, nil
}

func (g *Grammar) incLine(s string) {
	for _, v := range s {
		if v == '\n' {
			g.line++
		}
	}
}

func (g *Grammar) pushCommand() {
	g.cmdStack = append(g.cmdStack, &commandUnion{Line: g.Line()})
}

func (g *Grammar) popCommand() {
	g.cmdStack[len(g.cmdStack)-1].Result = nil
	g.cmdStack, g.values = g.cmdStack[:len(g.cmdStack)-1], nil
}

func (g *Grammar) topCmd() *commandUnion {
	return g.nCmd(1)
}

func (g *Grammar) nCmd(n int) *commandUnion {
	return g.cmdStack[len(g.cmdStack)-n]
}

func (g *Grammar) addValue(s string) {
	if g.inId == 0 && len(g.values) > 0 {
		g.values[len(g.values)-1] += s
	}
}

func (g *Grammar) pushValue(s string) {
	g.values = append(g.values, s)
}
