package frags

const progressActionStart = "START"
const progressActionEnd = "END"
const progressActionError = "ERROR"

// ProgressMessage is a message sent to the progress channel.
type ProgressMessage struct {
	Action  string
	Session string
	Phase   int
	Error   error
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
