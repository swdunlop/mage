package main

import "sync"

// A OnceMap is a map of named sync.Once instances that can be used by
// Dependency implementations to prevent concurrency and repetition.
type OnceMap struct {
	mutex sync.Mutex
	runs  map[string]*onceRun
}

// Do will run a function only once for a given name, returning any error that
// occurred when it was run.
func (om *OnceMap) Do(name string, task func() error) error {
	om.mutex.Lock()
	if om.runs == nil {
		om.runs = make(map[string]*onceRun)
	}
	run := om.runs[name]
	if run == nil {
		run = new(onceRun)
		om.runs[name] = run
	}
	om.mutex.Unlock()
	run.once.Do(func() {
		run.err = task()
	})
	return run.err
}

type onceRun struct {
	once sync.Once
	err  error
}
