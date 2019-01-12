package mg

import (
	"context"
	"errors"
	"testing"
)

// TestResolveFuncAddr must pass for targets to run reliably with mg.Deps and
// mg.ContextDeps.
func TestResolveFuncAddr(t *testing.T) {
	t.Run(`Closure`, func(t *testing.T) {
		var newThunk = func() func() {
			return func() {}
		}
		f, g := newThunk(), newThunk()
		a, b := resolveFuncAddr(f), resolveFuncAddr(g)
		if a == 0 || b == 0 {
			t.Errorf(`expected non-zero address`)
		} else if a != b {
			t.Errorf(`expected same address`)
		}
	})
	t.Run(`Method`, func(t *testing.T) {
		var q, p testStruct
		a, b := resolveFuncAddr(q.method), resolveFuncAddr(p.method)
		if a == 0 || b == 0 {
			t.Errorf(`expected non-zero address`)
		} else if a != b {
			t.Errorf(`expected same address`)
		}
	})
}

// TestNewFuncDep must pass for mg.Deps and mg.ContextDeps to reliably provide
// contexts to targets and detect returned errors.
func TestNewFuncDep(t *testing.T) {
	t.Run(`NoContextNoErr`, func(t *testing.T) {
		d := newFuncDep(t1)
		fd, ok := d.(funcDep)
		if !ok {
			t.Errorf(`expected newFuncDep to produce a funcDep`)
		}
		if fd.addr != resolveFuncAddr(t1) {
			t.Errorf(`expected address of dep to match target`)
		}
		if fd.fn == nil {
			t.Errorf(`expected newFuncDep to specify fn`)
		}
	})
	t.Run(`ContextNoErr`, func(t *testing.T) {
		d := newFuncDep(t2)
		fd, ok := d.(funcDep)
		if !ok {
			t.Errorf(`expected newFuncDep to produce a funcDep`)
		}
		if fd.addr != resolveFuncAddr(t2) {
			t.Errorf(`expected address of dep to match target`)
		}
		if fd.fn == nil {
			t.Errorf(`expected newFuncDep to specify fn`)
		}
	})
	t.Run(`NoContextErr`, func(t *testing.T) {
		d := newFuncDep(t3)
		fd, ok := d.(funcDep)
		if !ok {
			t.Errorf(`expected newFuncDep to produce a funcDep`)
		}
		if fd.addr != resolveFuncAddr(t3) {
			t.Errorf(`expected address of dep to match target`)
		}
		if fd.fn == nil {
			t.Errorf(`expected newFuncDep to specify fn`)
		}
	})
	t.Run(`ContextErr`, func(t *testing.T) {
		d := newFuncDep(t4)
		fd, ok := d.(funcDep)
		if !ok {
			t.Errorf(`expected newFuncDep to produce a funcDep`)
		}
		if fd.addr != resolveFuncAddr(t4) {
			t.Errorf(`expected address of dep to match target`)
		}
		if resolveFuncAddr(fd.fn) != resolveFuncAddr(t4) {
			t.Errorf(`expected newFuncDep to use the target as fn`)
		}
	})
}

func TestFuncDepDependency(t *testing.T) {
	todo := context.TODO()
	errTest := errors.New(`test`)
	t.Run(`NoContextNoErr`, func(t *testing.T) {
		d := newFuncDep(t1)
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
		d := newFuncDep(t2)
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
		d := newFuncDep(t3)
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
		d := newFuncDep(t4)
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

type testStruct struct{}

func (ts *testStruct) method(ctx context.Context) error { return nil }

var (
	t1runs = 0
	t1     = func() { t1runs++ }
)

var (
	t2runs                 = 0
	t2ctx  context.Context = nil
	t2                     = func(ctx context.Context) {
		t2runs++
		t2ctx = ctx
	}
)

var (
	t3err  error = nil
	t3runs       = 0
	t3           = func() error {
		t3runs++
		return t3err
	}
)

var (
	t4err  error           = nil
	t4runs                 = 0
	t4ctx  context.Context = nil
	t4                     = func(ctx context.Context) error {
		t4runs++
		t4ctx = ctx
		return t3err
	}
)
