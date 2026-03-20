package runtime

import "fmt"

type ResourceLimits struct {
	RenderLengthLimit int
	RenderScoreLimit  int
	AssignScoreLimit  int

	renderScore       int
	assignScore       int
	reachedLimit      bool
	lastCaptureLength *int
}

func NewResourceLimits(limits map[string]interface{}) *ResourceLimits {
	rl := &ResourceLimits{}
	if v, ok := limits["render_length_limit"].(int); ok {
		rl.RenderLengthLimit = v
	}
	if v, ok := limits["render_score_limit"].(int); ok {
		rl.RenderScoreLimit = v
	}
	if v, ok := limits["assign_score_limit"].(int); ok {
		rl.AssignScoreLimit = v
	}
	rl.Reset()
	return rl
}

func (rl *ResourceLimits) IncrementRenderScore(amount int) {
	rl.renderScore += amount
	if rl.RenderScoreLimit > 0 && rl.renderScore > rl.RenderScoreLimit {
		rl.RaiseLimitsReached()
	}
}

func (rl *ResourceLimits) IncrementAssignScore(amount int) {
	rl.assignScore += amount
	if rl.AssignScoreLimit > 0 && rl.assignScore > rl.AssignScoreLimit {
		rl.RaiseLimitsReached()
	}
}

func (rl *ResourceLimits) IncrementWriteScore(outputLen int) error {
	if rl.lastCaptureLength != nil {
		captured := outputLen
		increment := captured - *rl.lastCaptureLength
		rl.lastCaptureLength = &captured
		rl.IncrementAssignScore(increment)
	} else if rl.RenderLengthLimit > 0 && outputLen > rl.RenderLengthLimit {
		return rl.RaiseLimitsReached()
	}
	return nil
}

func (rl *ResourceLimits) RaiseLimitsReached() error {
	rl.reachedLimit = true
	return fmt.Errorf("Memory limits exceeded")
}

func (rl *ResourceLimits) Reached() bool {
	return rl.reachedLimit
}

// Fork crea un nuevo ResourceLimits con los mismos límites configurados
// pero con contadores limpios. Usar en cada Render para evitar compartir estado.
func (rl *ResourceLimits) Fork() *ResourceLimits {
	fresh := &ResourceLimits{
		RenderLengthLimit: rl.RenderLengthLimit,
		RenderScoreLimit:  rl.RenderScoreLimit,
		AssignScoreLimit:  rl.AssignScoreLimit,
	}
	fresh.Reset()
	return fresh
}

func (rl *ResourceLimits) Reset() {
	rl.reachedLimit = false
	rl.lastCaptureLength = nil
	rl.renderScore = 0
	rl.assignScore = 0
}

func (rl *ResourceLimits) WithCapture(block func()) {
	oldCaptureLength := rl.lastCaptureLength
	zero := 0
	rl.lastCaptureLength = &zero
	defer func() {
		rl.lastCaptureLength = oldCaptureLength
	}()
	block()
}

func (rl *ResourceLimits) RenderScore() int {
	return rl.renderScore
}

func (rl *ResourceLimits) AssignScore() int {
	return rl.assignScore
}
