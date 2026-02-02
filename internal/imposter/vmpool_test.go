package imposter

import (
	"sync"
	"testing"
)

func TestVMPool_AcquireRelease(t *testing.T) {
	pool := NewVMPool()

	// Acquire a VM
	vm := pool.Acquire()
	if vm == nil {
		t.Fatal("Expected non-nil VM from Acquire")
	}

	// Set a value and verify it works
	vm.Set("testVar", 42)
	val := vm.Get("testVar")
	if val.ToInteger() != 42 {
		t.Errorf("Expected testVar to be 42, got %v", val.Export())
	}

	// Set a known variable that should be reset
	vm.Set("request", map[string]interface{}{"method": "GET"})

	// Release the VM
	pool.Release(vm)

	// Acquire again
	vm2 := pool.Acquire()
	if vm2 == nil {
		t.Fatal("Expected non-nil VM from second Acquire")
	}

	// The request variable should be undefined after reset (it's a known variable)
	val2 := vm2.Get("request")
	if val2 != nil {
		exported := val2.Export()
		if exported != nil {
			t.Errorf("Expected request to be undefined after reset, got %v", exported)
		}
	}

	pool.Release(vm2)
}

func TestVMPool_ConcurrentAccess(t *testing.T) {
	pool := NewVMPool()

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				vm := pool.Acquire()
				if vm == nil {
					t.Error("Got nil VM")
					return
				}

				// Do some work with the VM
				vm.Set("x", id*iterations+j)
				result, err := vm.RunString("x * 2")
				if err != nil {
					t.Errorf("Script execution failed: %v", err)
					pool.Release(vm)
					return
				}

				expected := int64((id*iterations + j) * 2)
				if result.ToInteger() != expected {
					t.Errorf("Expected %d, got %d", expected, result.ToInteger())
				}

				pool.Release(vm)
			}
		}(i)
	}

	wg.Wait()
}

func TestVMPool_Reset(t *testing.T) {
	pool := NewVMPool()
	vm := pool.Acquire()

	// Set various globals that should be reset
	vm.Set("request", map[string]interface{}{"method": "GET"})
	vm.Set("response", map[string]interface{}{"statusCode": 200})
	vm.Set("state", map[string]interface{}{"counter": 1})
	vm.Set("imposterState", map[string]interface{}{"value": "test"})
	vm.Set("logger", map[string]interface{}{})
	vm.Set("config", map[string]interface{}{})
	vm.Set("requestData", "test data")
	vm.Set("rawBytes", []byte{1, 2, 3})
	vm.Set("data", "some data")

	// Release (which calls resetVM)
	pool.Release(vm)

	// Acquire again
	vm2 := pool.Acquire()

	// All the variables should be undefined
	varsToCheck := []string{"request", "response", "state", "imposterState", "logger", "config", "requestData", "rawBytes", "data"}
	for _, varName := range varsToCheck {
		val := vm2.Get(varName)
		if val != nil {
			exported := val.Export()
			if exported != nil {
				t.Errorf("Expected %s to be undefined after reset, got %v", varName, exported)
			}
		}
	}

	pool.Release(vm2)
}

func TestVMPool_ModulesEnabled(t *testing.T) {
	pool := NewVMPool()
	vm := pool.Acquire()
	defer pool.Release(vm)

	// Test that Buffer is available
	result, err := vm.RunString("Buffer.from('hello').toString()")
	if err != nil {
		t.Fatalf("Buffer should be available: %v", err)
	}
	if result.String() != "hello" {
		t.Errorf("Expected 'hello', got %s", result.String())
	}

	// Test that console is available
	_, err = vm.RunString("console.log('test')")
	if err != nil {
		t.Fatalf("console should be available: %v", err)
	}
}

func BenchmarkVMPool_AcquireRelease(b *testing.B) {
	pool := NewVMPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm := pool.Acquire()
		pool.Release(vm)
	}
}

func BenchmarkVMPool_ConcurrentAcquireRelease(b *testing.B) {
	pool := NewVMPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			vm := pool.Acquire()
			pool.Release(vm)
		}
	})
}
