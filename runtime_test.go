package router

import (
	"fmt"
	"testing"
)

func ExampleRuntime() {
	c := &compiler{}

	c.Insert("/_git/blobs/{hash([0-9a-f]{40})}", stringHandler("blob"))
	c.Insert("/admin/{*}", stringHandler("auth-check"))
	c.Insert("/admin/auth", stringHandler("auth"))
	c.Insert("/admin/{*}", stringHandler("admin"))
	c.Insert("/about/{*}", stringHandler("about"))
	c.Insert("/{prepass(about-.*)}/{*}", stringHandler("about-prepass"))
	c.Insert("/about-us/{*}", stringHandler("about"))
	c.Insert("/about-office/{*}", stringHandler("about"))
	c.Insert("/{*}", stringHandler("public"))
	c.Optimize()
	prog := c.Compile()

	var urls = []string{
		"/admin//auth",
		"/about-office",
		"/about-office/test",
	}

	for _, url := range urls {
		r := newRuntime(url, prog)
		runtimeMatch(r)

		fmt.Printf("match %q:\n", url)
		for _, m := range r.matches {
			fmt.Printf("- %v\n", m)
		}

		r.free()
	}

	// Output:
	// match "/admin//auth":
	// - {{1 auth-check} [{1 auth}]}
	// - {{2 auth} []}
	// - {{3 admin} [{1 auth}]}
	// - {{8 public} [{1 admin} {1 auth}]}
	// match "/about-office":
	// - {{5 about-prepass} [{prepass about-office}]}
	// - {{7 about} []}
	// - {{8 public} [{1 about-office}]}
	// match "/about-office/test":
	// - {{5 about-prepass} [{prepass about-office} {1 test}]}
	// - {{7 about} [{1 test}]}
	// - {{8 public} [{1 about-office} {1 test}]}
}

func BenchmarkRuntime(b *testing.B) {
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

	path := "/admin//auth"
	runtimeExecTest(path, prog)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		runtimeExecTest(path, prog)
	}
}
