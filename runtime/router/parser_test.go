package router

import "fmt"

func ExampleParse() {
	tokens, err := parse("//hello/{who?}///{how(\\w){5}}/{+}/")
	if err != nil {
		panic(err)
	}

	for _, token := range tokens {
		fmt.Printf("- %v\n", token)
	}

	// Output:
	// - eps('/')
	// - lit("hello")
	// - eps('/')
	// - var("who", none, 0, 1)
	// - eps('/')
	// - var("how", "(\\w)", 5, 5)
	// - eps('/')
	// - var("1", none, 1, -1)
	// - eps(end)
}
