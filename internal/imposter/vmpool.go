package imposter

import (
	"sync"

	"github.com/TetsujinOni/go-tartuffe/internal/metrics"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

// VMPool manages a pool of Goja VMs for an imposter.
// It uses sync.Pool for thread-safe acquisition and release of VMs.
type VMPool struct {
	pool     sync.Pool
	registry *require.Registry
}

// NewVMPool creates a new VM pool.
// The pool will create new VMs as needed and reuse them when released.
func NewVMPool() *VMPool {
	reg := require.NewRegistry()

	vp := &VMPool{
		registry: reg,
	}

	vp.pool.New = vp.createVM

	return vp
}

// createVM creates a new configured Goja VM with all required modules enabled.
func (vp *VMPool) createVM() interface{} {
	metrics.RecordVMCreated()
	vm := goja.New()
	vp.registry.Enable(vm)
	buffer.Enable(vm)
	console.Enable(vm)
	return vm
}

// Acquire gets a VM from the pool. The caller must call Release when done.
func (vp *VMPool) Acquire() *goja.Runtime {
	metrics.RecordVMAcquire()
	return vp.pool.Get().(*goja.Runtime)
}

// Release returns a VM to the pool after resetting it.
// This clears all global variables to prevent state leakage between requests.
func (vp *VMPool) Release(vm *goja.Runtime) {
	vp.resetVM(vm)
	vp.pool.Put(vm)
	metrics.RecordVMRelease()
}

// resetVM clears VM global state between uses.
// This prevents cross-request contamination of request-specific data.
func (vp *VMPool) resetVM(vm *goja.Runtime) {
	// Clear common variables that are set during execution
	vm.Set("request", goja.Undefined())
	vm.Set("response", goja.Undefined())
	vm.Set("state", goja.Undefined())
	vm.Set("imposterState", goja.Undefined())
	vm.Set("logger", goja.Undefined())
	vm.Set("config", goja.Undefined())
	vm.Set("requestData", goja.Undefined())
	vm.Set("rawBytes", goja.Undefined())
	vm.Set("data", goja.Undefined())
}
