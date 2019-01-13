package mg

import (
	"context"
	"sync"
)

// NewProcedure defines a new procedure for implementing named dependencies.
// The resulting procedure can then be used to describe dependencies that differ
// by name but share implementation.
func NewProcedure(
	impl func(context.Context, string) error,
) func(name string) Dependency {
	proc := &procedure{
		impl:     impl,
		products: make(map[string]*product),
	}
	return proc.newProduct
}

type procedure struct {
	// impl cannot be changed once the procedure has been created.
	impl func(context.Context, string) error

	// mutex ensures the product table is consistent.
	mutex sync.Mutex

	// products is a map of names to products.
	products map[string]*product
}

// newProduct produces either a previous product of the procedure with the
// provided name, or registers a new product.
func (proc *procedure) newProduct(name string) Dependency {
	proc.mutex.Lock()
	defer proc.mutex.Unlock()
	prod := proc.products[name]
	if prod != nil {
		return prod
	}
	prod = &product{
		impl: proc.impl,
		name: name,
	}
	proc.products[name] = prod
	return prod
}

// A product is something that can be used as a Dependency that is produced by
// a procedure.
type product struct {
	impl func(context.Context, string) error
	name string
	once sync.Once
	err  error
}

// RunDependency implements Dependency by running the implementation once with
// the product name.  The error, if any, returned by when the implementation
// was run will be returned whenever the dependency is run.
func (prod *product) RunDependency(ctx context.Context) error {
	prod.once.Do(func() {
		prod.err = prod.impl(ctx, prod.name)
	})
	return prod.err
}
