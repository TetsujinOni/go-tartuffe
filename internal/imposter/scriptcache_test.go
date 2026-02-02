package imposter

import (
	"sync"
	"testing"

	"github.com/dop251/goja"
)

func TestScriptCache_GetOrCompile(t *testing.T) {
	cache := NewScriptCache()

	script := "(function() { return 42; })()"

	// First call should compile
	program1, err := cache.GetOrCompile(script)
	if err != nil {
		t.Fatalf("GetOrCompile failed: %v", err)
	}
	if program1 == nil {
		t.Fatal("Expected non-nil program")
	}

	// Second call should return cached program
	program2, err := cache.GetOrCompile(script)
	if err != nil {
		t.Fatalf("Second GetOrCompile failed: %v", err)
	}

	// Should be the same pointer (cached)
	if program1 != program2 {
		t.Error("Expected same cached program")
	}
}

func TestScriptCache_DifferentScripts(t *testing.T) {
	cache := NewScriptCache()

	script1 := "(function() { return 1; })()"
	script2 := "(function() { return 2; })()"

	program1, err := cache.GetOrCompile(script1)
	if err != nil {
		t.Fatalf("GetOrCompile for script1 failed: %v", err)
	}

	program2, err := cache.GetOrCompile(script2)
	if err != nil {
		t.Fatalf("GetOrCompile for script2 failed: %v", err)
	}

	// Should be different programs
	if program1 == program2 {
		t.Error("Expected different programs for different scripts")
	}
}

func TestScriptCache_ExecuteProgram(t *testing.T) {
	cache := NewScriptCache()
	vm := goja.New()

	script := "(function() { return 42; })()"

	program, err := cache.GetOrCompile(script)
	if err != nil {
		t.Fatalf("GetOrCompile failed: %v", err)
	}

	result, err := vm.RunProgram(program)
	if err != nil {
		t.Fatalf("RunProgram failed: %v", err)
	}

	if result.ToInteger() != 42 {
		t.Errorf("Expected 42, got %v", result.Export())
	}
}

func TestScriptCache_InvalidScript(t *testing.T) {
	cache := NewScriptCache()

	invalidScript := "function( { invalid syntax"

	_, err := cache.GetOrCompile(invalidScript)
	if err == nil {
		t.Error("Expected error for invalid script")
	}
}

func TestScriptCache_ConcurrentAccess(t *testing.T) {
	cache := NewScriptCache()

	const goroutines = 10
	const iterations = 100

	scripts := []string{
		"(function() { return 1; })()",
		"(function() { return 2; })()",
		"(function() { return 3; })()",
		"(function() { return 4; })()",
		"(function() { return 5; })()",
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				script := scripts[j%len(scripts)]
				program, err := cache.GetOrCompile(script)
				if err != nil {
					t.Errorf("GetOrCompile failed: %v", err)
					return
				}
				if program == nil {
					t.Error("Got nil program")
					return
				}
			}
		}()
	}

	wg.Wait()
}

func TestScriptCache_ProgramReusableAcrossVMs(t *testing.T) {
	cache := NewScriptCache()

	script := `
		(function() {
			x = x + 1;
			return x;
		})()
	`

	program, err := cache.GetOrCompile(script)
	if err != nil {
		t.Fatalf("GetOrCompile failed: %v", err)
	}

	// Run on first VM
	vm1 := goja.New()
	vm1.Set("x", 10)
	result1, err := vm1.RunProgram(program)
	if err != nil {
		t.Fatalf("RunProgram on vm1 failed: %v", err)
	}
	if result1.ToInteger() != 11 {
		t.Errorf("Expected 11, got %v", result1.Export())
	}

	// Run on second VM with different x
	vm2 := goja.New()
	vm2.Set("x", 100)
	result2, err := vm2.RunProgram(program)
	if err != nil {
		t.Fatalf("RunProgram on vm2 failed: %v", err)
	}
	if result2.ToInteger() != 101 {
		t.Errorf("Expected 101, got %v", result2.Export())
	}
}

func BenchmarkScriptCache_Compile(b *testing.B) {
	script := `
		(function() {
			var fn = function(config) {
				return { statusCode: 200, body: config.path };
			};
			var config = { path: '/test', method: 'GET' };
			return fn(config);
		})()
	`

	b.Run("WithoutCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := goja.Compile("", script, true)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		cache := NewScriptCache()
		// Prime the cache
		cache.GetOrCompile(script)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := cache.GetOrCompile(script)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkScriptCache_Execute(b *testing.B) {
	script := `
		(function() {
			var result = 0;
			for (var i = 0; i < 100; i++) {
				result += i;
			}
			return result;
		})()
	`

	b.Run("RunString", func(b *testing.B) {
		vm := goja.New()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := vm.RunString(script)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RunProgram", func(b *testing.B) {
		cache := NewScriptCache()
		program, _ := cache.GetOrCompile(script)
		vm := goja.New()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := vm.RunProgram(program)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
