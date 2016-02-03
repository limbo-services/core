package tests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/limbo-services/core/runtime/limbo"
	"github.com/limbo-services/core/runtime/router"
)

func TestList(t *testing.T) {
	var (
		svc    = testService{}
		router router.Router
	)

	RegisterTestServiceGateway(&router, &svc)

	err := router.Compile()
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := router.ServeHTTP(context.Background(), w, r)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/list")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	data = bytes.TrimSpace(data)

	expected := []byte(`{"items":[{"index":"0"},{"index":"1"},{"index":"2"},{"index":"3"},{"index":"4"},{"index":"5"},{"index":"6"},{"index":"7"},{"index":"8"},{"index":"9"},{"index":"10"},{"index":"11"},{"index":"12"},{"index":"13"},{"index":"14"},{"index":"15"},{"index":"16"},{"index":"17"},{"index":"18"},{"index":"19"},{"index":"20"},{"index":"21"},{"index":"22"},{"index":"23"},{"index":"24"}]}`)

	if !bytes.Equal(data, expected) {
		t.Errorf("expected: %s\nactual:   %s", expected, data)
	}
}
func TestListWithParam(t *testing.T) {
	var (
		svc    = testService{}
		router router.Router
	)

	RegisterTestServiceGateway(&router, &svc)

	err := router.Compile()
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := router.ServeHTTP(context.Background(), w, r)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/list?after=128")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	data = bytes.TrimSpace(data)

	expected := []byte(`{"items":[{"index":"128"},{"index":"129"},{"index":"130"},{"index":"131"},{"index":"132"},{"index":"133"},{"index":"134"},{"index":"135"},{"index":"136"},{"index":"137"},{"index":"138"},{"index":"139"},{"index":"140"},{"index":"141"},{"index":"142"},{"index":"143"},{"index":"144"},{"index":"145"},{"index":"146"},{"index":"147"},{"index":"148"},{"index":"149"},{"index":"150"},{"index":"151"},{"index":"152"}]}`)

	if !bytes.Equal(data, expected) {
		t.Errorf("expected: %s\nactual:   %s", expected, data)
	}
}

func TestGreeting(t *testing.T) {
	var (
		svc    = testService{}
		router router.Router
	)

	RegisterTestServiceGateway(&router, &svc)

	err := router.Compile()
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := router.ServeHTTP(context.Background(), w, r)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/greet", "application/json", strings.NewReader(`{"name":"Simon"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	data = bytes.TrimSpace(data)

	expected := []byte(`{"text":"Hello Simon!"}`)

	if !bytes.Equal(data, expected) {
		t.Errorf("expected: %s\nactual:   %s", expected, data)
	}
}

func TestFetchPerson(t *testing.T) {
	var (
		svc    = testService{}
		router router.Router
	)

	RegisterTestServiceGateway(&router, &svc)

	err := router.Compile()
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := router.ServeHTTP(context.Background(), w, r)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/person/25")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	data = bytes.TrimSpace(data)

	expected := []byte(`{"name":"Person 25"}`)

	if !bytes.Equal(data, expected) {
		t.Errorf("expected: %s\nactual:   %s", expected, data)
	}
}

type testService struct{}

func (s *testService) Greet(ctx context.Context, person *Person) (*Greeting, error) {
	if !limbo.FromGateway(ctx) {
		panic("not from gateway")
	}
	return &Greeting{Text: fmt.Sprintf("Hello %s!", person.Name)}, nil
}

func (s *testService) List(options *ListOptions, stream TestService_ListServer) error {
	if !limbo.FromGateway(stream.Context()) {
		panic("not from gateway")
	}
	for i, l := options.After, options.After+25; i < l; i++ {
		stream.Send(&Item{Index: i})
	}

	return nil
}

func (s *testService) FetchPerson(ctx context.Context, options *FetchOptions) (*Person, error) {
	return &Person{Name: fmt.Sprintf("Person %d", options.Id)}, nil
}
