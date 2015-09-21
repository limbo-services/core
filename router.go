package router

import (
	"net/http"
	"sync"

	"golang.org/x/net/context"
)

// Router dispatches a request to the correct handlers
type Router struct {
	mtx     sync.RWMutex
	routes  []route
	program []instruction
	err     error
}

// Params holds the parameters for the current route
type Params struct {
	params []param
}

type route struct {
	pattern string
	handler Handler
}

type paramsKeyType string

const paramsKey = paramsKeyType("params")

// P returns the Params for the current context
func P(ctx context.Context) Params {
	x, _ := ctx.Value(paramsKey).([]param)
	return Params{x}
}

// Get the first value for the provided key
func (p Params) Get(key string) string {
	for _, p := range p.params {
		if p.name == key {
			return p.value
		}
	}

	return ""
}

// GetAll values for the provided key
func (p Params) GetAll(key string) []string {
	var s = make([]string, 0, 8)

	for _, p := range p.params {
		if p.name == key {
			s = append(s, p.value)
		}
	}

	return s
}

// Add a route to the router
func (r *Router) Add(pattern string, handlers ...Handler) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.program = nil
	r.err = nil

	for _, h := range handlers {
		r.routes = append(r.routes, route{pattern, h})
	}
}

// Compile the router rules
func (r *Router) Compile() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.program != nil {
		return nil
	}

	c := compiler{}
	for _, route := range r.routes {
		err := c.Insert(route.pattern, route.handler)
		if err != nil {
			r.err = err
			return err
		}
	}

	c.Optimize()
	r.program = c.Compile()
	return nil
}

func (r *Router) getProgram() ([]instruction, error) {
	r.mtx.RLock()
	program, err := r.program, r.err
	r.mtx.RUnlock()

	if err != nil {
		return nil, err
	}

	if program != nil {
		return program, nil
	}

	r.Compile()
	return r.getProgram()
}

func (r *Router) ServeHTTP(ctx context.Context, rw http.ResponseWriter, req *http.Request) bool {
	program, err := r.getProgram()
	if err != nil {
		panic(err)
	}

	return runtimeExec(program, ctx, rw, req)
}
