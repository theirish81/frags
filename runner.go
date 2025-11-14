package frags

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sync"
)

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

type sessionTask struct {
	id      string
	session Session
}

type RunnerOptions struct {
	sessionWorkers int
	logger         *slog.Logger
}

type RunnerOption func(*RunnerOptions)

func WithLogger(logger *slog.Logger) RunnerOption {
	return func(o *RunnerOptions) {
		o.logger = logger
	}
}

func WithSessionWorkers(sessionWorkers int) RunnerOption {
	return func(o *RunnerOptions) {
		o.sessionWorkers = sessionWorkers
	}
}
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
		r.sessionChan <- sessionTask{
			id:      k,
			session: s,
		}
	}
	r.wg.Wait()
	close(r.sessionChan)
	r.running = false
	return r.dataStructure, nil
}

func (r *Runner[T]) runSession(sessionID string, session Session) error {
	resources, err := r.loadSessionResources(session)
	if err != nil {
		return err
	}
	sessionSchema, err := r.sessionManager.Schema.GetSession(sessionID)
	if err != nil {
		return err
	}
	for idx, phaseIndex := range sessionSchema.GetPhaseIndexes() {
		phaseSchema, err := sessionSchema.GetPhase(phaseIndex)
		if err != nil {
			return err
		}
		var data []byte
		if idx == 0 {
			data, err = r.ai.Ask(session.Prompt, phaseSchema, resources...)
			if err != nil {
				return err
			}
		} else {
			data, err = r.ai.Ask(session.NextPhasePrompt, phaseSchema, resources...)
		}
		if err := r.safeUnmarshal(data); err != nil {
			return err
		}
	}
	return nil
}

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

func (r *Runner[T]) safeUnmarshal(data []byte) error {
	r.unmarshalMutex.Lock()
	defer r.unmarshalMutex.Unlock()
	err := json.Unmarshal(data, r.dataStructure)
	return err
}

func (r *Runner[T]) runSessionWorker(index int) {
	for t := range r.sessionChan {
		r.logger.Debug("worker consuming message", "workerID", index, "sessionID", t.id)
		func() {
			defer func() {
				r.logger.Debug("worker finished consuming message", "workerID", index, "sessionID", t.id)
				r.wg.Done()
			}()
			if err := r.runSession(t.id, t.session); err != nil {
				r.logger.Error("worked failed at running session", "workerID", index, "sessionID", t.id, "err", err)
				return
			}
		}()

	}
}
