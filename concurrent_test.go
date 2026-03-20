package liquid

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConcurrentRender verifica que el mismo *Template puede renderizarse
// concurrentemente desde múltiples goroutines sin data races ni resultados incorrectos.
// Ejecutar con: go test -race -count=1 ./...
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

// TestConcurrentRenderWithFilters verifica concurrencia con filtros encadenados.
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

// TestConcurrentRenderWithAssign verifica que assign no introduce data races.
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

// TestConcurrentTagForName verifica que TagForName es safe bajo lectura concurrente.
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

// TestConcurrentRegisterTagAndRender verifica que registrar un tag mientras
// se renderiza concurrentemente no causa data races.
func TestConcurrentRegisterTagAndRender(t *testing.T) {
	env := NewEnvironment()

	tmpl, err := ParseWithEnv(`{{ greeting | upcase }}`, env, nil)
	require.NoError(t, err)

	const goroutines = 30
	var wg sync.WaitGroup

	// Goroutines que renderizan
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = tmpl.Render(map[string]interface{}{"greeting": fmt.Sprintf("hello%d", n)}, nil)
		}(i)
	}

	// Goroutines que registran tags (el env no está frozen, simula setup tardío)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = env.RegisterTag(fmt.Sprintf("custom_%d", n), nil)
		}(i)
	}

	wg.Wait()
}
