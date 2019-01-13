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
func makeDependency(v interface{}) (Dependency, error) {
	//TODO(swdunlop): we should detect methods and warn the user that this will
	// not do what they think it will do.
	var id string
	var fn targetFunc
	switch v := v.(type) {
	case Dependency:
		return v, nil
	case func(context.Context) error:
		id, fn = name(v), v
	case func(context.Context):
		id, fn = name(v), func(ctx context.Context) error {
			v(ctx)
			return nil
		}
	case func() error:
		id, fn = name(v), func(_ context.Context) error {
			return v()
		}
	case func():
		id, fn = name(v), func(_ context.Context) error {
			v()
			return nil
		}
	default:
		return nil,
			fmt.Errorf(`%T is not a valid Mage target or dependency`, v)
	}
	dep := targetDep(id)
	dep.setFunc(fn)
	return dep, nil
}

func resolveFuncAddr(v interface{}) uintptr {
	return reflect.ValueOf(v).Pointer()
}

// targetDep wraps a target to produce a Dependency.  This is automatically used
// by mg.Deps and mg.ContextDeps to convert targets into dependencies.
type targetDep string

// RunDependency implements Dependency using a global map of function addresses
// to sync.Once.  This ensures a target is only run once.
func (dep targetDep) RunDependency(ctx context.Context) error {
	run := dep.getRun()
	run.once.Do(func() { run.err = run.fn(ctx) })
	return run.err
}

func (dep targetDep) setFunc(fn targetFunc) {
	targetRunCtl.Lock()
	defer targetRunCtl.Unlock()
	_, dup := targetRunMap[dep]
	if dup {
		return
	}
	targetRunMap[dep] = &targetRun{fn: fn}
}

func (dep targetDep) getRun() *targetRun {
	targetRunCtl.RLock()
	defer targetRunCtl.RUnlock()
	return targetRunMap[dep]
}

var (
	targetRunCtl sync.RWMutex
	targetRunMap = make(map[targetDep]*targetRun)
)

type targetRun struct {
	once sync.Once
	err  error
	fn   targetFunc
}

type targetFunc func(ctx context.Context) error
