package antiflood

import "github.com/astralmaster/elrond-go/process"

func (af *p2pAntiflood) Debugger() process.AntifloodDebugger {
	return af.debugger
}
