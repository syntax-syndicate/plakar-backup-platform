package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
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
	retention, err := stringToDuration(task.Retention)
	if err != nil {
		return err
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
			} else {
				fmt.Println("Backup succeeded")

				rmSubcommand := &rm.Rm{}
				rmSubcommand.RepositoryLocation = taskset.Repository.URL
				if taskset.Repository.Passphrase != "" {
					rmSubcommand.RepositorySecret = []byte(taskset.Repository.Passphrase)
					_ = rmSubcommand.RepositorySecret
				}
				rmSubcommand.Job = task.Name
				rmSubcommand.BeforeDate = time.Now().Add(-retention)
				retval, err := rmSubcommand.Execute(s.ctx, repo)
				if err != nil || retval != 0 {
					fmt.Println("Error executing rm task: ", err)
				} else {
					fmt.Println("Retention succeeded")
				}
			}

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
	checkSubcommand.Job = taskset.Name
	checkSubcommand.Silent = true

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

func (s *Scheduler) Run() {
	for _, tasksetCfg := range s.config.Agent.TaskSets {
		fmt.Println(tasksetCfg)
		for _, backupCfg := range tasksetCfg.Backup {
			err := s.backupTask(tasksetCfg, backupCfg)
			if err != nil {
				fmt.Println("Error configuring backup task: ", err)
			}
		}
		for _, checkCfg := range tasksetCfg.Check {
			err := s.checkTask(tasksetCfg, checkCfg)
			if err != nil {
				fmt.Println("Error configuring backup task: ", err)
			}
		}
	}
	<-make(chan struct{})
}
