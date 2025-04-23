package check

import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/events"
	"github.com/charmbracelet/lipgloss"
)

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
)

func eventsProcessorStdio(ctx *appcontext.AppContext, quiet bool) chan struct{} {
	done := make(chan struct{})
	go func() {
		for event := range ctx.Events().Listen() {
			switch event := event.(type) {
			case events.DirectoryMissing:
				ctx.GetLogger().Warn("%x: %s %s: missing directory", event.SnapshotID[:4], crossMark, event.Pathname)
			case events.FileMissing:
				ctx.GetLogger().Warn("%x: %s %s: missing file", event.SnapshotID[:4], crossMark, event.Pathname)
			case events.ObjectMissing:
				ctx.GetLogger().Warn("%x: %s %x: missing object", event.SnapshotID[:4], crossMark, event.MAC)
			case events.ChunkMissing:
				ctx.GetLogger().Warn("%x: %s %x: missing chunk", event.SnapshotID[:4], crossMark, event.MAC)

			case events.DirectoryCorrupted:
				ctx.GetLogger().Warn("%x: %s %s: corrupted directory", event.SnapshotID[:4], crossMark, event.Pathname)
			case events.FileCorrupted:
				ctx.GetLogger().Warn("%x: %s %s: corrupted file", event.SnapshotID[:4], crossMark, event.Pathname)
			case events.ObjectCorrupted:
				ctx.GetLogger().Warn("%x: %s %x: corrupted object", event.SnapshotID[:4], crossMark, event.MAC)
			case events.ChunkCorrupted:
				ctx.GetLogger().Warn("%x: %s %x: corrupted chunk", event.SnapshotID[:4], crossMark, event.MAC)

			case events.DirectoryOK:
				if !quiet {
					ctx.GetLogger().Info("%x: %s %s", event.SnapshotID[:4], checkMark, event.Pathname)
				}
			case events.FileOK:
				if !quiet {
					ctx.GetLogger().Info("%x: %s %s", event.SnapshotID[:4], checkMark, event.Pathname)
				}

			case events.DirectoryError:
				ctx.GetLogger().Stderr("%x: KO %s %s: %s", event.SnapshotID[:4], crossMark, event.Pathname, event.Message)
			case events.FileError:
				ctx.GetLogger().Stderr("%x: KO %s %s: %s", event.SnapshotID[:4], crossMark, event.Pathname, event.Message)
			default:
			}
		}
		done <- struct{}{}
	}()
	return done
}
