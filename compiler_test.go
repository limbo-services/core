package router

import (
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/net/context"
)

func ExampleCompilerCompile() {
	c := &compiler{}

	c.Insert("/_git/blobs/{hash([0-9a-f]{40})}", stringHandler("blob"))
	c.Insert("/admin/{*}", stringHandler("auth-check"))
	c.Insert("/admin/auth", stringHandler("auth"))
	c.Insert("/admin/{*}", stringHandler("admin"))
	c.Insert("/about/{*}", stringHandler("about"))
	c.Insert("/about-us/{*}", stringHandler("about"))
	c.Insert("/about-office/{*}", stringHandler("about"))
	c.Insert("/{*}", stringHandler("public"))
	c.Optimize()
	prog := c.Compile()

	for idx, inst := range prog {
		fmt.Printf("(% 3d) %s\n", idx, inst)
	}

	// Output:
	// (  0) matchEpsilon(frames: 1, onErr: {-1 0})
	// (  1) matchByte('a', frames: 2, onErr: {21 1})
	// (  2) matchBytes("bout", frames: 3, onErr: {15 2})
	// (  3) matchByte('-', frames: 4, onErr: {12 3})
	// (  4) matchBytes("us", frames: 5, onErr: {8 4})
	// (  5) matchEpsilon(frames: 6, onErr: {8 4})
	// (  6) matchVariable(frames: 7, onErr: {8 4})
	// (  7) matchEnd(frames: 8, jump: {8 4})
	// (  8) matchBytes("office", frames: 5, onErr: {12 3})
	// (  9) matchEpsilon(frames: 6, onErr: {12 3})
	// ( 10) matchVariable(frames: 7, onErr: {12 3})
	// ( 11) matchEnd(frames: 8, jump: {12 3})
	// ( 12) matchEpsilon(frames: 4, onErr: {15 2})
	// ( 13) matchVariable(frames: 5, onErr: {15 2})
	// ( 14) matchEnd(frames: 6, jump: {15 2})
	// ( 15) matchBytes("dmin", frames: 3, onErr: {21 1})
	// ( 16) matchEpsilon(frames: 4, onErr: {21 1})
	// ( 17) matchVariable(frames: 5, onErr: {19 4})
	// ( 18) matchEnd(frames: 6, jump: {19 4})
	// ( 19) matchBytes("auth", frames: 5, onErr: {21 1})
	// ( 20) matchEnd(frames: 6, jump: {21 1})
	// ( 21) matchBytes("_git", frames: 2, onErr: {27 1})
	// ( 22) matchEpsilon(frames: 3, onErr: {27 1})
	// ( 23) matchBytes("blobs", frames: 4, onErr: {27 1})
	// ( 24) matchEpsilon(frames: 5, onErr: {27 1})
	// ( 25) matchVariable(frames: 6, onErr: {27 1})
	// ( 26) matchEnd(frames: 7, jump: {27 1})
	// ( 27) matchVariable(frames: 2, onErr: {-1 0})
	// ( 28) matchEnd(frames: 3, jump: {-1 0})
}

func ExampleCompilerOptimize() {
	c := &compiler{}

	c.Insert("/_git/blobs/{hash([0-9a-f]{40})}", stringHandler("blob"))
	c.Insert("/admin/{*}", stringHandler("auth-check"))
	c.Insert("/admin/auth", stringHandler("auth"))
	c.Insert("/admin/{*}", stringHandler("admin"))
	c.Insert("/about/{*}", stringHandler("about"))
	c.Insert("/about-us/{*}", stringHandler("about"))
	c.Insert("/about-office/{*}", stringHandler("about"))
	c.Insert("/{*}", stringHandler("public"))
	c.Optimize()

	fmt.Println(c.root.String())

	// Output:
	// ┬╴eps(28, '/')
	// ├┬╴lit(19, "a")
	// │├┬╴lit(12, "bout")
	// ││├┬╴lit(8, "-")
	// │││├┬╴lit(3, "us")
	// ││││└┬╴eps(2, '/')
	// ││││ └┬╴var(1, [1], none, 0, -1)
	// ││││  └┬╴eps(end)
	// ││││   └─╴handler(5): "about"
	// │││└┬╴lit(3, "office")
	// │││ └┬╴eps(2, '/')
	// │││  └┬╴var(1, [1], none, 0, -1)
	// │││   └┬╴eps(end)
	// │││    └─╴handler(6): "about"
	// ││└┬╴eps(2, '/')
	// ││ └┬╴var(1, [1], none, 0, -1)
	// ││  └┬╴eps(end)
	// ││   └─╴handler(4): "about"
	// │└┬╴lit(5, "dmin")
	// │ └┬╴eps(4, '/')
	// │  ├┬╴var(1, [1], none, 0, -1)
	// │  │└┬╴eps(end)
	// │  │ ├─╴handler(1): "auth-check"
	// │  │ └─╴handler(3): "admin"
	// │  └┬╴lit(1, "auth")
	// │   └┬╴eps(end)
	// │    └─╴handler(2): "auth"
	// ├┬╴lit(5, "_git")
	// │└┬╴eps(4, '/')
	// │ └┬╴lit(3, "blobs")
	// │  └┬╴eps(2, '/')
	// │   └┬╴var(1, [hash], "([0-9a-f]{40})", 1, 1)
	// │    └┬╴eps(end)
	// │     └─╴handler(0): "blob"
	// └┬╴var(1, [1], none, 0, -1)
	//  └┬╴eps(end)
	//   └─╴handler(7): "public"
}

func TestCompiler(t *testing.T) {
	c := &compiler{}

	assert(t, c.Insert("/_git/blobs/{hash([0-9a-f]{40})}", stringHandler("blob")))
	assert(t, c.Insert("/admin/{*}", stringHandler("auth-check")))
	assert(t, c.Insert("/admin/auth", stringHandler("auth")))
	assert(t, c.Insert("/admin/{*}", stringHandler("admin")))
	assert(t, c.Insert("/about/{*}", stringHandler("about")))
	assert(t, c.Insert("/{*}", stringHandler("public")))
}

func BenchmarkCompiler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := &compiler{}
		c.Insert("/_git/blobs/{hash([0-9a-f]{40})}", stringHandler("blob"))
		c.Insert("/admin/{*}", stringHandler("auth-check"))
		c.Insert("/admin/auth", stringHandler("auth"))
		c.Insert("/admin/{*}", stringHandler("admin"))
		c.Insert("/about/{*}", stringHandler("about"))
		c.Insert("/{*}", stringHandler("public"))
		c.Optimize()
	}
}

func assert(t *testing.T, err error) {
	if err != nil {
		panic(err)
	}
}

type stringHandler string

func (h stringHandler) ServeHTTP(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	rw.Write([]byte(string(h) + "\n"))
	return Pass
}
