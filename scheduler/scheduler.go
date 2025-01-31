package scheduler

import (
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
)

func stringToDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

type Scheduler struct {
	ctx       *appcontext.AppContext
	tasksSets []*TaskSet
}

func NewScheduler(ctx *appcontext.AppContext) *Scheduler {
	return &Scheduler{
		ctx:       ctx,
		tasksSets: []*TaskSet{},
	}
}

func (s *Scheduler) configure() {
	for _, tasksetCfg := range s.ctx.Configuration.Agent.Tasks {
		s.tasksSets = append(s.tasksSets, NewTaskSet(s, &tasksetCfg))
	}
}

func (s *Scheduler) Run() {

	s.configure()

	wg := sync.WaitGroup{}
	for _, taskSet := range s.tasksSets {
		for _, backupTask := range taskSet.Backup {
			wg.Add(1)
			go func() {
				defer wg.Done()
				backupTask.Run()
			}()
		}
		for _, checkTask := range taskSet.Check {
			wg.Add(1)
			go func() {
				defer wg.Done()
				checkTask.Run()
			}()
		}
	}

	wg.Wait()

}
