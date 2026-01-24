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

package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/theirish81/frags"
)

type Streamer struct {
	c  echo.Context
	l  *frags.StreamerLogger
	mx sync.Mutex
}

func NewStreamer(c echo.Context, logger *frags.StreamerLogger) *Streamer {
	return &Streamer{
		c:  c,
		l:  logger,
		mx: sync.Mutex{},
	}
}

func (s *Streamer) Start() {
	go func() {
		if err := s.streamEvents(s.c, s.l); err != nil {
			s.c.Logger().Error(err)
		}
	}()
}

func (s *Streamer) streamEvents(c echo.Context, streamerLogger *frags.StreamerLogger) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-streamerLogger.Channel():
			if !ok {
				return nil
			}
			data, err := toData(event)
			if err != nil {
				return err
			}
			if err := s.Write(data); err != nil {
				return err
			}
		}
	}
}

func (s *Streamer) Finish(finalEvent frags.Event) error {
	defer func() {
		s.l.Close()
	}()
	for i := 0; i < 10 && len(s.l.Channel()) > 0; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	data, err := toData(finalEvent)
	if err != nil {
		return err
	}
	return s.Write(data)
}

func (s *Streamer) Write(data []byte) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	_, err := s.c.Response().Write(data)
	if err == nil {
		s.c.Response().Flush()
	}
	return err
}

func toData(event frags.Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("id: %s\nevent:%s\ndata:%s\n\n", event.ID, event.Type, string(data))), nil
}
