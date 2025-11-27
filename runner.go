package frags

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/expr-lang/expr"
)

// Runner is a struct that runs a session manager.
type Runner[T any] struct {
	sessionManager  SessionManager
	status          map[string]SessionStatus
	resourceLoader  ResourceLoader
	ai              Ai
	dataStructure   *T
	params          any
	unmarshalMutex  sync.Mutex
	statusMutex     sync.Mutex
	sessionChan     chan sessionTask
	sessionWorkers  int
	wg              sync.WaitGroup
	running         bool
	logger          *slog.Logger
	progressChannel chan ProgressMessage
}

type SessionStatus string

var queuedSessionStatus = SessionStatus("queued")
var committedSessionStatus = SessionStatus("committed")
var runningSessionStatus = SessionStatus("running")
var finishedSessionStatus = SessionStatus("finished")
var failedSessionStatus = SessionStatus("failed")
var noOpSessionStatus = SessionStatus("noop")

// sessionTask is a message to run a session.
type sessionTask struct {
	id      string
	session Session
	timeout time.Duration
}

type ProgressMessage struct {
	Action  string
	Session string
	Phase   int
	Error   error
}

// RunnerOptions are options for the runner.
type RunnerOptions struct {
	sessionWorkers  int
	logger          *slog.Logger
	progressChannel chan ProgressMessage
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

func WithProgressChannel(progressChannel chan ProgressMessage) RunnerOption {
	return func(o *RunnerOptions) {
		o.progressChannel = progressChannel
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
	status := make(map[string]SessionStatus)
	for k, _ := range sessionManager.Sessions {
		status[k] = queuedSessionStatus
	}
	return Runner[T]{
		sessionManager:  sessionManager,
		status:          status,
		resourceLoader:  resourceLoader,
		ai:              ai,
		unmarshalMutex:  sync.Mutex{},
		statusMutex:     sync.Mutex{},
		sessionWorkers:  opts.sessionWorkers,
		logger:          opts.logger,
		progressChannel: opts.progressChannel,
	}
}

// Run runs the runner.
func (r *Runner[T]) Run(params any) (*T, error) {
	if r.running {
		return nil, errors.New("this frags instance is running")
	}
	r.params = params
	r.running = true
	r.sessionChan = make(chan sessionTask)
	var v T
	val := reflect.ValueOf(&v).Elem()
	if val.Kind() == reflect.Map {
		val.Set(reflect.MakeMap(val.Type()))
		r.dataStructure = &v
	} else {
		r.dataStructure = new(T)
	}
	for i := 0; i < r.sessionWorkers; i++ {
		r.logger.Debug("starting session worker", "index", i)
		go r.runSessionWorker(i)
	}
	for !r.IsCompleted() {
		if err := r.scanSessions(); err != nil {
			r.logger.Error("failed to scan sessions", "err", err)
		}
	}
	close(r.sessionChan)
	r.running = false
	return r.dataStructure, nil
}

func (r *Runner[T]) scanSessions() error {
	r.wg = sync.WaitGroup{}
	for k, s := range r.ListQueued() {
		depCheck, err := r.CheckDependencies(s.DependsOn)
		if err != nil {
			return err
		}
		switch depCheck {
		case DependencyCheckFailed:
			continue
		case DependencyCheckUnsolvable:
			r.SetStatus(k, noOpSessionStatus)
			continue
		}

		r.wg.Add(1)
		r.logger.Debug("sending message to workers for session", "session", k)
		timeout := 10 * time.Minute
		if s.Timeout != nil {
			var err error
			timeout, err = time.ParseDuration(*s.Timeout)
			if err != nil {
				return err
			}
		}
		r.SetStatus(k, committedSessionStatus)
		r.sessionChan <- sessionTask{
			id:      k,
			session: s,
			timeout: timeout,
		}
	}
	r.wg.Wait()
	return nil
}

// runSession runs a session.
func (r *Runner[T]) runSession(ctx context.Context, sessionID string, session Session) error {
	r.SetStatus(sessionID, runningSessionStatus)
	resources, err := r.loadSessionResources(session)
	if err != nil {
		return err
	}
	sessionSchema, err := r.sessionManager.Schema.GetSession(sessionID)
	if err != nil {
		return err
	}
	ai := r.ai.New()
	for idx, phaseIndex := range sessionSchema.GetPhaseIndexes() {
		r.sendProgress("START", sessionID, phaseIndex, nil)
		deadline, _ := ctx.Deadline()
		if time.Now().After(deadline) {
			r.sendProgress("END", sessionID, phaseIndex, ctx.Err())
			return ctx.Err()
		}
		phaseSchema, err := sessionSchema.GetPhase(phaseIndex)
		if err != nil {
			r.sendProgress("END", sessionID, phaseIndex, err)
			return err
		}
		var data []byte
		if idx == 0 {
			prompt, err := session.RenderPrompt(r.params)
			if err != nil {
				r.sendProgress("END", sessionID, phaseIndex, err)
				return err
			}
			data, err = ai.Ask(ctx, prompt, phaseSchema, resources...)
			if err != nil {
				r.sendProgress("END", sessionID, phaseIndex, err)
				return err
			}
		} else {
			prompt, err := session.RenderNextPhasePrompt(r.params)
			if err != nil {
				r.sendProgress("END", sessionID, phaseIndex, err)
				return err
			}
			data, err = ai.Ask(ctx, prompt, phaseSchema)
			if err != nil {
				r.sendProgress("END", sessionID, phaseIndex, err)
				return err
			}
		}
		if err := r.safeUnmarshal(data); err != nil {
			r.sendProgress("END", sessionID, phaseIndex, err)
			return err
		}
		r.sendProgress("END", sessionID, phaseIndex, nil)
	}
	return nil
}

func (r *Runner[T]) ListQueued() Sessions {
	sessions := make(Sessions)
	for k, v := range r.status {
		if v == queuedSessionStatus {
			sessions[k] = r.sessionManager.Sessions[k]
		}
	}
	return sessions
}

func (r *Runner[T]) sendProgress(action string, sessionID string, phaseIndex int, err error) {
	if r.progressChannel != nil {
		r.progressChannel <- ProgressMessage{
			Action:  action,
			Session: sessionID,
			Phase:   phaseIndex,
			Error:   err,
		}
	}
}

// loadSessionResources loads resources for a session.
func (r *Runner[T]) loadSessionResources(session Session) ([]ResourceData, error) {
	resources := make([]ResourceData, 0)
	for _, resource := range session.Resources {
		resourceData, err := r.resourceLoader.LoadResource(resource.Identifier, resource.Params)
		if err != nil {
			return resources, err
		}
		resources = append(resources, resourceData)
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
			success := true
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, t.timeout)
			defer func() {
				cancel()
				if success {
					r.SetStatus(t.id, finishedSessionStatus)
				} else {
					r.SetStatus(t.id, failedSessionStatus)
				}
				r.logger.Debug("worker finished consuming message", "workerID", index, "sessionID", t.id)
				r.wg.Done()
			}()
			if err := r.runSession(ctx, t.id, t.session); err != nil {
				success = false
				r.logger.Error("worked failed at running session", "workerID", index, "sessionID", t.id, "err", err)
				return
			}
		}()
	}
}

