package limbo

import (
	"errors"
	"fmt"
	"testing"

	//"github.com/fd/featherhead/pkg/api/svcauth"
)

func TestPanic(t *testing.T) {
	h := &handler{}

	err := func() (err error) {
		defer RecoverPanic(&err, h)
		panic("whoops")
		return errors.New("ignored")
	}()

	if err.Error() != "panic: whoops" {
		t.Error("expected handler error to be returned")
	}
	if h.err != nil {
		t.Error("expected handler err to be nil")
	}
	if r, _ := h.recover.(string); r != "whoops" {
		t.Error("expected panic to be `whoops`")
	}
}

func TestError(t *testing.T) {
	h := &handler{}

	err := func() (err error) {
		defer RecoverPanic(&err, h)
		return errors.New("whoops")
	}()

	if err.Error() != "whoops" {
		t.Error("expected handler error to be returned")
	}
	if h.err.Error() != "whoops" {
		t.Error("expected handler err to be  `whoops`")
	}
	if h.recover != nil {
		t.Error("expected panic to be be nil")
	}
}

type handler struct {
	recover interface{}
	err     error
}

func (h *handler) HandlePanic(r interface{}) error {
	h.recover = r
	return fmt.Errorf("panic: %v", r)
}

func (h *handler) HandleError(err error) error {
	h.err = err
	return err
}
