package scheduler

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/backup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/check"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/cleanup"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/restore"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/rm"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands/sync"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

func (s *Scheduler) locationToConfig(location string) (map[string]string, error) {
	if strings.HasPrefix(location, "@") {
		remote, ok := s.ctx.Config.GetRepository(location[1:])
		if !ok {
			return nil, fmt.Errorf("could not resolve repository: %s", location)
		}
		if _, ok := remote["location"]; !ok {
			return nil, fmt.Errorf("could not resolve repository location: %s", location)
		} else {
			return remote, nil
		}
	}
	return map[string]string{"location": location}, nil
}

func (s *Scheduler) openRepository(storeConfig map[string]string) (*repository.Repository, error) {
	store, wrappedConfig, err := storage.Open(storeConfig)
	if err != nil {
		return nil, fmt.Errorf("error opening storage: %s", err)
	}

	repoConfig, err := storage.NewConfigurationFromWrappedBytes(wrappedConfig)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("error opening storage: %s", err)
	}

	if repoConfig.Version != versioning.FromString(storage.VERSION) {
		store.Close()
		return nil, fmt.Errorf("%s: incompatible repository version: %s != %s\n",
			flag.CommandLine.Name(), repoConfig.Version, storage.VERSION)
	}

	newCtx := appcontext.NewAppContextFrom(s.ctx)
	if repoConfig.Encryption != nil {
		var secret []byte
		derived := false
		if passphrase, ok := storeConfig["passphrase"]; ok {
			key, err := encryption.DeriveKey(repoConfig.Encryption.KDFParams, []byte(passphrase))
			if err == nil {
				if encryption.VerifyCanary(repoConfig.Encryption, key) {
					secret = key
					derived = true
				}
			}
		}
		if !derived {
			store.Close()
			return nil, fmt.Errorf("could not derive secret")

		}
		newCtx.SetSecret(secret)
	}

	repo, err := repository.New(newCtx, store, wrappedConfig)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("Error opening repository: %s", err)
	}

	return repo, nil
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

	storeConfig, err := s.locationToConfig(taskset.Repository.URL)
	if err != nil {
		return fmt.Errorf("could not resolve repository: %s", storeConfig["location"])
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
	if task.Check {
		backupSubcommand.OptCheck = true
	}

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

			repo, err := s.openRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			backupCtx := appcontext.NewAppContextFrom(repo.AppContext())
			backupCtx.SetSecret(repo.AppContext().GetSecret())
			retval, err := backupSubcommand.Execute(backupCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error creating backup: %s", err)
				backupCtx.Close()
				goto close
			}
			backupCtx.Close()

			if task.Retention != "" {
				rmCtx := appcontext.NewAppContextFrom(repo.AppContext())
				rmCtx.SetSecret(repo.AppContext().GetSecret())
				rmSubcommand.OptBefore = time.Now().Add(-retention)
				retval, err = rmSubcommand.Execute(rmCtx, repo)
				if err != nil || retval != 0 {
					s.ctx.GetLogger().Error("Error removing obsolete backups: %s", err)
				}
				rmCtx.Close()
			}

		close:
			repo.Close()
		}
	}()

	return nil
}

func (s *Scheduler) checkTask(taskset TaskSet, task CheckConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	storeConfig, err := s.locationToConfig(taskset.Repository.URL)
	if err != nil {
		return fmt.Errorf("could not resolve repository: %s", storeConfig["location"])
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

			repo, err := s.openRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			checkCtx := appcontext.NewAppContextFrom(repo.AppContext())
			checkCtx.SetSecret(repo.AppContext().GetSecret())
			retval, err := checkSubcommand.Execute(checkCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing check: %s", err)
			}
			checkCtx.Close()

			repo.Close()
		}
	}()

	return nil
}

