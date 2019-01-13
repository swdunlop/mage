package mg

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// A NamedDependency offers a name that is useful for debugging output.
type NamedDependency interface {
	// DependencyName is a user-recognizable name that is useful for identifying
	// the source of an error.
	DependencyName() string
}

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

// makeDependencies converts each of the provided values to a dependency, if
// needed, using makeDependency.
func makeDependencies(seq ...interface{}) ([]Dependency, error) {
	deps := make([]Dependency, len(seq))
	var err error
	for i, v := range seq {
		deps[i], err = makeDependency(v)
		if err != nil {
			return nil, err
		}
	}
	return deps, nil
}

// makeDependency converts the provided value to a dependency, if needed, or
// returns an error.
func makeDependency(dep interface{}) (Dependency, error) {
	switch dep := dep.(type) {
	case Dependency:
		return dep, nil
	case func(context.Context) error:
		return needTargetDep(name(dep), dep), nil
	}

	// time to get reflective..
	dv := reflect.ValueOf(dep)
	dt := dv.Type()
	if dt.Kind() != reflect.Func {
		return nil, fmt.Errorf(msgInvalidType, dep)
	}

	hasNamespace, hasContext, hasError := false, false, false

	x, i := 0, dt.NumIn()
	if x < i && isNamespace(dt.In(x)) {
		hasNamespace = true
		x++
	}
	if x < i && isContext(dt.In(x)) {
		hasContext = true
		x++
	}
	if x < i {
		return nil, fmt.Errorf(msgInvalidType, dep)
	}

	x, o := 0, dt.NumOut()
	if x < o && isError(dt.Out(x)) {
		hasError = true
		x++
	}
	if x < o {
		return nil, fmt.Errorf(msgInvalidType, dep)
	}

	return needTargetDep(name(dep), func(ctx context.Context) error {
		in := make([]reflect.Value, 0, 2)
		if hasNamespace {
			in = append(in, reflect.Zero(dt.In(0)))
		}
		if hasContext {
			in = append(in, reflect.ValueOf(ctx))
		}
		out := dv.Call(in)
		if !hasError {
			return nil
		}
		if out[0].IsNil() {
			return nil
		}
		return out[0].Interface().(error)
	}), nil
}

func isNamespace(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	if t.NumField() != 0 {
		return false
	}
	return true // TODO
}

func isContext(t reflect.Type) bool {
	return t == reflect.TypeOf((*context.Context)(nil)).Elem()
}

func isError(t reflect.Type) bool {
	return t == reflect.TypeOf((*error)(nil)).Elem()
}

const msgInvalidType = `Invalid type for dependency: %T. Dependencies must ` +
	` be a mage target function or implement mg.Dependency`

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
	run.once.Do(func() {
		if Verbose() {
			logger.Println("Running dependency:", displayName(string(dep)))
		}
		run.err = run.fn(ctx)
	})
	return run.err
}

// needTargetDep ensures we have a targetRun record and returns a targetDep.
func needTargetDep(id string, fn targetFunc) targetDep {
	targetRunCtl.Lock()
	defer targetRunCtl.Unlock()
	dep := targetDep(id)
	_, dup := targetRunMap[dep]
	if !dup {
		targetRunMap[dep] = &targetRun{fn: fn}
	}
	return dep
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
