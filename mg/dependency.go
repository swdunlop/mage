package mg

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// A Dependency is a requirement provided to Deps or ContextDeps.  Since
// multiple targets may have the same requirement, dependencies must protect
// themselves from being run repeatedly.  When targets are used as dependencies,
// mg.Deps and mg.ContextDeps will automatically use an internal map of target
// dependencies to prevent repeated runs.
type Dependency interface {
	// RunDependency should run a dependency no more than once.  The easy way to
	// do this is to use sync.Once to guard against repeated attempts to run a
	// dependency.
	RunDependency(ctx context.Context) error
}

// makeDependency converts the provided value to a dependency, if needed, or
// returns an error.
func makeDependency(target interface{}) (Dependency, error) {
	//TODO(swdunlop): we should detect methods and warn the user that this will
	// not do what they think it will do.
	var id string
	var fn func(context.Context) error
	switch target := target.(type) {
	case Dependency:
		return target, nil
	case func(context.Context) error:
		id, fn = name(target), target
	case func(context.Context):
		id, fn = name(target), func(ctx context.Context) error {
			target(ctx)
			return nil
		}
	case func() error:
		id, fn = name(target), func(_ context.Context) error {
			return target()
		}
	case func():
		id, fn = name(target), func(_ context.Context) error {
			target()
			return nil
		}
	default:
		return nil,
			fmt.Errorf(`%T is not a valid Mage target or dependency`, target)
	}
	return targetDep{id, fn}, nil
}

func resolveFuncAddr(v interface{}) uintptr {
	return reflect.ValueOf(v).Pointer()
}

// targetDep wraps a target to produce a Dependency.  This is automatically used
// by mg.Deps and mg.ContextDeps to convert targets into dependencies.
type targetDep struct {
	// id is the name of the target function that was provided to newTargetDep
	id string

	// fn is either the original target that was given to newTargetDependency or
	// a closure that will call that function.
	fn func(ctx context.Context) error
}

// RunDependency implements Dependency using a global map of function addresses
// to sync.Once.  This ensures a target is only run once.
func (dep targetDep) RunDependency(ctx context.Context) error {
	var err error
	dep.runOnce().Do(func() {
		err = dep.fn(ctx)
	})
	return err
}

func (dep targetDep) runOnce() *sync.Once {
	runOnceCtl.Lock()
	defer runOnceCtl.Unlock()
	ro := runOnceMap[dep.id]
	if ro == nil {
		ro = new(sync.Once)
		runOnceMap[dep.id] = ro
	}
	return ro
}

var runOnceCtl sync.Mutex
var runOnceMap = make(map[string]*sync.Once)