func (s *Scheduler) restoreTask(taskset TaskSet, task RestoreConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	storeConfig, err := s.locationToConfig(taskset.Repository.URL)
	if err != nil {
		return fmt.Errorf("could not resolve repository: %s", storeConfig["location"])
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

			repo, err := s.openRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			restoreCtx := appcontext.NewAppContextFrom(repo.AppContext())
			restoreCtx.SetSecret(repo.AppContext().GetSecret())
			retval, err := restoreSubcommand.Execute(restoreCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing restore: %s", err)
			}
			restoreCtx.Close()

			repo.Close()
		}
	}()

	return nil
}

func (s *Scheduler) syncTask(taskset TaskSet, task SyncConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	storeConfig, err := s.locationToConfig(taskset.Repository.URL)
	if err != nil {
		return fmt.Errorf("could not resolve repository: %s", storeConfig["location"])
	}

	syncSubcommand := &sync.Sync{}
	syncSubcommand.SourceRepositoryLocation = taskset.Repository.URL
	if taskset.Repository.Passphrase != "" {
		syncSubcommand.SourceRepositorySecret = []byte(taskset.Repository.Passphrase)
		_ = syncSubcommand.SourceRepositorySecret
	}

	syncSubcommand.PeerRepositoryLocation = task.Peer
	if task.Direction == "to" {
		syncSubcommand.Direction = "to"
	} else if task.Direction == "from" {
		syncSubcommand.Direction = "from"
	} else if task.Direction == "with" || task.Direction == "" {
		syncSubcommand.Direction = "with"
	} else {
		return fmt.Errorf("invalid sync direction: %s", task.Direction)
	}
	//	if taskset.Repository.Passphrase != "" {
	//		syncSubcommand.DestinationRepositorySecret = []byte(taskset.Repository.Passphrase)
	//		_ = syncSubcommand.DestinationRepositorySecret

	//	syncSubcommand.OptJob = taskset.Name
	//	syncSubcommand.Target = task.Target
	//	syncSubcommand.Silent = true

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

			repo, err := s.openRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			syncCtx := appcontext.NewAppContextFrom(repo.AppContext())
			syncCtx.SetSecret(repo.AppContext().GetSecret())
			retval, err := syncSubcommand.Execute(syncCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("sync: %s", err)
			} else {
				s.ctx.GetLogger().Info("sync: synchronization succeeded")
			}
			syncCtx.Close()

			repo.Close()
		}
	}()

	return nil
}

func (s *Scheduler) cleanupTask(task CleanupConfig) error {
	interval, err := stringToDuration(task.Interval)
	if err != nil {
		return err
	}

	storeConfig, err := s.locationToConfig(task.Repository.URL)
	if err != nil {
		return fmt.Errorf("could not resolve repository: %s", storeConfig["location"])
	}

	cleanupSubcommand := &cleanup.Cleanup{}
	cleanupSubcommand.RepositoryLocation = task.Repository.URL
	if task.Repository.Passphrase != "" {
		cleanupSubcommand.RepositorySecret = []byte(task.Repository.Passphrase)
		_ = cleanupSubcommand.RepositorySecret
	}

	rmSubcommand := &rm.Rm{}
	rmSubcommand.RepositoryLocation = task.Repository.URL
	if task.Repository.Passphrase != "" {
		rmSubcommand.RepositorySecret = []byte(task.Repository.Passphrase)
		_ = rmSubcommand.RepositorySecret
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

			repo, err := s.openRepository(storeConfig)
			if err != nil {
				s.ctx.GetLogger().Error("Error opening repository: %s", err)
				continue
			}

			cleanupCtx := appcontext.NewAppContextFrom(repo.AppContext())
			cleanupCtx.SetSecret(repo.AppContext().GetSecret())

			retval, err := cleanupSubcommand.Execute(cleanupCtx, repo)
			if err != nil || retval != 0 {
				s.ctx.GetLogger().Error("Error executing cleanup: %s", err)
			} else {
				s.ctx.GetLogger().Info("maintenance of repository %s succeeded", cleanupSubcommand.RepositoryLocation)
			}
			cleanupCtx.Close()

			repo.Close()
		}
	}()

	return nil
}
