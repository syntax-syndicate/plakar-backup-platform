package scheduler

import (
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
)

func (s *Scheduler) backupTask(taskset TaskSet, task BackupConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	var retention time.Duration
	if task.Retention != "" {
		retention, err = stringToDuration(task.Retention)
		if err != nil {
			return err
		}
	}

	backupSubcommand := &backup.Backup{}
	backupSubcommand.RepositoryLocation = taskset.Repository.URL
	if taskset.Repository.Passphrase != "" {
		backupSubcommand.RepositorySecret = []byte(taskset.Repository.Passphrase)
		_ = backupSubcommand.RepositorySecret
	}
	backupSubcommand.Silent = true
	backupSubcommand.Job = taskset.Name
	backupSubcommand.Path = task.Path
	backupSubcommand.Quiet = true

	checkSubcommand := &check.Check{}
	checkSubcommand.RepositoryLocation = taskset.Repository.URL
	if taskset.Repository.Passphrase != "" {
		checkSubcommand.RepositorySecret = []byte(taskset.Repository.Passphrase)
		_ = checkSubcommand.RepositorySecret
	}
	checkSubcommand.OptJob = task.Name
	checkSubcommand.Silent = true
	checkSubcommand.OptLatest = true

	rmSubcommand := &rm.Rm{}
	rmSubcommand.RepositoryLocation = taskset.Repository.URL
	if taskset.Repository.Passphrase != "" {
		rmSubcommand.RepositorySecret = []byte(taskset.Repository.Passphrase)
		_ = rmSubcommand.RepositorySecret
	}
	rmSubcommand.OptJob = task.Name

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			store, config, err := storage.Open(backupSubcommand.RepositoryLocation)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			backupCtx := appcontext.NewAppContextFrom(newCtx)
			retval, err := backupSubcommand.Execute(backupCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error creating backup: %s", err)
				backupCtx.Close()
				goto close
			} else {
				s.ctx.GetLogger().Info("Backup succeeded")
			}
			backupCtx.Close()

			if task.Check {
				checkCtx := appcontext.NewAppContextFrom(newCtx)
				retval, err = checkSubcommand.Execute(checkCtx, repo)
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error checking backup: %s", err)
					checkCtx.Close()
					goto close
				} else {
					s.ctx.GetLogger().Info("Backup succeeded")
				}
				checkCtx.Close()
			}

			if task.Retention != "" {
				rmCtx := appcontext.NewAppContextFrom(newCtx)
				rmSubcommand.OptBefore = time.Now().Add(-retention)
				retval, err = rmSubcommand.Execute(rmCtx, repo)
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
				} else {
					s.ctx.GetLogger().Info("Retention purge succeeded")
				}
				rmCtx.Close()
			}

		close:
			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) checkTask(taskset TaskSet, task CheckConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	checkSubcommand := &check.Check{}
	checkSubcommand.RepositoryLocation = taskset.Repository.URL
	if taskset.Repository.Passphrase != "" {
		checkSubcommand.RepositorySecret = []byte(taskset.Repository.Passphrase)
		_ = checkSubcommand.RepositorySecret
	}
	checkSubcommand.OptJob = taskset.Name
	checkSubcommand.OptLatest = task.Latest
	checkSubcommand.Silent = true
	if task.Path != "" {
		checkSubcommand.Snapshots = []string{":" + task.Path}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			store, config, err := storage.Open(checkSubcommand.RepositoryLocation)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			retval, err := checkSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing check: %s", err)
			} else {
				s.ctx.GetLogger().Info("Check succeeded")
			}

			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) restoreTask(taskset TaskSet, task RestoreConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	restoreSubcommand := &restore.Restore{}
	restoreSubcommand.RepositoryLocation = taskset.Repository.URL
	if taskset.Repository.Passphrase != "" {
		restoreSubcommand.RepositorySecret = []byte(taskset.Repository.Passphrase)
		_ = restoreSubcommand.RepositorySecret
	}
	restoreSubcommand.OptJob = taskset.Name
	restoreSubcommand.Target = task.Target
	restoreSubcommand.Silent = true
	if task.Path != "" {
		restoreSubcommand.Snapshots = []string{":" + task.Path}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		firstRun := true
		for {
			if firstRun {
				firstRun = false
			} else {
				time.Sleep(interval)
			}

			store, config, err := storage.Open(restoreSubcommand.RepositoryLocation)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening storage: %s", err)
				continue
			}

			newCtx := appcontext.NewAppContextFrom(s.ctx)

			repo, err := repository.New(newCtx, store, config)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				store.Close()
				continue
			}

			retval, err := restoreSubcommand.Execute(newCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing restore: %s", err)
			} else {
				s.ctx.GetLogger().Info("Restore succeeded")
			}

			newCtx.Close()
			repo.Close()
			store.Close()
		}
	}()

	return nil
}
