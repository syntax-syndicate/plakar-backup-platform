package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
)

type Scheduler struct {
	config    *Configuration
	ctx       *appcontext.AppContext
	tasksSets []*TaskSet
	wg        sync.WaitGroup
}

func stringToDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return d, nil
}

func NewScheduler(ctx *appcontext.AppContext, config *Configuration) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		config: config,
		wg:     sync.WaitGroup{},
	}
}

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
				fmt.Println("Error opening storage: ", err)
				continue
			}

			repo, err := repository.New(s.ctx, store, config)
			if err != nil {
				fmt.Println("Error creating repository: ", err)
				store.Close()
				continue
			}

			retval, err := backupSubcommand.Execute(s.ctx, repo)
			if err != nil || retval != 0 {
				fmt.Println("Error executing backup: ", err)
				goto close
			}

			if task.Check {
				retval, err = checkSubcommand.Execute(s.ctx, repo)
				if err != nil || retval != 0 {
					fmt.Println("Error executing check task: ", err)
					goto close
				}
			}

			if task.Retention != "" {
				rmSubcommand.OptBefore = time.Now().Add(-retention)
				retval, err = rmSubcommand.Execute(s.ctx, repo)
				if err != nil || retval != 0 {
					fmt.Println("Error executing rm task: ", err)
				}
			}

		close:
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
	checkSubcommand.Silent = false

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
				fmt.Println("Error opening storage: ", err)
				continue
			}

			repo, err := repository.New(s.ctx, store, config)
			if err != nil {
				fmt.Println("Error creating repository: ", err)
				store.Close()
				continue
			}

			retval, err := checkSubcommand.Execute(s.ctx, repo)
			if err != nil || retval != 0 {
				fmt.Println("Error executing check: ", err)
			} else {
				fmt.Println("Check succeeded")
			}
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

	fmt.Println("Restore path", task.Path, "for job", taskset.Name, "io target", task.Target)

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
				fmt.Println("Error opening storage: ", err)
				continue
			}

			repo, err := repository.New(s.ctx, store, config)
			if err != nil {
				fmt.Println("Error creating repository: ", err)
				store.Close()
				continue
			}

			//retval, err := restoreSubcommand.Execute(s.ctx, repo)
			retval := 1
			if err != nil || retval != 0 {
				fmt.Println("Error executing check: ", err)
			} else {
				fmt.Println("Restore succeeded")
			}
			repo.Close()
			store.Close()
		}
	}()

	return nil
}

func (s *Scheduler) Run() {
	for _, tasksetCfg := range s.config.Agent.TaskSets {
		fmt.Println(tasksetCfg)
		if tasksetCfg.Backup != nil {
			err := s.backupTask(tasksetCfg, *tasksetCfg.Backup)
			if err != nil {
				fmt.Println("Error configuring backup task: ", err)
			}
		}
		for _, checkCfg := range tasksetCfg.Check {
			err := s.checkTask(tasksetCfg, checkCfg)
			if err != nil {
				fmt.Println("Error configuring check task: ", err)
			}
		}

		for _, restoreCfg := range tasksetCfg.Restore {
			err := s.restoreTask(tasksetCfg, restoreCfg)
			if err != nil {
				fmt.Println("Error configuring restore task: ", err)
			}
		}
	}
	<-make(chan struct{})
}
