/*
 * Copyright (C) 2026 Simone Pezzano
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

package log

import (
	"encoding/json"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EventType string

const GenericEventType EventType = "generic"
const StartEventType EventType = "start"
const EndEventType EventType = "end"
const LoadEventType EventType = "load"
const ErrorEventType EventType = "error"
const ResultEventType EventType = "result"

type EventComponent string

const RunnerComponent EventComponent = "runner"
const WorkerComponent EventComponent = "worker"
const FunctionComponent EventComponent = "function"
const TransformerComponent EventComponent = "transformer"
const SessionComponent EventComponent = "session"
const PrePromptComponent EventComponent = "prePrompt"
const PromptComponent EventComponent = "prompt"
const AiComponent EventComponent = "ai"

type ChannelLevel string

const DebugChannelLevel ChannelLevel = "debug"
const InfoChannelLevel ChannelLevel = "info"

type Event struct {
	Level       string         `json:"level"`
	Component   EventComponent `json:"component"`
	ID          string         `json:"id"`
	Type        EventType      `json:"type"`
	Time        time.Time      `json:"time"`
	Message     string         `json:"message,omitempty"`
	Session     *string        `json:"session,omitempty"`
	Resource    *string        `json:"resource,omitempty"`
	Phase       *int           `json:"phase,omitempty"`
	Iteration   *int           `json:"iteration,omitempty"`
	Content     *any           `json:"content,omitempty"`
	Function    *string        `json:"function,omitempty"`
	Transformer *string        `json:"transformer,omitempty"`
	Engine      *string        `json:"engine,omitempty"`
	Err         *EventError    `json:"error,omitempty"`
	Args        map[string]any `json:"args,omitempty"`
}

type EventError struct {
	Message string
}

func (e EventError) Error() string {
	return e.Message
}

func (e EventError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Message)
}

func NewEvent(eType EventType, component EventComponent) Event {
	return Event{
		Component: component,
		Type:      eType,
		Time:      time.Now(),
		ID:        uuid.NewString(),
	}
}

func (e Event) WithMessage(message string) Event {
	e.Message = message
	return e
}

func (e Event) WithSession(session string) Event {
	e.Session = &session
	return e
}

func (e Event) WithResource(resource string) Event {
	e.Resource = &resource
	return e
}

func (e Event) WithPhase(phase int) Event {
	e.Phase = &phase
	return e
}

func (e Event) WithIteration(iteration int) Event {
	e.Iteration = &iteration
	return e
}

func (e Event) WithErr(err error) Event {
	e.Err = &EventError{Message: err.Error()}
	return e
}

func (e Event) WithArgs(args map[string]any) Event {
	e.Args = args
	return e
}

func (e Event) WithFunction(function string) Event {
	e.Function = &function
	return e
}
func (e Event) WithContent(content any) Event {
	e.Content = &content
	return e
}

func (e Event) WithTransformer(transformer string) Event {
	e.Transformer = &transformer
	return e
}

func (e Event) WithEngine(engine string) Event {
	e.Engine = &engine
	return e
}

func (e Event) ToArray() []any {
	result := make([]any, 0)
	v := reflect.ValueOf(e)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := strings.ToLower(field.Name)

		// Skip fields which make no sense in the logging context
		if slices.Contains([]string{"args", "level", "message", "id", "time"}, fieldName) {
			continue
		}
		fieldValue := v.Field(i)
		if (fieldValue.Kind() == reflect.Pointer && !fieldValue.IsNil()) || fieldValue.Kind() != reflect.Pointer {
			var val any
			if fieldValue.Kind() == reflect.Pointer {
				val = fieldValue.Elem().Interface()
			} else {
				val = fieldValue.Interface()
			}
			result = append(result, fieldName, val)
		}
	}
	// Append Args map entries
	for k, val := range e.Args {
		result = append(result, k, val)
	}

	return result
}

func (e Event) SetArgs(args []any) {
	e.Args = make(map[string]any)
	for i := 0; i < len(args); i += 2 {
		e.Args[args[i].(string)] = args[i+1]
	}
}

func (e Event) WithArg(key string, value any) Event {
	if e.Args == nil {
		e.Args = make(map[string]any)
	}
	e.Args[key] = value
	return e
}

type StreamerLogger struct {
	progressChannel chan Event
	logger          *slog.Logger
	channelLevel    ChannelLevel
}

func NewStreamerLogger(logger *slog.Logger, channel chan Event, channelLevel ChannelLevel) *StreamerLogger {
	return &StreamerLogger{
		logger:          logger,
		progressChannel: channel,
		channelLevel:    channelLevel,
	}
}

func (l *StreamerLogger) SetChannel(channel chan Event, level ChannelLevel) {
	l.progressChannel = channel
	l.channelLevel = level
}

func (l *StreamerLogger) Close() {
	if l.progressChannel != nil {
		close(l.progressChannel)
		l.progressChannel = nil
	}
}
func (l *StreamerLogger) Channel() chan Event {
	return l.progressChannel
}

func (l *StreamerLogger) Debug(event Event) {
	event.Level = "debug"
	l.logger.Debug(event.Message, event.ToArray()...)
	if l.channelLevel == DebugChannelLevel || l.channelLevel == InfoChannelLevel {
		l.Send(event)
	}
}

func (l *StreamerLogger) Info(event Event) {
	event.Level = "info"
	l.logger.Info(event.Message, event.ToArray()...)
	l.Send(event)
}

func (l *StreamerLogger) Warn(event Event) {
	event.Level = "warn"
	l.logger.Warn(event.Message, event.ToArray()...)
	l.Send(event)
}

func (l *StreamerLogger) Err(event Event) {
	event.Level = "err"
	l.logger.Error(event.Message, event.ToArray()...)
	l.Send(event)
}

func (l *StreamerLogger) Send(event Event) {
	if l.progressChannel != nil {
		select {
		case l.progressChannel <- event:
		default:
			l.logger.Warn("streamer logger channel full, dropping event")
		}
	}
}