func (r *Runner[T]) SetStatus(sessionID string, status SessionStatus) {
	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()
	r.status[sessionID] = status
}

func (r *Runner[T]) IsCompleted() bool {
	for _, s := range r.status {
		if s == queuedSessionStatus {
			return false
		}
	}
	return true
}

type DependencyCheckResult string

const (
	DependencyCheckPassed     DependencyCheckResult = "passed"
	DependencyCheckFailed     DependencyCheckResult = "failed"
	DependencyCheckUnsolvable DependencyCheckResult = "unsolvable"
)

func (r *Runner[T]) CheckDependencies(dependencies Dependencies) (DependencyCheckResult, error) {
	if dependencies == nil {
		return DependencyCheckPassed, nil
	}
	for _, dep := range dependencies {
		if dep.Session != nil {
			dependencyStatus := r.status[*dep.Session]
			if slices.Contains([]SessionStatus{failedSessionStatus, noOpSessionStatus}, dependencyStatus) {
				return DependencyCheckUnsolvable, nil
			}
			if slices.Contains([]SessionStatus{queuedSessionStatus, committedSessionStatus, runningSessionStatus}, dependencyStatus) {
				return DependencyCheckFailed, nil
			}
		}
		if dep.Expression != nil {
			c, err := expr.Compile(*dep.Expression, expr.Env(*r.dataStructure))
			if err != nil {
				return DependencyCheckUnsolvable, err
			}
			res, err := expr.Run(c, *r.dataStructure)
			if err != nil {
				return DependencyCheckUnsolvable, err
			}
			if !res.(bool) {
				return DependencyCheckUnsolvable, nil
			}
		}
	}
	return DependencyCheckPassed, nil
}
