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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/go-playground/validator/v10"
	"github.com/theirish81/frags/log"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/util"
)

type ExportableRunner interface {
	Transformers() *Transformers
	RunFunction(ctx *util.FragsContext, name string, args map[string]any) (any, error)
	ScriptEngine() ScriptEngine
	Logger() *log.StreamerLogger
}

// Runner is a struct that runs a session manager.
type Runner[T any] struct {
	sessionManager  SessionManager
	status          *SafeMap[string, SessionStatus]
	resourceLoader  resources.ResourceLoader
	ai              Ai
	dataStructure   *T
	params          any
	marshalingMutex sync.Mutex
	statusMutex     sync.Mutex
	sessionChan     chan sessionTask
	sessionWorkers  int
	wg              sync.WaitGroup
	running         bool
	logger          *log.StreamerLogger
	scriptEngine    ScriptEngine
	kFormat         bool
	vars            Vars
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
	sessionWorkers int
	logger         *log.StreamerLogger
	kFormat        bool
	scriptEngine   ScriptEngine
}

// RunnerOption is an option for the runner.
type RunnerOption func(*RunnerOptions)

// WithLogger sets the logger for the runner.
func WithLogger(logger *log.StreamerLogger) RunnerOption {
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
func NewRunner[T any](sessionManager SessionManager, resourceLoader resources.ResourceLoader, ai Ai, options ...RunnerOption) Runner[T] {
	opts := RunnerOptions{
		sessionWorkers: 1,
		logger: log.NewStreamerLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})), nil, log.DebugChannelLevel),
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
		kFormat:         opts.kFormat,
		scriptEngine:    opts.scriptEngine,
		vars:            make(Vars),
	}
}

// Run runs the runner against an optional collection fo parameters
func (r *Runner[T]) Run(ctx *util.FragsContext, params any) (*T, error) {
	// you cannot invoke Run if an existing Run is in progress
	if r.running {
		return nil, errors.New("this frags instance is running")
	}
	r.running = true
	defer func() {
		r.running = false
	}()

	if err := validator.New().Struct(r.sessionManager); err != nil {
		return nil, err
	}
	r.params = params

	// checking whether the plan has input parameters required, and comparing with the input params
	if err := r.checkParametersRequirements(); err != nil {
		return nil, err
	}

	// initializing the session channel. This is where session tasks will be dispatched
	r.sessionChan = make(chan sessionTask)
	defer func() {
		close(r.sessionChan)
	}()

	// sessionManager vars are copied to the runner
	r.vars.Apply(r.sessionManager.Vars)

	// the dataStructure is instantiated. This is a more complex task than it seems, with generics
	r.dataStructure = util.InitDataStructure[T]()

	// if the sessionManager has no schema, we come up with one based on the sessions.
	r.sessionManager.initNullSchema()

	// we resolve all the $refs
	if err := r.sessionManager.Schema.Resolve(r.sessionManager.Components.Schemas); err != nil {
		return r.dataStructure, errors.New("failed to resolve schema")
	}

	// if the system prompt is available, it evaluates it and set it to the AI
	if r.sessionManager.SystemPrompt != nil {
		systemPrompt, err := EvaluateTemplate(*r.sessionManager.SystemPrompt, r.newEvalScope())
		if err != nil {
			return nil, err
		}
		r.ai.SetSystemPrompt(systemPrompt)
	}

	// start all workers
	for i := 0; i < r.sessionWorkers; i++ {
		r.logger.Debug(log.NewEvent(log.StartEventType, log.WorkerComponent).WithIteration(i))
		go r.runSessionWorker(ctx, i)
	}

	// run all functions with "context" as destination and set the results to the context
	callContextResults, err := r.RunAllFunctionCallers(ctx, r.sessionManager.PreCalls.FilterContextFunctionCalls(), r.newEvalScope())
	if err != nil {
		return r.dataStructure, err
	}
	if err = util.SetAllInContext(r.dataStructure, callContextResults); err != nil {
		return r.dataStructure, err
	}
	// run all functions with "vars" as destination and set the results to the runners vars
	callVarResults, err := r.RunAllFunctionCallers(ctx, r.sessionManager.PreCalls.FilterVarsFunctionCalls(), r.newEvalScope())
	if err != nil {
		return r.dataStructure, err
	}
	r.vars.Apply(callVarResults)

	// as long as all sessions have no reached a terminal state, keep scanning sessions
	for !r.IsCompleted() {
		// if the scan fails, we return the error and stop scanning. This will end the program
		if err := r.scanSessions(ctx); err != nil {
			r.logger.Err(log.NewEvent(log.ErrorEventType, log.RunnerComponent).WithMessage("failed to scan sessions").WithErr(err))
			return r.dataStructure, err
		}
	}
	r.running = false
	err = nil
	if failedSessions := r.ListFailedSessions(); len(failedSessions) > 0 {
		err = errors.New("some sessions failed: " + strings.Join(failedSessions, ","))
	}
	return r.dataStructure, err
}
func (r *Runner[T]) checkParametersRequirements() error {
	if r.sessionManager.Parameters == nil || len(r.sessionManager.Parameters.Parameters) == 0 {
		return nil
	}
	return r.sessionManager.Parameters.Validate(r.params)
}

