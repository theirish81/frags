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

	"github.com/avast/retry-go/v5"
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

// SessionStatus is the status of a session.
type SessionStatus string

// Session statuses.
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

// ProgressMessage is a message sent to the progress channel.
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

// RunnerOption is an option for the runner.
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

// WithProgressChannel sets the progress channel for the runner.
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

// Run runs the runner against an optional collection fo parameters
func (r *Runner[T]) Run(params any) (*T, error) {
	if r.running {
		return nil, errors.New("this frags instance is running")
	}
	r.params = params
	r.running = true
	r.sessionChan = make(chan sessionTask)
	defer func() {
		close(r.sessionChan)
	}()
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
			return r.dataStructure, err
		}
	}
	r.running = false
	return r.dataStructure, nil
}

// scanSessions keeps scanning sessions until completion, sending tasks to workers and orchestrating priority and
// concurrency
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
	if session.Attempts <= 0 {
		session.Attempts = 1
	}
	ai := r.ai.New()
	for idx, phaseIndex := range sessionSchema.GetPhaseIndexes() {
		err := retry.New(retry.Attempts(uint(session.Attempts)), retry.Delay(time.Second*5), retry.Context(ctx)).Do(func() error {
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
			scope := map[string]any{
				"components": r.sessionManager.Components,
				"context":    *r.dataStructure,
				"params":     r.params,
			}
			if idx == 0 {
				prompt, err := session.RenderPrompt(scope)
				if err != nil {
					r.sendProgress("END", sessionID, phaseIndex, err)
					return err
				}
				p := prompt
				if session.Context {
					llmContext, err := json.MarshalIndent(r.dataStructure, "", " ")
					if err != nil {
						return err
					}
					p = "=== CURRENT CONTEXT ===\n" + string(llmContext) + "\n===\n\n" + p
				}
				data, err = ai.Ask(ctx, p, phaseSchema, resources...)
				if err != nil {
					r.sendProgress("END", sessionID, phaseIndex, err)
					return err
				}
			} else {
				prompt, err := session.RenderNextPhasePrompt(scope)
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
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ListQueued returns a list of queued sessions
func (r *Runner[T]) ListQueued() Sessions {
	sessions := make(Sessions)
	for k, v := range r.status {
		if v == queuedSessionStatus {
			sessions[k] = r.sessionManager.Sessions[k]
		}
	}
	return sessions
}

// sendProgress sends a progress message to the progress channel
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
				r.logger.Error("worker failed at running session", "workerID", index, "sessionID", t.id, "err", err)
			}
		}()
	}
}

// SetStatus sets the status of a session (thread-safe)
func (r *Runner[T]) SetStatus(sessionID string, status SessionStatus) {
	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()
	r.status[sessionID] = status
}

// IsCompleted returns true if all sessions are completed
func (r *Runner[T]) IsCompleted() bool {
	for _, s := range r.status {
		if s == queuedSessionStatus {
			return false
		}
	}
	return true
}

// DependencyCheckResult is the result of a dependency check.
type DependencyCheckResult string

const (
	DependencyCheckPassed     DependencyCheckResult = "passed"
	DependencyCheckFailed     DependencyCheckResult = "failed"
	DependencyCheckUnsolvable DependencyCheckResult = "unsolvable"
)

// CheckDependencies checks whether a session can start, cannot start yet, or will never start
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
		scope := map[string]any{
			"params":  r.params,
			"context": *r.dataStructure,
		}
		if dep.Expression != nil {
			c, err := expr.Compile(*dep.Expression, expr.Env(scope))
			if err != nil {
				return DependencyCheckUnsolvable, err
			}
			res, err := expr.Run(c, scope)
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
