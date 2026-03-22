package liquid

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// same *Template rendered concurrently from many goroutines must be race-free; run with -race
func TestConcurrentRender(t *testing.T) {
	tmpl, err := Parse(`{{ name | upcase }} {% if active %}yes{% endif %}`, nil)
	require.NoError(t, err)

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			out, err := tmpl.Render(map[string]interface{}{
				"name":   fmt.Sprintf("user%d", n),
				"active": n%2 == 0,
			}, nil)
			if err != nil {
				errors <- err
				return
			}
			expected := fmt.Sprintf("USER%d ", n)
			if n%2 == 0 {
				expected += "yes"
			}
			if out != expected {
				errors <- fmt.Errorf("goroutine %d: got %q, want %q", n, out, expected)
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

// concurrent renders with chained filters must be race-free
func TestConcurrentRenderWithFilters(t *testing.T) {
	tmpl, err := Parse(`{{ items | join: ", " | upcase }}`, nil)
	require.NoError(t, err)

	const goroutines = 30
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := map[string]interface{}{
				"items": []interface{}{
					fmt.Sprintf("item%d", n),
					fmt.Sprintf("other%d", n),
				},
			}
			out, err := tmpl.Render(data, nil)
			if err != nil {
				errors <- err
				return
			}
			expected := fmt.Sprintf("ITEM%d, OTHER%d", n, n)
			if out != expected {
				errors <- fmt.Errorf("goroutine %d: got %q, want %q", n, out, expected)
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

// assign must not introduce data races under concurrent renders
func TestConcurrentRenderWithAssign(t *testing.T) {
	tmpl, err := Parse(`{% assign result = name | upcase %}{{ result }}`, nil)
	require.NoError(t, err)

	const goroutines = 30
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("user%d", n)
			out, err := tmpl.Render(map[string]interface{}{"name": name}, nil)
			if err != nil {
				errors <- err
				return
			}
			expected := fmt.Sprintf("USER%d", n)
			if out != expected {
				errors <- fmt.Errorf("goroutine %d: got %q, want %q", n, out, expected)
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

// TagForName must be safe for concurrent reads
func TestConcurrentTagForName(t *testing.T) {
	env := NewEnvironment()

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			factory := env.TagForName("for")
			if factory == nil {
				errors <- fmt.Errorf("TagForName(for) returned nil")
			}
		}()
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

// registering a tag while rendering concurrently must not cause data races
func TestConcurrentRegisterTagAndRender(t *testing.T) {
	env := NewEnvironment()

	tmpl, err := ParseWithEnv(`{{ greeting | upcase }}`, env, nil)
	require.NoError(t, err)

	const goroutines = 30
	var wg sync.WaitGroup

	// rendering goroutines
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = tmpl.Render(map[string]interface{}{"greeting": fmt.Sprintf("hello%d", n)}, nil)
		}(i)
	}

	// tag-registration goroutines — env is not frozen, simulates late setup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = env.RegisterTag(fmt.Sprintf("custom_%d", n), nil)
		}(i)
	}

	wg.Wait()
}

// counter lives in per-render Registers, so concurrent increments must not interfere
func TestConcurrentIncrementIsolation(t *testing.T) {
	tmpl, err := Parse(`{% increment x %}{% increment x %}{% increment x %}`, nil)
	require.NoError(t, err)

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := tmpl.Render(nil, nil)
			if err != nil {
				errors <- err
				return
			}
			if out != "012" {
				errors <- fmt.Errorf("expected \"012\", got %q", out)
			}
		}()
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