// scanSessions keeps scanning sessions until completion, sending tasks to workers and orchestrating priority and
// concurrency
func (r *Runner[T]) scanSessions(ctx *util.FragsContext) error {
	r.wg = sync.WaitGroup{}
	// listing all the sessions still in queued state
	for k, s := range r.ListQueued() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
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
		r.logger.Debug(log.NewEvent(log.GenericEventType, log.RunnerComponent).
			WithMessage("sending message to workers for session").WithSession(k))
		timeout := util.ParseDurationOrDefault(s.Timeout, 10*time.Minute)
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
func (r *Runner[T]) runSession(ctx *util.FragsContext, sessionID string, session Session) error {
	if session.Vars == nil {
		session.Vars = make(map[string]any)
	}
	if session.Attempts <= 0 {
		session.Attempts = 1
	}
	// localVars will collect the variables as they get computed in the course of the session.
	localVars := Vars{}
	// first off, the session vars
	localVars.Apply(session.Vars)

	// load all the referenced resources
	sessionResources, err := r.loadSessionResources(ctx, sessionID, session)
	if err != nil {
		return err
	}
	// for all the resources that are destined to be loaded into memory, we get them and set them into localVars
	localVars.Apply(r.resourcesDataToVars(sessionResources.FilterVarResourcesData()))

	// for all the resources that are destined to be loaded into the AI, we remove the others and keep them for later use
	aiResources := sessionResources.FilterAiResources()

	// we initialize the iterator to 1 element in case there's no iterator configured. In this way we make sure we're
	// processing the session at least once.
	iterator := make([]any, 1)

	if session.IterateOn != nil {
		// if there's an iterator, we need to evaluate iterator expression into an array. If it doesn't evaluate to an
		// array, the plan is broken and we return.
		if iterator, err = EvaluateArrayExpression(*session.IterateOn, r.newEvalScope().WithVars(localVars)); err != nil {
			return err
		}
	}

	for itIdx, it := range iterator {
		// here we're creating a new instance of the AI for this session, so it has no state.
		ai := r.ai.New()

		// we take a reference of AI resources. This is useful because we may empty the local collection as we don't
		// want the resources to be loaded into the AI context more than once. For example, if we have a prePrompt,
		// that will load the resources and the prompt will not. If we only have a prompt, then ONLY the first phase
		// will load the resources, and the rest will use them from the AI context.
		localResources := aiResources

		// run all the pre-calls with CONTEXT as destination and set the results to the context
		callContextResults, err := r.RunAllFunctionCallers(ctx, session.PreCalls.FilterContextFunctionCalls(), r.newEvalScope().WithVars(localVars).WithIterator(it))
		if err != nil {
			return err
		}
		if err = util.SetAllInContext(r.dataStructure, callContextResults); err != nil {
			return err
		}

		// run all the pre-calls with VARS as destination and set the results to the local vars
		callVarResults, err := r.RunAllFunctionCallers(ctx, session.PreCalls.FilterVarsFunctionCalls(), r.newEvalScope().WithVars(localVars).WithIterator(it))
		if err != nil {
			return err
		}
		localVars.Apply(callVarResults)

		scope := r.newEvalScope().WithVars(localVars).WithIterator(it)
		if session.HasPrePrompt() {
			// these are the resources that will be loaded into the AI context by the first prePrompt.
			ppResources := append(localResources, sessionResources.FilterPrePromptResources()...)
			// we reset localResources because they've already been introduced in the context by the first prePrompt.
			// The prompt would need to include them only if the prePrompt was not present.
			localResources = make(resources.ResourceDataItems, 0)
			if err := r.runPrePrompts(ctx, ai, sessionID, session, itIdx, scope, ppResources); err != nil {
				return err
			}
		}
		if session.HasPrompt() {
			pResources := append(localResources, sessionResources.FilterPromptResources()...)
			if err := r.runPrompt(ctx, ai, sessionID, session, itIdx, scope, pResources); err != nil {
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

func (r *Runner[T]) runPrePrompts(ctx *util.FragsContext, ai Ai, sessionID string, session Session, iteratorIdx int, scope EvalScope, resources resources.ResourceDataItems) error {
	if !session.HasPrompt() {
		return errors.New("runPrePrompts called on a session without a prompt")
	}
	r.logger.Info(log.NewEvent(log.StartEventType, log.PrePromptComponent).WithSession(sessionID).WithIteration(iteratorIdx))
	// a PrePrompt is a special prompt that runs before the first phase of the session, if present. This kind
	// of prompt does not convert to structured data (doesn't have a schema), and its sole purpose is to enrich
	// the context of the session.
	prePrompts, err := session.RenderPrePrompts(scope)
	if err != nil {
		r.logger.Info(log.NewEvent(log.ErrorEventType, log.PrePromptComponent).
			WithMessage("failed to render pre-prompt").WithErr(err).WithSession(sessionID).WithIteration(iteratorIdx))
		return err
	}
	// the FIRST prePrompt carries all the contextualization payload, so we add whatever needs to go here.
	prePrompt, err := r.contextualizePrompt(ctx, prePrompts[0], session, scope)
	if err != nil {
		r.logger.Err(log.NewEvent(log.ErrorEventType, log.PrePromptComponent).
			WithMessage("failed to contextualize pre-prompt").WithErr(err).WithSession(sessionID).WithIteration(iteratorIdx))
		return err
	}

	// finally we ask the AI for the FIRST prePrompt's response. Notice that the prePrompt is getting the tools.
	if err = util.Retry(ctx, session.Attempts, func() error {
		_, err := ai.Ask(ctx, prePrompt, nil, session.Tools, r, resources...)
		if err != nil {
			r.logger.Err(log.NewEvent(log.ErrorEventType, log.PrePromptComponent).WithMessage("error asking pre-prompt").
				WithSession(sessionID).WithErr(err).WithIteration(iteratorIdx))
		}
		return err
	}); err != nil {
		r.logger.Err(log.NewEvent(log.ErrorEventType, log.PrePromptComponent).WithMessage("error asking pre-prompt").
			WithSession(sessionID).WithErr(err).WithIteration(iteratorIdx))
		return err
	}

	// we run the remaining prePrompts, if any.
	for _, pp := range prePrompts[1:] {
		if err = util.Retry(ctx, session.Attempts, func() error {
			_, err := ai.Ask(ctx, pp, nil, session.Tools, r, resources...)
			if err != nil {
				r.logger.Err(log.NewEvent(log.ErrorEventType, log.PrePromptComponent).
					WithMessage("error asking pre-prompt").WithSession(sessionID).WithErr(err).
					WithIteration(iteratorIdx))
			}
			return err
		}); err != nil {
			r.logger.Err(log.NewEvent(log.ErrorEventType, log.PrePromptComponent).WithMessage("error asking pre-prompt").
				WithSession(sessionID).WithErr(err).WithIteration(iteratorIdx))
			return err
		}
	}
	r.logger.Info(log.NewEvent(log.EndEventType, log.PrePromptComponent).WithSession(sessionID).WithIteration(iteratorIdx))
	return nil
}

func (r *Runner[T]) runPrompt(ctx *util.FragsContext, ai Ai, sessionID string, session Session, iteratorIdx int,
	scope EvalScope, promptResources resources.ResourceDataItems) error {
	if len(session.Prompt) == 0 {
		return errors.New("runPrompt called on a session without a prompt")
	}
	sessionSchema, err := r.sessionManager.Schema.GetSession(sessionID)
	if err != nil {
		return err
	}
	// For each phase...
	for idx, phaseIndex := range sessionSchema.GetPhaseIndexes() {
		// ...we retry the prompt a number of times, depending on the session's attempts.
		err := retry.New(retry.Attempts(uint(session.Attempts)), retry.Delay(time.Second*5), retry.Context(ctx)).Do(func() error {
			if ctx.Err() != nil {
				r.logger.Info(log.NewEvent(log.ErrorEventType, log.PromptComponent).
					WithMessage("context cancelled").WithSession(sessionID).WithPhase(phaseIndex).
					WithIteration(iteratorIdx).WithErr(ctx.Err()))
				return ctx.Err()
			}
			r.logger.Info(log.NewEvent(log.StartEventType, log.PromptComponent).WithSession(sessionID).WithPhase(phaseIndex).
				WithIteration(iteratorIdx))
			phaseSchema, err := sessionSchema.GetPhase(phaseIndex)
			if err != nil {
				r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).
					WithMessage("failed to get phase schema").WithErr(err).WithSession(sessionID).
					WithPhase(phaseIndex).WithIteration(iteratorIdx))
				return err
			}
			var data []byte
			if idx == 0 {
				prompt, err := session.RenderPrompt(scope)
				if err != nil {
					r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).
						WithMessage("failed to render prompt").WithErr(err).WithSession(sessionID).
						WithPhase(phaseIndex).WithIteration(iteratorIdx))
					return err
				}
				// as this is the first phase, and there was no prePrompt, we contextualize the prompt with Frags
				// context or preCalls, if so configured.
				if !session.HasPrePrompt() {
					prompt, err = r.contextualizePrompt(ctx, prompt, session, scope)
					if err != nil {
						r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).
							WithMessage("failed to contextualize prompt").WithErr(err).WithSession(sessionID).
							WithPhase(phaseIndex).WithIteration(iteratorIdx))
						return err
					}
				}
				// finally, we ask the LLM for an answer. Notice we pass NO TOOLS, as only  the prePrompt is allowed
				// to use tools.
				data, err = ai.Ask(ctx, prompt, &phaseSchema, ToolDefinitions{}, r, promptResources...)
				if err != nil {
					r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).WithMessage("error asking prompt").
						WithErr(err).WithSession(sessionID).WithPhase(phaseIndex).WithIteration(iteratorIdx))
					return err
				}
				// we reset localResources because they've already been introduced in the context by this prompt and
				// we don't want subsequent phases to load them again.
				promptResources = make(resources.ResourceDataItems, 0)
			} else {
				// subsequent phases. All context data has been already loaded, we can live peacefully.
				prompt, err := session.RenderNextPhasePrompt(scope)
				if err != nil {
					r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).
						WithMessage("failed to render prompt").WithErr(err).WithSession(sessionID).
						WithPhase(phaseIndex).WithIteration(iteratorIdx))
					return err
				}
				data, err = ai.Ask(ctx, prompt, &phaseSchema, ToolDefinitions{}, r)
				if err != nil {
					r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).WithMessage("error asking prompt").
						WithErr(err).WithSession(sessionID).WithPhase(phaseIndex).WithIteration(iteratorIdx))
					return err
				}
			}
			// regardless of the phase, data is returned and is ideally structured. We can now unmarshal it to
			// the runner data structure.
			if err := r.safeUnmarshalDataStructure(data); err != nil {
				r.logger.Err(log.NewEvent(log.ErrorEventType, log.PromptComponent).WithMessage("failed to unmarshal data").
					WithErr(err).WithSession(sessionID).WithPhase(phaseIndex).WithIteration(iteratorIdx))
				return err
			}
			r.logger.Info(log.NewEvent(log.EndEventType, log.PromptComponent).WithSession(sessionID).WithPhase(phaseIndex).
				WithIteration(iteratorIdx))
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// loadSessionResources loads resources for a session.
func (r *Runner[T]) loadSessionResources(ctx *util.FragsContext, sessionID string, session Session) (resources.ResourceDataItems, error) {
	sessionResources := make(resources.ResourceDataItems, 0)
	for _, resource := range session.Resources {
		if ctx.Err() != nil {
			return sessionResources, ctx.Err()
		}
		// the resource identifier can be a template, so we evaluate it here
		identifier, err := EvaluateTemplate(resource.Identifier, r.newEvalScope().WithVars(session.Vars))
		if err != nil {
			return sessionResources, err
		}
		r.logger.Debug(log.NewEvent(log.LoadEventType, log.RunnerComponent).WithResource(identifier).WithSession(sessionID))
		resourceData, err := r.resourceLoader.LoadResource(identifier, resource.Params)
		if err != nil {
			return sessionResources, err
		}
		// We set the resource's Var. If none is defined, then AiResourceDestination is used. This will determine
		// whether the resource will end up in memory or in the Ai context
		resourceData.Var = resource.Var
		if resource.In != nil {
			resourceData.In = *resource.In
		} else {
			resourceData.In = resources.AiResourceDestination
		}

		// For each filter that has an OnResource hook for this resource identifier
		for _, t := range r.Transformers().FilterOnResource(resource.Identifier) {
			// whether the transformer will operate on byte content or resource data depends on whether the resource
			// data contains structured content or not.
			var data any = resourceData.ByteContent
			if resourceData.StructuredContent != nil {
				data = *resourceData.StructuredContent
			}
			data, err := t.Transform(ctx, data, r)
			if err != nil {
				return sessionResources, err
			}
			if err := resourceData.SetContent(data); err != nil {
				return sessionResources, err
			}
		}
		sessionResources = append(sessionResources, resourceData)
	}
	return sessionResources, nil
}

