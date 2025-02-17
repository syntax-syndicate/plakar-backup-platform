package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
)

type Scheduler struct {
	config *Configuration
	ctx    *appcontext.AppContext
	wg     sync.WaitGroup
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

func (s *Scheduler) Run() {
	for _, tasksetCfg := range s.config.Agent.TaskSets {
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
