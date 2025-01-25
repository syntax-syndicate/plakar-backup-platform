package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
)

func stringToDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

type Scheduler struct {
	configuration []config.Task
	tasksSet      []*TaskSet
}

func NewScheduler(ctx *appcontext.AppContext, configuration []config.Task) *Scheduler {
	return &Scheduler{
		configuration: configuration,
	}
}

func (s *Scheduler) Run() {
	wg := sync.WaitGroup{}
	for _, tasks := range s.configuration {
		taskSet := NewTaskSet(tasks.Name, tasks.Repository.URL)
		for _, backupTask := range tasks.Backup {
			tmp := NewBackupTask(taskSet.Name,
				backupTask.Description,
				backupTask.Path,
				stringToDuration(backupTask.Interval),
				stringToDuration(backupTask.Retention))
			taskSet.Backup = append(taskSet.Backup, tmp)
		}
		for _, checkTask := range tasks.Check {
			tmp := NewCheckTask(taskSet.Name,
				checkTask.Description,
				checkTask.Path,
				stringToDuration(checkTask.Interval))
			taskSet.Check = append(taskSet.Check, tmp)
		}
		s.tasksSet = append(s.tasksSet, taskSet)
	}

	for _, taskSet := range s.tasksSet {
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

type Task interface {
	Interval() time.Duration
}

type BackupTask struct {
	TaskSet     string
	Description string
	Path        string
	interval    time.Duration
	retention   time.Duration
}

func (t *BackupTask) Interval() time.Duration {
	return t.interval
}

func (t *BackupTask) Retention() time.Duration {
	return t.retention
}

func (t *BackupTask) Run() {
	for {
		fmt.Printf("[%s] taskSet %s: task %s: running backup task\n", time.Now().UTC().Format(time.RFC3339), t.TaskSet, t.Description)
		time.Sleep(t.Interval())
	}
}

func NewBackupTask(taskset string, description string, path string, interval time.Duration, retention time.Duration) *BackupTask {
	return &BackupTask{
		TaskSet:     taskset,
		Description: description,
		Path:        path,
		interval:    interval,
		retention:   retention,
	}
}

type CheckTask struct {
	TaskSet     string
	Description string
	Path        string
	interval    time.Duration
}

func (t *CheckTask) Interval() time.Duration {
	return t.interval
}

func (t *CheckTask) Run() {
	for {
		fmt.Printf("[%s] taskSet %s: task %s: running check task\n", time.Now().UTC().Format(time.RFC3339), t.TaskSet, t.Description)
		time.Sleep(t.Interval())
	}
}

func NewCheckTask(taskset string, description string, path string, interval time.Duration) *CheckTask {
	return &CheckTask{
		TaskSet:     taskset,
		Description: description,
		Path:        path,
		interval:    interval,
	}
}

//

type TaskSet struct {
	Name       string
	Repository string
	Backup     []*BackupTask
	Check      []*CheckTask
}

func NewTaskSet(name string, repository string) *TaskSet {
	return &TaskSet{
		Name:       name,
		Repository: repository,
	}
}