func (r *Runner[T]) resourcesDataToVars(resources resources.ResourceDataItems) map[string]any {
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
func (r *Runner[T]) runSessionWorker(mainContext *util.FragsContext, index int) {
	for t := range r.sessionChan {
		// if the main context has been cancelled, we discard any message that may be on the channel to drive
		// the runner to its demise as soon as possible
		if mainContext.Err() != nil {
			r.wg.Done()
			continue
		}
		r.logger.Info(log.NewEvent(log.StartEventType, log.SessionComponent).WithSession(t.id))
		func() {
			success := true
			// we will create a context for each session, so we can cancel it if it takes too long
			sessionContext := mainContext.Child(t.timeout)
			// This "defer" is crucial!
			// Other than canceling the context, it also ensures that:
			// 1. if the worker panics, the failure is handled gracefully
			// 2. the status is always set either to finishedSessionStatus or failedSessionStatus.
			// 3. the worker is removed from the wait group.
			defer func() {
				if err := recover(); err != nil {
					r.logger.Err(log.NewEvent(log.ErrorEventType, log.WorkerComponent).WithMessage("worker panicked").
						WithIteration(index).WithSession(t.id))
					success = false
				}
				sessionContext.Cancel()
				if success {
					r.SetStatus(t.id, finishedSessionStatus)
				} else {
					r.SetStatus(t.id, failedSessionStatus)
					mainContext.Cancel()
				}
				r.logger.Info(log.NewEvent(log.EndEventType, log.SessionComponent).WithSession(t.id))
				r.wg.Done()
			}()
			r.SetStatus(t.id, runningSessionStatus)
			if err := r.runSession(sessionContext, t.id, t.session); err != nil {
				success = false
				r.logger.Err(log.NewEvent(log.ErrorEventType, log.WorkerComponent).WithIteration(index).WithSession(t.id).WithErr(err))
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

func (r *Runner[T]) ListFailedSessions() []string {
	failedSessions := make([]string, 0)
	for id, s := range r.status.Iter() {
		if s == failedSessionStatus {
			failedSessions = append(failedSessions, id)
		}
	}
	return failedSessions
}

func (r *Runner[T]) RunFunction(ctx *util.FragsContext, name string, args map[string]any) (any, error) {
	return r.ai.RunFunction(ctx, FunctionCaller{Name: name, Args: args}, r)
}

func (r *Runner[T]) Logger() *log.StreamerLogger {
	return r.logger
}

func (r *Runner[T]) Transformers() *Transformers {
	if r.sessionManager.Transformers == nil {
		return &Transformers{}
	}
	return r.sessionManager.Transformers
}

func (r *Runner[T]) ScriptEngine() ScriptEngine {
	if r.scriptEngine == nil {
		r.logger.Warn(log.NewEvent(log.GenericEventType, log.RunnerComponent).
			WithMessage(fmt.Sprintf("no script engine provided, using dummy engine")))
		return &DummyScriptEngine{}
	}
	return r.scriptEngine
}
