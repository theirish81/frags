/*
 * Copyright (C) 2025 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package frags

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

type ExportableRunner interface {
	Transformers() *Transformers
	RunFunction(name string, args map[string]any) (any, error)
	ScriptEngine() ScriptEngine
}

// Runner is a struct that runs a session manager.
type Runner[T any] struct {
	sessionManager  SessionManager
	status          *SafeMap[string, SessionStatus]
	resourceLoader  ResourceLoader
	ai              Ai
	dataStructure   *T
	params          any
	marshalingMutex sync.Mutex
	statusMutex     sync.Mutex
	sessionChan     chan sessionTask
	sessionWorkers  int
	wg              sync.WaitGroup
	running         bool
	logger          *slog.Logger
	progressChannel chan ProgressMessage
	scriptEngine    ScriptEngine
	kFormat         bool
	vars            map[string]any
}

// SessionStatus is the status of a session.
type SessionStatus string

// Session statuses.
const (
	queuedSessionStatus    = SessionStatus("queued")
	committedSessionStatus = SessionStatus("committed")
	runningSessionStatus   = SessionStatus("running")
	finishedSessionStatus  = SessionStatus("finished")
	failedSessionStatus    = SessionStatus("failed")
	noOpSessionStatus      = SessionStatus("noop")
)

// sessionTask is a message to run a session.
type sessionTask struct {
	id      string
	session Session
	timeout time.Duration
}

// RunnerOptions are options for the runner.
type RunnerOptions struct {
	sessionWorkers  int
	logger          *slog.Logger
	progressChannel chan ProgressMessage
	kFormat         bool
	scriptEngine    ScriptEngine
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

func WithUseKFormat(kFormat bool) RunnerOption {
	return func(o *RunnerOptions) {
		o.kFormat = kFormat
	}
}
func WithScriptEngine(scriptEngine ScriptEngine) RunnerOption {
	return func(o *RunnerOptions) {
		o.scriptEngine = scriptEngine
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
	status := NewSafeMap[string, SessionStatus]()
	for k, _ := range sessionManager.Sessions {
		status.Store(k, queuedSessionStatus)
	}
	return Runner[T]{
		sessionManager:  sessionManager,
		status:          status,
		resourceLoader:  resourceLoader,
		ai:              ai,
		marshalingMutex: sync.Mutex{},
		statusMutex:     sync.Mutex{},
		sessionWorkers:  opts.sessionWorkers,
		logger:          opts.logger,
		progressChannel: opts.progressChannel,
		kFormat:         opts.kFormat,
		scriptEngine:    opts.scriptEngine,
		vars:            make(map[string]any),
	}
}

// Run runs the runner against an optional collection fo parameters
func (r *Runner[T]) Run(params any) (*T, error) {
	if r.running {
		return nil, errors.New("this frags instance is running")
	}
	if err := validator.New().Struct(r.sessionManager); err != nil {
		return nil, err
	}
	r.params = params
	if err := r.checkParametersRequirements(); err != nil {
		return nil, err
	}
	r.running = true
	r.sessionChan = make(chan sessionTask)
	defer func() {
		close(r.sessionChan)
	}()
	if r.sessionManager.Vars != nil {
		r.vars = r.sessionManager.Vars
	}
	r.dataStructure = initDataStructure[T]()
	r.sessionManager.initNullSchema()
	if err := r.sessionManager.Schema.Resolve(r.sessionManager.Components); err != nil {
		return r.dataStructure, errors.New("failed to resolve schema")
	}
	if r.sessionManager.SystemPrompt != nil {
		systemPrompt, err := EvaluateTemplate(*r.sessionManager.SystemPrompt, r.newEvalScope())
		if err != nil {
			return nil, err
		}
		r.ai.SetSystemPrompt(systemPrompt)
	}
	for i := 0; i < r.sessionWorkers; i++ {
		r.logger.Debug("starting session worker", "index", i)
		go r.runSessionWorker(i)
	}
	// as long as all sessions have no reached a terminal state, keep scanning sessions
	for !r.IsCompleted() {
		// if the scan fails, we return the error and stop scanning. This will end the program
		if err := r.scanSessions(); err != nil {
			r.logger.Error("failed to scan sessions", "err", err)
			return r.dataStructure, err
		}
	}
	r.running = false
	return r.dataStructure, nil
}
func (r *Runner[T]) checkParametersRequirements() error {
	if r.sessionManager.Parameters == nil || len(r.sessionManager.Parameters.Parameters) == 0 {
		return nil
	}
	return r.sessionManager.Parameters.Validate(r.params)
}

// scanSessions keeps scanning sessions until completion, sending tasks to workers and orchestrating priority and
// concurrency
func (r *Runner[T]) scanSessions() error {
	r.wg = sync.WaitGroup{}
	// listing all the sessions still in queued state
	for k, s := range r.ListQueued() {
		depCheck, err := r.CheckDependencies(s.DependsOn)
		if err != nil {
			return err
		}
		switch depCheck {
		// if the dependency check fails, it means that RIGHT NOW, we cannot start this session, but we may later
		case DependencyCheckFailed:
			continue
		// if the dependency check results as unsolvable, it means that we will never be able to start this session.
		// A dependency is unsolvable in 2 different scenarios
		// * The dependency is a session that has failed or won't run because of its dependencies
		// * The dependency is an expression that fails
		// We mark it as no-op, which is a terminal state for the session, and we move on.
		case DependencyCheckUnsolvable:
			r.SetStatus(k, noOpSessionStatus)
			continue
		}
		// our parallelism has a layered approach. Every run of scanSessions is a layer, and we will wait for the whole
		// layer to complete before moving on to the next layer.
		r.wg.Add(1)
		r.logger.Debug("sending message to workers for session", "session", k)
		timeout := parseDurationOrDefault(s.Timeout, 10*time.Minute)
		r.SetStatus(k, committedSessionStatus)
		// sending the message to the workers. If all workers are busy, we'll hang here for a while, until a worker
		// is free, so we can complete this layer
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
	if session.Vars == nil {
		session.Vars = make(map[string]any)
	}
	// load all the referenced resources
	resources, err := r.loadSessionResources(session)
	if err != nil {
		return err
	}
	// for all the resources that are destined to be loaded into memory, we get them and set them into Session.Vars
	resourceVars := r.resourcesDataToVars(r.filterVarResourcesData(resources))

	// for all the resources that are destined to be loaded into the AI, we remove the others and keep the for later use
	resources = r.filterAiResources(resources)

	sessionSchema, err := r.sessionManager.Schema.GetSession(sessionID)
	if err != nil {
		return err
	}
	if session.Attempts <= 0 {
		session.Attempts = 1
	}
	iterator := make([]any, 1)
	if session.IterateOn != nil {
		if iterator, err = EvaluateArrayExpression(*session.IterateOn, r.newEvalScope().
			WithVars(session.Vars).WithVars(resourceVars)); err != nil {
			return err
		}
	}
	for itIdx, it := range iterator {
		// here we're creating a new instance of the AI for this session, so it has no state.
		ai := r.ai.New()
		localResources := resources
		if session.PrePrompt != nil {
			// a PrePrompt is a special prompt that runs before the first phase of the session, if present. This kind
			// of prompt does not convert to structured data (doesn't have a schema), and its sole purpose is to enrich
			// the context of the session.
			prePrompt, err := session.RenderPrePrompt(r.newEvalScope().WithIterator(it).
				WithVars(session.Vars).WithVars(resourceVars))
			if err != nil {
				r.sendProgress(progressActionError, sessionID, -1, itIdx, err)
				return err
			}
			if prePrompt != nil {
				prePrompt, err := r.contextualizePrompt(ctx, *prePrompt, session)
				if err != nil {
					r.sendProgress(progressActionError, sessionID, -1, itIdx, err)
					return err
				}

				r.sendProgress(progressActionStart, sessionID, -1, itIdx, nil)
				if _, err := ai.Ask(ctx, prePrompt, nil, session.Tools, r, localResources...); err != nil {
					r.sendProgress(progressActionError, sessionID, -1, itIdx, err)
					return err
				}
				localResources = make([]ResourceData, 0)
				r.sendProgress(progressActionEnd, sessionID, -1, itIdx, nil)
			}
		}
		// For each phase...
		for idx, phaseIndex := range sessionSchema.GetPhaseIndexes() {
			// ...we retry the prompt a number of times, depending on the session's attempts.
			err := retry.New(retry.Attempts(uint(session.Attempts)), retry.Delay(time.Second*5), retry.Context(ctx)).Do(func() error {
				r.sendProgress(progressActionStart, sessionID, phaseIndex, itIdx, nil)
				deadline, _ := ctx.Deadline()
				if time.Now().After(deadline) {
					r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, ctx.Err())
					return ctx.Err()
				}
				phaseSchema, err := sessionSchema.GetPhase(phaseIndex)
				if err != nil {
					r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
					return err
				}
				var data []byte
				scope := r.newEvalScope().WithIterator(it).WithVars(session.Vars).WithVars(resourceVars)
				if idx == 0 {
					prompt, err := session.RenderPrompt(scope)
					if err != nil {
						r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
						return err
					}
					// as this is the first prompt, and there was no prePrompt, we may be asked to additional information
					// to the prompt, like the already extracted context, and pre-calls
					if session.PrePrompt == nil {
						prompt, err = r.contextualizePrompt(ctx, prompt, session)
						if err != nil {
							r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
							return err
						}
					}
					data, err = ai.Ask(ctx, prompt, &phaseSchema, ToolDefinitions{}, r, localResources...)
					if err != nil {
						r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
						return err
					}
				} else {
					prompt, err := session.RenderNextPhasePrompt(scope)
					if err != nil {
						r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
						return err
					}
					data, err = ai.Ask(ctx, prompt, &phaseSchema, ToolDefinitions{}, r)
					if err != nil {
						r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
						return err
					}
				}
				if err := r.safeUnmarshalDataStructure(data); err != nil {
					r.sendProgress(progressActionError, sessionID, phaseIndex, itIdx, err)
					return err
				}
				r.sendProgress(progressActionEnd, sessionID, phaseIndex, itIdx, nil)
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ListQueued returns a list of queued sessions
func (r *Runner[T]) ListQueued() Sessions {
	sessions := make(Sessions)
	for k, v := range r.status.Iter() {
		if v == queuedSessionStatus {
			sessions[k] = r.sessionManager.Sessions[k]
		}
	}
	return sessions
}

// loadSessionResources loads resources for a session.
func (r *Runner[T]) loadSessionResources(session Session) ([]ResourceData, error) {
	resources := make([]ResourceData, 0)
	for _, resource := range session.Resources {
		// the resource identifier can be a template, so we evaluate it here
		identifier, err := EvaluateTemplate(resource.Identifier, r.newEvalScope().WithVars(session.Vars))
		if err != nil {
			return resources, err
		}
		resourceData, err := r.resourceLoader.LoadResource(identifier, resource.Params)
		if err != nil {
			return resources, err
		}
		// We set the resource's Var. If none is defined, then AiResourceDestination is used. This will determine
		// whether the resource will end up in memory or in the Ai context
		resourceData.Var = resource.Var
		if resource.In != nil {
			resourceData.In = *resource.In
		} else {
			resourceData.In = AiResourceDestination
		}

		// For each filter that has an OnResource hook for this resource identifier
		for _, t := range r.Transformers().FilterOnResource(resource.Identifier) {
			// whether the transformer will operate on byte content or resource data depends on whether the resource
			// data contains structured content or not.
			var data any = resourceData.ByteContent
			if resourceData.StructuredContent != nil {
				data = *resourceData.StructuredContent
			}
			data, err := t.Transform(data, r)
			if err != nil {
				return resources, err
			}
			if err := resourceData.SetContent(data); err != nil {
				return resources, err
			}
		}
		resources = append(resources, resourceData)
	}
	return resources, nil
}

func (r *Runner[T]) filterAiResources(resources []ResourceData) []ResourceData {
	return lo.Filter(resources, func(res ResourceData, index int) bool {
		return res.In == AiResourceDestination
	})
}

func (r *Runner[T]) filterVarResourcesData(resources []ResourceData) []ResourceData {
	return lo.Filter(resources, func(res ResourceData, index int) bool {
		return res.In == VarsResourceDestination
	})
}

func (r *Runner[T]) resourcesDataToVars(resources []ResourceData) map[string]any {
	res := make(map[string]any)
	for _, resourceData := range resources {
		vx := ""
		if resourceData.Var != nil {
			vx = *resourceData.Var
		} else {
			re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
			vx = re.ReplaceAllString(resourceData.Identifier, "_")
		}
		if resourceData.StructuredContent == nil {
			res[vx] = string(resourceData.ByteContent)
		} else {
			res[vx] = resourceData.StructuredContent
		}
	}
	return res
}

// runSessionWorker runs a session worker.
func (r *Runner[T]) runSessionWorker(index int) {
	for t := range r.sessionChan {
		r.logger.Debug("worker consuming message", "workerID", index, "sessionID", t.id)
		func() {
			success := true
			// we will create a context for each session, so we can cancel it if it takes too long
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, t.timeout)

			// This "defer" is crucial!
			// Other than canceling the context, it also ensures that:
			// 1. if the worker panics, the failure is handled gracefully
			// 2. the status is always set either to finishedSessionStatus or failedSessionStatus.
			// 3. the worker is removed from the wait group.
			defer func() {
				if err := recover(); err != nil {
					r.logger.Error("worker panicked", "workerID", index, "sessionID", t.id, "err", err)
					success = false
				}
				cancel()
				if success {
					r.SetStatus(t.id, finishedSessionStatus)
				} else {
					r.SetStatus(t.id, failedSessionStatus)
				}
				r.logger.Debug("worker finished consuming message", "workerID", index, "sessionID", t.id)
				r.wg.Done()
			}()
			r.SetStatus(t.id, runningSessionStatus)
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
	r.status.Store(sessionID, status)
}

// IsCompleted returns true if all sessions are completed
func (r *Runner[T]) IsCompleted() bool {
	for _, s := range r.status.Iter() {
		if s == queuedSessionStatus {
			return false
		}
	}
	return true
}

func (r *Runner[T]) RunFunction(name string, args map[string]any) (any, error) {
	return r.ai.RunFunction(FunctionCall{Name: name, Args: args}, r)
}

func (r *Runner[T]) Transformers() *Transformers {
	if r.sessionManager.Transformers == nil {
		return &Transformers{}
	}
	return r.sessionManager.Transformers
}

func (r *Runner[T]) ScriptEngine() ScriptEngine {
	if r.scriptEngine == nil {
		r.logger.Warn("no script engine provided, using dummy engine")
		return &DummyScriptEngine{}
	}
	return r.scriptEngine
}
