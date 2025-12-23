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

type ProgressAction string

const (
	progressActionStart ProgressAction = "START"
	progressActionEnd   ProgressAction = "END"
	progressActionError ProgressAction = "ERROR"
)

// ProgressMessage is a message sent to the progress channel.
type ProgressMessage struct {
	Action    ProgressAction
	Session   string
	Phase     int
	Iteration int
	Error     error
}

// sendProgress sends a progress message to the progress channel
func (r *Runner[T]) sendProgress(action ProgressAction, sessionID string, phaseIndex int, iteration int, err error) {
	if r.progressChannel != nil {
		r.progressChannel <- ProgressMessage{
			Action:    action,
			Session:   sessionID,
			Phase:     phaseIndex,
			Iteration: iteration,
			Error:     err,
		}
	}
}
