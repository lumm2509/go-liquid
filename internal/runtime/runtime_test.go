package runtime_test

import (
	"testing"

	"github.com/go-liquid/internal/runtime"
)

// --- Registers ---

func TestRegisters_SetGet(t *testing.T) {
	r := runtime.NewRegisters(nil)
	r.Set("key", "value")
	if got := r.Get("key"); got != "value" {
		t.Fatalf("expected 'value', got %v", got)
	}
}

func TestRegisters_StaticFallthrough(t *testing.T) {
	static := map[string]interface{}{"base": 42}
	r := runtime.NewRegisters(static)
	if got := r.Get("base"); got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
}

func TestRegisters_ChangesOverrideStatic(t *testing.T) {
	static := map[string]interface{}{"k": "old"}
	r := runtime.NewRegisters(static)
	r.Set("k", "new")
	if got := r.Get("k"); got != "new" {
		t.Fatalf("expected 'new', got %v", got)
	}
}

func TestRegisters_Delete(t *testing.T) {
	r := runtime.NewRegisters(nil)
	r.Set("x", 1)
	r.Delete("x")
	if got := r.Get("x"); got != nil {
		t.Fatalf("expected nil after delete, got %v", got)
	}
}

func TestRegisters_Key(t *testing.T) {
	static := map[string]interface{}{"a": 1}
	r := runtime.NewRegisters(static)
	if !r.Key("a") {
		t.Fatal("expected Key('a') == true")
	}
	if r.Key("z") {
		t.Fatal("expected Key('z') == false")
	}
}

func TestRegisters_FromRegisters(t *testing.T) {
	parent := runtime.NewRegisters(nil)
	parent.Set("p", "inherited")
	// NewRegisters from *Registers shares static from parent's static (nil initially).
	// Test that static field is accessible via Static() after SetStatic.
	parent.SetStatic("s", "staticval")
	child := runtime.NewRegisters(parent)
	if got := child.Get("s"); got != "staticval" {
		t.Fatalf("child should see parent static, got %v", got)
	}
}

func TestRegisters_FetchDefault(t *testing.T) {
	r := runtime.NewRegisters(nil)
	got := r.Fetch("missing", "default", nil)
	if got != "default" {
		t.Fatalf("expected 'default', got %v", got)
	}
}

func TestRegisters_FetchBlock(t *testing.T) {
	r := runtime.NewRegisters(nil)
	got := r.Fetch("missing", nil, func() interface{} { return "from-block" })
	if got != "from-block" {
		t.Fatalf("expected 'from-block', got %v", got)
	}
}

func TestRegisters_Static(t *testing.T) {
	static := map[string]interface{}{"x": 10}
	r := runtime.NewRegisters(static)
	if r.Static()["x"] != 10 {
		t.Fatal("Static() should return the underlying static map")
	}
}

// --- ResourceLimits ---

func TestResourceLimits_NoLimits(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{})
	rl.IncrementRenderScore(1000)
	rl.IncrementAssignScore(1000)
	if rl.Reached() {
		t.Fatal("unlimited limits should never be reached")
	}
}

func TestResourceLimits_RenderScoreLimit(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{RenderScoreLimit: 5})
	rl.IncrementRenderScore(3)
	if rl.Reached() {
		t.Fatal("not yet over limit")
	}
	rl.IncrementRenderScore(3)
	if !rl.Reached() {
		t.Fatal("should have reached limit at score 6 > 5")
	}
}

func TestResourceLimits_AssignScoreLimit(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{AssignScoreLimit: 10})
	rl.IncrementAssignScore(11)
	if !rl.Reached() {
		t.Fatal("should have reached assign limit")
	}
}

func TestResourceLimits_Reset(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{RenderScoreLimit: 1})
	rl.IncrementRenderScore(5)
	if !rl.Reached() {
		t.Fatal("expected limit reached")
	}
	rl.Reset()
	if rl.Reached() {
		t.Fatal("after Reset, Reached should be false")
	}
	if rl.RenderScore() != 0 {
		t.Fatalf("after Reset, RenderScore should be 0, got %d", rl.RenderScore())
	}
}

func TestResourceLimits_Fork(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{
		RenderScoreLimit: 100,
		AssignScoreLimit: 50,
	})
	rl.IncrementRenderScore(30)

	fork := rl.Fork()
	if fork.RenderScore() != 0 {
		t.Fatalf("fork should start with fresh counters, got %d", fork.RenderScore())
	}
	if fork.RenderScoreLimit != 100 {
		t.Fatalf("fork should inherit limits, got %d", fork.RenderScoreLimit)
	}
}

func TestResourceLimits_WithCapture(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{AssignScoreLimit: 100})
	rl.WithCapture(func() {
		// inside capture, IncrementWriteScore goes to assign score
		_ = rl.IncrementWriteScore(20)
		_ = rl.IncrementWriteScore(35)
	})
	// assign score should have been incremented by 35-0 + 35-20 = 35+15 = 50... actually:
	// First call: captured=20, increment=20-0=20
	// Second call: captured=35, increment=35-20=15
	// Total assign: 35
	if rl.AssignScore() != 35 {
		t.Fatalf("expected assign score 35, got %d", rl.AssignScore())
	}
}

func TestResourceLimits_IncrementRenderScoreReturnsErrorOnLimit(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{RenderScoreLimit: 5})
	if err := rl.IncrementRenderScore(3); err != nil {
		t.Fatalf("expected no error at score 3, got %v", err)
	}
	if err := rl.IncrementRenderScore(3); err == nil {
		t.Fatal("expected error when score exceeds limit, got nil")
	}
}

func TestResourceLimits_IncrementAssignScoreReturnsErrorOnLimit(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{AssignScoreLimit: 10})
	if err := rl.IncrementAssignScore(11); err == nil {
		t.Fatal("expected error when assign score exceeds limit, got nil")
	}
}

func TestResourceLimits_IncrementWriteScorePropagatesAssignError(t *testing.T) {
	rl := runtime.NewResourceLimits(runtime.ResourceLimitsConfig{AssignScoreLimit: 5})
	// Prime the capture mode so IncrementWriteScore routes through IncrementAssignScore
	rl.WithCapture(func() {
		_ = rl.IncrementWriteScore(10) // increment=10 > limit=5, should propagate error
		if !rl.Reached() {
			t.Fatal("expected limits reached inside capture after exceeding assign limit")
		}
	})
}

// --- Interrupts ---

func TestBreakInterrupt(t *testing.T) {
	b := runtime.NewBreakInterrupt("")
	if b.Message() != "interrupt" {
		t.Fatalf("expected 'interrupt', got %q", b.Message())
	}
}

func TestBreakInterrupt_Message(t *testing.T) {
	b := runtime.NewBreakInterrupt("stop")
	if b.Message() != "stop" {
		t.Fatalf("expected 'stop', got %q", b.Message())
	}
}

func TestContinueInterrupt(t *testing.T) {
	c := runtime.NewContinueInterrupt("next")
	if c.Message() != "next" {
		t.Fatalf("expected 'next', got %q", c.Message())
	}
}

func TestInterrupt_Interface(t *testing.T) {
	var i runtime.Interrupt
	i = runtime.NewBreakInterrupt("b")
	if i.Message() != "b" {
		t.Fatal("BreakInterrupt must satisfy Interrupt interface")
	}
	i = runtime.NewContinueInterrupt("c")
	if i.Message() != "c" {
		t.Fatal("ContinueInterrupt must satisfy Interrupt interface")
	}
}
