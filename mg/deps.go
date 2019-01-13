package mg

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

var logger = log.New(os.Stderr, "", 0)

// SerialDeps is like Deps except it runs each dependency serially, instead of
// in parallel. This can be useful for resource intensive dependencies that
// shouldn't be run at the same time.
func SerialDeps(fns ...interface{}) { SerialCtxDeps(context.Background(), fns...) }

// SerialCtxDeps is like CtxDeps except it runs each dependency serially,
// instead of in parallel. This can be useful for resource intensive
// dependencies that shouldn't be run at the same time.
func SerialCtxDeps(ctx context.Context, fns ...interface{}) {
	deps, err := makeDependencies(fns...)
	if err != nil {
		panic(Fatal(1, err.Error()))
	}

	for _, dep := range deps {
		err := dep.RunDependency(ctx)
		if err == nil {
			continue
		}
		exit := ExitStatus(err)
		msg := err.Error()
		if nd, ok := dep.(NamedDependency); ok {
			msg = fmt.Sprintf(`%v: %v`, nd.DependencyName(), msg)
		}
		panic(Fatal(exit, err.Error()))
	}
}

// CtxDeps runs the given functions as dependencies of the calling function.
// Dependencies must only be of type:
//     func()
//     func() error
//     func(context.Context)
//     func(context.Context) error
// Or a similar method on a mg.Namespace type.
//
// The function calling Deps is guaranteed that all dependent functions will be
// run exactly once when Deps returns.  Dependent functions may in turn declare
// their own dependencies using Deps. Each dependency is run in their own
// goroutines. Each function is given the context provided if the function
// prototype allows for it.
func CtxDeps(ctx context.Context, fns ...interface{}) {
	deps, err := makeDependencies(fns...)
	if err != nil {
		panic(Fatal(1, err.Error()))
	}

	errs := make([]error, len(deps))
	var group sync.WaitGroup
	for i, dep := range deps {
		group.Add(1)
		go func(perr *error, dep Dependency) {
			defer group.Done()
			defer recoverPanic(perr)
			err := dep.RunDependency(ctx)
			if err != nil {
				*perr = err
				return
			}
		}(&errs[i], dep)
	}
	group.Wait()

	exit := 0
	msgs := make([]string, 0, len(errs))
	for i, err := range errs {
		if err == nil {
			continue
		}

		exit = changeExit(exit, ExitStatus(err))
		msg := err.Error()
		if nd, ok := deps[i].(NamedDependency); ok {
			msg = fmt.Sprintf(`%v: %v`, nd.DependencyName(), msg)
		}
		msgs = append(msgs, msg)
	}
	if exit > 0 {
		panic(Fatal(exit, strings.Join(msgs, "\n")))
	}
}

func recoverPanic(perr *error) {
	switch err := recover().(type) {
	case nil:
	case error:
		*perr = err
	default:
		*perr = fmt.Errorf("%v", err)
	}
}

// Deps runs the given functions in parallel, exactly once. Dependencies must
// only be of type:
//     func()
//     func() error
//     func(context.Context)
//     func(context.Context) error
// Or a similar method on a mg.Namespace type.
//
// This is a way to build up a tree of dependencies with each dependency
// defining its own dependencies.  Functions must have the same signature as a
// Mage target, i.e. optional context argument, optional error return.
func Deps(fns ...interface{}) { CtxDeps(context.Background(), fns...) }

func changeExit(old, new int) int {
	if new == 0 {
		return old
	}
	if old == 0 {
		return new
	}
	if old == new {
		return old
	}
	// both different and both non-zero, just set
	// exit to 1. Nothing more we can do.
	return 1
}

func name(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func displayName(name string) string {
	splitByPackage := strings.Split(name, ".")
	if len(splitByPackage) == 2 && splitByPackage[0] == "main" {
		return splitByPackage[len(splitByPackage)-1]
	}
	return name
}
