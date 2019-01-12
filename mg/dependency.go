package mg

import (
	"context"
	"reflect"
	"sync"
)

type Dependency interface {
	// RunDependency should run a dependency no more than once.  The easy way to do this is to
	// use sync.Once to guard against repeated attempts to run a dependency.  Targets supplied
	// to mg.Deps and mg.ContextDeps are converted to Dependency using FuncDep.
	RunDependency(ctx context.Context) error
}

// newFuncDep creates a Dependency using the provided function using funcDep with the address
// of the function and a wrapper.
func newFuncDep(target interface{}) Dependency {
	var addr uintptr
	var fn func(context.Context) error
	switch run := target.(type) {
	case Dependency:
		return run
	case func(context.Context) error:
		addr, fn = resolveFuncAddr(run), run
	case func(context.Context):
		addr, fn = resolveFuncAddr(run), func(ctx context.Context) error {
			run(ctx)
			return nil
		}
	case func() error:
		addr, fn = resolveFuncAddr(run), func(_ context.Context) error {
			return run()
		}
	case func():
		addr, fn = resolveFuncAddr(run), func(_ context.Context) error {
			run()
			return nil
		}
	}
	return funcDep{addr, fn}
}

func resolveFuncAddr(v interface{}) uintptr {
	return reflect.ValueOf(v).Pointer()
}

// funcDep wraps a function to produce a Dependency.  This is automatically used by
// mg.Deps and mg.ContextDeps to convert targets into dependencies.
type funcDep struct {
	// addr is the address of the function that was given to newFuncDep, and NOT the closure
	// that may have been wrapped around the function.
	addr uintptr

	// fn is either the original function that was given to newFuncDependency or a closure that
	// will call that function.
	fn func(ctx context.Context) error
}

// RunDependency implements Dependency using a global map of function addresses to sync.Once.
// This ensures a dependency function is only run once.
func (dep funcDep) RunDependency(ctx context.Context) error {
	var err error
	dep.runOnce().Do(func() {
		err = dep.fn(ctx)
	})
	return err
}

func (dep funcDep) runOnce() *sync.Once {
	//TODO(swdunlop): calculate address of the function.
	runOnceCtl.Lock()
	defer runOnceCtl.Unlock()
	ro := runOnceMap[dep.addr]
	if ro == nil {
		ro = new(sync.Once)
		runOnceMap[dep.addr] = ro
	}
	return ro
}

var runOnceCtl sync.Mutex
var runOnceMap = make(map[uintptr]*sync.Once)
