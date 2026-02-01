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
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestStreamerLogger_Info(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, InfoChannelLevel)

	event := NewEvent(GenericEventType, AppComponent).WithMessage("test info")
	streamerLogger.Info(event)

	select {
	case receivedEvent := <-ch:
		if receivedEvent.Message != "test info" {
			t.Errorf("Expected message 'test info', got '%s'", receivedEvent.Message)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for event")
	}
}

func TestStreamerLogger_Debug(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, DebugChannelLevel)

	event := NewEvent(GenericEventType, AppComponent).WithMessage("test debug")
	streamerLogger.Debug(event)

	select {
	case receivedEvent := <-ch:
		if receivedEvent.Message != "test debug" {
			t.Errorf("Expected message 'test debug', got '%s'", receivedEvent.Message)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for event")
	}
}

func TestStreamerLogger_DebugWithInfoLevel(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, InfoChannelLevel)

	event := NewEvent(GenericEventType, AppComponent).WithMessage("test debug with info level")
	streamerLogger.Debug(event)

	select {
	case <-ch:
		t.Error("Received unexpected event on channel for debug message with info level")
	case <-time.After(100 * time.Millisecond):
		// Expected behavior
	}
}

func TestStreamerLogger_Warn(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, InfoChannelLevel)

	event := NewEvent(GenericEventType, AppComponent).WithMessage("test warn")
	streamerLogger.Warn(event)

	select {
	case receivedEvent := <-ch:
		if receivedEvent.Message != "test warn" {
			t.Errorf("Expected message 'test warn', got '%s'", receivedEvent.Message)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for event")
	}
}

func TestStreamerLogger_Err(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, InfoChannelLevel)

	event := NewEvent(ErrorEventType, AppComponent).WithMessage("test error")
	streamerLogger.Err(event)

	select {
	case receivedEvent := <-ch:
		if receivedEvent.Message != "test error" {
			t.Errorf("Expected message 'test error', got '%s'", receivedEvent.Message)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for event")
	}
}

func TestStreamerLogger_FullChannel(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, InfoChannelLevel)

	// Fill the channel
	streamerLogger.Info(NewEvent(GenericEventType, AppComponent).WithMessage("first"))

	// Try to send another event, should not block
	go func() {
		streamerLogger.Info(NewEvent(GenericEventType, AppComponent).WithMessage("second"))
	}()

	select {
	case <-time.After(100 * time.Millisecond):
		// Good, didn't block
	}

	// Make sure only the first event is there
	receivedEvent := <-ch
	if receivedEvent.Message != "first" {
		t.Errorf("Expected message 'first', got '%s'", receivedEvent.Message)
	}

	select {
	case <-ch:
		t.Error("Received unexpected second event on channel")
	case <-time.After(100 * time.Millisecond):
		// Expected behavior
	}
}

func TestStreamerLogger_Close(t *testing.T) {
	ch := make(chan Event, 1)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	streamerLogger := NewStreamerLogger(logger, ch, InfoChannelLevel)

	streamerLogger.Close()

	if streamerLogger.Channel() != nil {
		t.Error("Channel should be nil after Close")
	}

	// Test that sending to a closed channel does not panic
	event := NewEvent(GenericEventType, AppComponent).WithMessage("test after close")
	streamerLogger.Info(event)
}
