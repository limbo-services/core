package limbo

import (
	"sort"

	"google.golang.org/grpc"
)

func Wrap(desc *grpc.ServiceDesc) {
	sort.Stable(sortedMiddlewares(middlewares))
	for _, m := range middlewares {
		m.Apply(desc)
	}
}

func RegisterMiddleware(priority int, middleware Middleware) {
	middlewares = append(middlewares, &registeredMiddleware{priority, middleware})
}

var middlewares []*registeredMiddleware

type Middleware interface {
	Apply(desc *grpc.ServiceDesc)
}

type registeredMiddleware struct {
	priority int
	Middleware
}

type sortedMiddlewares []*registeredMiddleware

func (s sortedMiddlewares) Len() int           { return len(s) }
func (s sortedMiddlewares) Less(i, j int) bool { return s[i].priority > s[j].priority }
func (s sortedMiddlewares) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
