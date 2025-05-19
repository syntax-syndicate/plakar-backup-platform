package backup

import (
	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/charmbracelet/lipgloss"
)

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
)

type eventsProcessor interface {
	Close()
}

func startEventsProcessor(ctx *appcontext.AppContext, basepath string, opt_stdio bool, opt_quiet bool) eventsProcessor {
	//if !opt_stdio && !opt_quiet && term.IsTerminal(int(os.Stdout.Fd())) {
	//	return startEventsProcessorInteractive(ctx, basepath)
	//}
	return startEventsProcessorStdio(ctx, opt_quiet)
}
