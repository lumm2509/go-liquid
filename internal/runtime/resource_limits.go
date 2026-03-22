package runtime

import "fmt"

// ResourceLimitsConfig holds the configured limits for a render.
// Use ResourceLimitsConfig{} (zero value) for unlimited rendering.
type ResourceLimitsConfig struct {
	RenderLengthLimit int
	RenderScoreLimit  int
	AssignScoreLimit  int
}

type ResourceLimits struct {
	RenderLengthLimit int
	RenderScoreLimit  int
	AssignScoreLimit  int

	renderScore       int
	assignScore       int
	reachedLimit      bool
	lastCaptureLength *int
}

func NewResourceLimits(cfg ResourceLimitsConfig) *ResourceLimits {
	rl := &ResourceLimits{
		RenderLengthLimit: cfg.RenderLengthLimit,
		RenderScoreLimit:  cfg.RenderScoreLimit,
		AssignScoreLimit:  cfg.AssignScoreLimit,
	}
	rl.Reset()
	return rl
}

func (rl *ResourceLimits) IncrementRenderScore(amount int) error {
	rl.renderScore += amount
	if rl.RenderScoreLimit > 0 && rl.renderScore > rl.RenderScoreLimit {
		return rl.RaiseLimitsReached()
	}
	return nil
}

func (rl *ResourceLimits) IncrementAssignScore(amount int) error {
	rl.assignScore += amount
	if rl.AssignScoreLimit > 0 && rl.assignScore > rl.AssignScoreLimit {
		return rl.RaiseLimitsReached()
	}
	return nil
}

func (rl *ResourceLimits) IncrementWriteScore(outputLen int) error {
	if rl.lastCaptureLength != nil {
		captured := outputLen
		increment := captured - *rl.lastCaptureLength
		rl.lastCaptureLength = &captured
		if err := rl.IncrementAssignScore(increment); err != nil {
			return err
		}
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
