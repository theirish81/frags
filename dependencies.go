package frags

import "slices"

// Dependency defines whether this session can run or should:
// * wait on another Session to complete
// * run at all, based on an Expression
type Dependency struct {
	Session    *string `json:"session" yaml:"session"`
	Expression *string `json:"expression" yaml:"expression"`
}

// Dependencies is a list of Dependencies
type Dependencies []Dependency

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
			dependencyStatus, _ := r.status.Load(*dep.Session)
			if slices.Contains([]SessionStatus{failedSessionStatus, noOpSessionStatus}, dependencyStatus) {
				return DependencyCheckUnsolvable, nil
			}
			if slices.Contains([]SessionStatus{queuedSessionStatus, committedSessionStatus, runningSessionStatus}, dependencyStatus) {
				return DependencyCheckFailed, nil
			}
		}

		if dep.Expression != nil {
			pass, err := EvaluateBooleanExpression(*dep.Expression, r.newEvalScope().WithVars(r.vars))
			if err != nil {
				return DependencyCheckUnsolvable, err
			}
			if !pass {
				return DependencyCheckUnsolvable, nil
			}
		}
	}
	return DependencyCheckPassed, nil
}
