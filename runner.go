package frags

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Runner is a struct that runs a session manager.
type Runner[T any] struct {
	sessionManager SessionManager
	resourceLoader ResourceLoader
	ai             Ai
	dataStructure  *T
	unmarshalMutex sync.Mutex
	sessionChan    chan sessionTask
	sessionWorkers int
	wg             sync.WaitGroup
	running        bool
	logger         *slog.Logger
}

// sessionTask is a message to run a session.
type sessionTask struct {
	id      string
	session Session
	timeout time.Duration
}

// RunnerOptions are options for the runner.
type RunnerOptions struct {
	sessionWorkers int
	logger         *slog.Logger
}

type RunnerOption func(*RunnerOptions)

// WithLogger sets the logger for the runner.
func WithLogger(logger *slog.Logger) RunnerOption {
	return func(o *RunnerOptions) {
		o.logger = logger
	}
}

// WithSessionWorkers sets the number of workers for the runner.
func WithSessionWorkers(sessionWorkers int) RunnerOption {
	return func(o *RunnerOptions) {
		o.sessionWorkers = sessionWorkers
	}
}

// NewRunner creates a new runner.
func NewRunner[T any](sessionManager SessionManager, resourceLoader ResourceLoader, ai Ai, options ...RunnerOption) Runner[T] {
	opts := RunnerOptions{
		sessionWorkers: 1,
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}
	for _, opt := range options {
		opt(&opts)
	}
	return Runner[T]{
		sessionManager: sessionManager,
		resourceLoader: resourceLoader,
		ai:             ai,
		unmarshalMutex: sync.Mutex{},
		sessionWorkers: opts.sessionWorkers,
		logger:         opts.logger,
	}
}

// Run runs the runner.
func (r *Runner[T]) Run() (*T, error) {
	if r.running {
		return nil, errors.New("this frags instance is running")
	}
	r.running = true
	r.sessionChan = make(chan sessionTask)
	r.dataStructure = new(T)
	for i := 0; i < r.sessionWorkers; i++ {
		r.logger.Debug("starting session worker", "index", i)
		go r.runSessionWorker(i)
	}
	r.wg = sync.WaitGroup{}
	for k, s := range r.sessionManager.Sessions {
		r.wg.Add(1)
		r.logger.Debug("sending message to workers for session", "session", k)
		timeout := 10 * time.Minute
		if s.Timeout != nil {
			var err error
			timeout, err = time.ParseDuration(*s.Timeout)
			if err != nil {
				return nil, err
			}
		}
		r.sessionChan <- sessionTask{
			id:      k,
			session: s,
			timeout: timeout,
		}
	}
	r.wg.Wait()
	close(r.sessionChan)
	r.running = false
	return r.dataStructure, nil
}

// runSession runs a session.
func (r *Runner[T]) runSession(ctx context.Context, sessionID string, session Session) error {
	resources, err := r.loadSessionResources(session)
	if err != nil {
		return err
	}
	sessionSchema, err := r.sessionManager.Schema.GetSession(sessionID)
	if err != nil {
		return err
	}
	for idx, phaseIndex := range sessionSchema.GetPhaseIndexes() {
		deadline, _ := ctx.Deadline()
		if time.Now().After(deadline) {
			return ctx.Err()
		}
		phaseSchema, err := sessionSchema.GetPhase(phaseIndex)
		if err != nil {
			return err
		}
		var data []byte
		if idx == 0 {
			data, err = r.ai.Ask(ctx, session.Prompt, phaseSchema, resources...)
			if err != nil {
				return err
			}
		} else {
			data, err = r.ai.Ask(ctx, session.NextPhasePrompt, phaseSchema, resources...)
		}
		if err := r.safeUnmarshal(data); err != nil {
			return err
		}
	}
	return nil
}

// loadSessionResources loads resources for a session.
func (r *Runner[T]) loadSessionResources(session Session) ([]Resource, error) {
	resources := make([]Resource, 0)
	for _, resourceID := range session.Resources {
		resource, err := r.resourceLoader.LoadResource(resourceID)
		if err != nil {
			return resources, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// safeUnmarshal is a thread safe version of json.Unmarshal.
func (r *Runner[T]) safeUnmarshal(data []byte) error {
	r.unmarshalMutex.Lock()
	defer r.unmarshalMutex.Unlock()
	err := json.Unmarshal(data, r.dataStructure)
	return err
}

// runSessionWorker runs a session worker.
func (r *Runner[T]) runSessionWorker(index int) {
	for t := range r.sessionChan {
		r.logger.Debug("worker consuming message", "workerID", index, "sessionID", t.id)
		func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, t.timeout)
			defer func() {
				cancel()
				r.logger.Debug("worker finished consuming message", "workerID", index, "sessionID", t.id)
				r.wg.Done()
			}()
			if err := r.runSession(ctx, t.id, t.session); err != nil {
				r.logger.Error("worked failed at running session", "workerID", index, "sessionID", t.id, "err", err)
				return
			}
		}()
	}
}
