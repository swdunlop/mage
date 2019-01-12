package mg

import (
	"context"
	"errors"
	"testing"
)

// TestMakeDependency must pass for mg.Deps and mg.ContextDeps to reliably
// convert values to dependencies.
func TestMakeDependency(t *testing.T) {
	t.Run(`NoContextNoErr`, func(t *testing.T) {
		d, _ := makeDependency(t1)
		fd, ok := d.(targetDep)
		if !ok {
			t.Errorf(`expected makeDependency to produce a targetDep`)
		}
		if fd.id != name(t1) {
			t.Errorf(`expected address of dep to match target`)
		}
		if fd.fn == nil {
			t.Errorf(`expected makeDependency to specify fn`)
		}
	})
	t.Run(`ContextNoErr`, func(t *testing.T) {
		d, _ := makeDependency(t2)
		fd, ok := d.(targetDep)
		if !ok {
			t.Errorf(`expected makeDependency to produce a targetDep`)
		}
		if fd.id != name(t2) {
			t.Errorf(`expected address of dep to match target`)
		}
		if fd.fn == nil {
			t.Errorf(`expected makeDependency to specify fn`)
		}
	})
	t.Run(`NoContextErr`, func(t *testing.T) {
		d, _ := makeDependency(t3)
		fd, ok := d.(targetDep)
		if !ok {
			t.Errorf(`expected makeDependency to produce a targetDep`)
		}
		if fd.id != name(t3) {
			t.Errorf(`expected address of dep to match target`)
		}
		if fd.fn == nil {
			t.Errorf(`expected makeDependency to specify fn`)
		}
	})
	t.Run(`ContextErr`, func(t *testing.T) {
		d, _ := makeDependency(t4)
		fd, ok := d.(targetDep)
		if !ok {
			t.Errorf(`expected makeDependency to produce a targetDep`)
		}
		if fd.id != name(t4) {
			t.Errorf(`expected address of dep to match target`)
		}
		if name(fd.fn) != name(t4) {
			t.Errorf(`expected makeDependency to use the target as fn`)
		}
	})
}

// TestRunTargetDependency must pass for targets to reliably be run as
// dependencies.
func TestRunTargetDependency(t *testing.T) {
	todo := context.TODO()
	errTest := errors.New(`test`)
	t.Run(`NoContextNoErr`, func(t *testing.T) {
		d, _ := makeDependency(t1)
		t1runs = 0
		err := d.RunDependency(todo)
		if t1runs != 1 {
			t.Error(`expected function to be run`)
			return
		}
		if err != nil {
			t.Errorf(`unexpected error: %v`, err)
		}
		err = d.RunDependency(todo)
		if t1runs != 1 {
			t.Error(`expected function to not run again`)
			return
		}
		if err != nil {
			t.Errorf(`unexpected error on repeat: %v`, err)
		}
	})
	t.Run(`ContextNoErr`, func(t *testing.T) {
		d, _ := makeDependency(t2)
		t2runs = 0
		err := d.RunDependency(todo)
		if t2runs != 1 {
			t.Error(`expected function to be run`)
			return
		}
		if t2ctx != todo {
			t.Error(`expected function to be given context`)
		}
		if err != nil {
			t.Errorf(`unexpected error: %v`, err)
		}
		err = d.RunDependency(todo)
		if t2runs != 1 {
			t.Error(`expected function to not run again`)
			return
		}
		if err != nil {
			t.Errorf(`unexpected error on repeat: %v`, err)
		}
	})
	t.Run(`NoContextErr`, func(t *testing.T) {
		d, _ := makeDependency(t3)
		t3runs, t3err = 0, errTest
		err := d.RunDependency(todo)
		if t3runs != 1 {
			t.Error(`expected function to be run`)
			return
		}
		switch err {
		case nil:
			t.Error(`internal error not relayed`)
		case errTest:
		default:
			t.Errorf(`unexpected error: %v`, err)
		}
		err = d.RunDependency(todo)
		if t3runs != 1 {
			t.Error(`expected function to not run again`)
			return
		}
		if err != nil {
			t.Errorf(`unexpected error on repeat: %v`, err)
		}
	})
	t.Run(`ContextErr`, func(t *testing.T) {
		d, _ := makeDependency(t4)
		t4runs, t4err = 0, errTest
		err := d.RunDependency(todo)
		if t4runs != 1 {
			t.Error(`expected function to be run`)
			return
		}
		if t4ctx != todo {
			t.Error(`expected function to be given context`)
		}
		switch err {
		case nil:
			t.Error(`internal error not relayed`)
		case errTest:
		default:
			t.Errorf(`unexpected error: %v`, err)
		}
		err = d.RunDependency(todo)
		if t4runs != 1 {
			t.Error(`expected function to not run again`)
			return
		}
		if err != nil {
			t.Errorf(`unexpected error on repeat: %v`, err)
		}
	})
}

var (
	t1runs = 0
)

func t1() { t1runs++ }

var (
	t2runs                 = 0
	t2ctx  context.Context = nil
)

func t2(ctx context.Context) {
	t2runs++
	t2ctx = ctx
}

var (
	t3err  error = nil
	t3runs       = 0
)

func t3() error {
	t3runs++
	return t3err
}

var (
	t4err  error           = nil
	t4runs                 = 0
	t4ctx  context.Context = nil
)

func t4(ctx context.Context) error {
	t4runs++
	t4ctx = ctx
	return t3err
}
