package status

import (
	"context"
	"sync"
	"time"
)

// Status represents the lifecycle state of a job.
type JobStatus string

const (
	JobPending JobStatus = "pending"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
)

// Job describes a unit of work in the queue.
type Job struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Status      JobStatus  `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Error       string     `json:"error,omitempty"`

	fn func(context.Context) error
}

// Queue holds pending, running, and recently completed jobs.
// Jobs execute sequentially in a single worker goroutine.
type JobQueue struct {
	mu         sync.Mutex
	jobs       []Job
	notify     chan struct{}
	maxHistory int
	done       *sync.Cond
}

// New creates a queue that retains up to maxHistory completed jobs.
func NewJobQueue(maxHistory int) *JobQueue {
	if maxHistory <= 0 {
		maxHistory = 20
	}
	q := &JobQueue{
		notify:     make(chan struct{}, 1),
		maxHistory: maxHistory,
	}
	q.done = sync.NewCond(&q.mu)
	return q
}

// Submit adds a job to the queue and returns its ID.
func (q *JobQueue) Submit(jobType, description string, fn func(context.Context) error) string {
	id := generateID(jobType)
	q.SubmitWithID(id, jobType, description, fn)
	return id
}

// SubmitWithID adds a job with a specific ID (used for resuming interrupted jobs).
func (q *JobQueue) SubmitWithID(id, jobType, description string, fn func(context.Context) error) {
	q.mu.Lock()
	q.jobs = append(q.jobs, Job{
		ID:          id,
		Type:        jobType,
		Description: description,
		Status:      JobPending,
		CreatedAt:   time.Now(),
		fn:          fn,
	})
	q.mu.Unlock()
	q.signal()
}

// Snapshot returns a copy of all jobs (pending, running, recently completed).
func (q *JobQueue) Snapshot() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Job, len(q.jobs))
	copy(out, q.jobs)
	for i := range out {
		out[i].fn = nil
	}
	return out
}

// RunningJobID returns the ID of the currently running job, or empty string.
func (q *JobQueue) RunningJobID() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		if q.jobs[i].Status == JobRunning {
			return q.jobs[i].ID
		}
	}
	return ""
}

// Wait blocks until the job with the given ID is no longer running or pending.
// It returns true if the job was found and reached a terminal state.
func (q *JobQueue) Wait(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		found := false
		for i := range q.jobs {
			if q.jobs[i].ID == id {
				found = true
				if q.jobs[i].Status == JobDone || q.jobs[i].Status == JobFailed {
					return true
				}
				break
			}
		}
		if !found {
			return false
		}
		q.done.Wait()
	}
}

// Start launches the worker goroutine. It blocks until ctx is cancelled.
func (q *JobQueue) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.notify:
			q.processNext(ctx)
		}
	}
}

func (q *JobQueue) processNext(ctx context.Context) {
	q.mu.Lock()
	idx := -1
	for i := range q.jobs {
		if q.jobs[i].Status == JobPending {
			idx = i
			break
		}
	}
	if idx == -1 {
		q.mu.Unlock()
		return
	}

	now := time.Now()
	q.jobs[idx].Status = JobRunning
	q.jobs[idx].StartedAt = &now
	job := q.jobs[idx]
	q.mu.Unlock()

	err := job.fn(context.WithoutCancel(ctx))

	q.mu.Lock()
	fin := time.Now()
	q.jobs[idx].FinishedAt = &fin
	if err != nil {
		q.jobs[idx].Status = JobFailed
		q.jobs[idx].Error = err.Error()
	} else {
		q.jobs[idx].Status = JobDone
	}
	q.jobs[idx].fn = nil
	q.prune()
	q.done.Broadcast()
	q.mu.Unlock()

	if err == nil {
		q.signal()
	}
}

// prune removes old completed jobs beyond maxHistory, keeping at most
// maxHistory non-pending/non-running entries.
func (q *JobQueue) prune() {
	if len(q.jobs) <= q.maxHistory {
		return
	}
	kept := make([]Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		if j.Status == JobPending || j.Status == JobRunning {
			kept = append(kept, j)
		}
	}
	for _, j := range q.jobs {
		if j.Status == JobDone || j.Status == JobFailed {
			if len(kept) >= q.maxHistory {
				break
			}
			kept = append(kept, j)
		}
	}
	q.jobs = kept
}

func (q *JobQueue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func generateID(prefix string) string {
	return prefix + "-" + time.Now().UTC().Format("20060102150405.000000")
}
