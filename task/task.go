package task


import (
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/maintenance"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/sync"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/reporting"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/services"
)


func RunCommand(ctx *appcontext.AppContext, cmd subcommands.Subcommand, repo *repository.Repository, taskName string) (int, error) {

	var taskKind string
	switch cmd.(type) {
	case *backup.Backup:
		taskKind = "backup"
	case *check.Check:
		taskKind = "check"
	case *restore.Restore:
		taskKind = "restore"
	case *sync.Sync:
		taskKind = "sync"
	case *rm.Rm:
		taskKind = "rm"
	case *maintenance.Maintenance:
		taskKind = "maintenance"
	}

	var doReport bool
	if repo != nil && taskKind != "" {
		authToken, err := ctx.GetAuthToken(repo.Configuration().RepositoryID)
		if err == nil && authToken != "" {
			sc := services.NewServiceConnector(ctx, authToken)
			enabled, err := sc.GetServiceStatus("alerting")
			if err == nil && enabled {
				doReport = true
			}
		}
	}

	reporter := reporting.NewReporter(doReport, repo, ctx.GetLogger())
	reporter.TaskStart(taskKind, taskName)
	if repo != nil {
		reporter.WithRepositoryName(repo.Location())
		reporter.WithRepository(repo)
	}

	var err error
	var status int
	var snapshotID objects.MAC
	var warning error
	if _, ok := cmd.(*backup.Backup); ok {
		cmd := cmd.(*backup.Backup)
		status, err, snapshotID, warning = cmd.DoBackup(ctx, repo)
		if err == nil {
			reporter.WithSnapshotID(snapshotID)
		}
	} else {
		status, err = cmd.Execute(ctx, repo)
	}

	if status == 0 {
		if warning != nil {
			reporter.TaskWarning("warning: %s", warning)
		} else {
			reporter.TaskDone()
		}
	} else if err != nil {
		reporter.TaskFailed(0, "error: %s", err)
	}

	return status, err
}
