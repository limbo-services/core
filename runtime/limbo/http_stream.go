package limbo

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func NewServerStream(
	ctx context.Context,
	rw http.ResponseWriter, req *http.Request,
	writeStream, readStream bool, pageSize int,
) (grpc.ServerStream, error) {
	var (
		r   streamReader
		w   streamWriter
		err error
	)

	if writeStream {
		w = newPagedStreamWriter(rw, pageSize)
	} else {
		w = newSingleStreamWriter(rw)
	}

	if readStream {
		r, err = newPagedStreamReader(req)
	} else {
		r, err = newSingleStreamReader(req)
	}
	if err != nil {
		return nil, err
	}

	s := &serverStream{
		ctx:          ctx,
		streamReader: r,
		streamWriter: w,
	}

	return s, nil
}

type serverStream struct {
	ctx context.Context
	streamReader
	streamWriter
}

func (ss *serverStream) Context() context.Context {
	return ss.ctx
}

type streamReader interface {
	RecvMsg(v interface{}) error
}

type streamWriter interface {
	SendHeader(md metadata.MD) error
	SetTrailer(md metadata.MD)
	SendMsg(v interface{}) error
	CloseSend() error
}

type headerWriter struct {
	rw      http.ResponseWriter
	mtx     sync.Mutex
	header  metadata.MD
	trailer metadata.MD
}

func (w *headerWriter) init(rw http.ResponseWriter) {
	w.rw = rw
}

func (w *headerWriter) SendHeader(md metadata.MD) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.header != nil {
		return errors.New("unable to call SendHeader multiple times")
	}

	w.header = md
	return nil
}

func (w *headerWriter) SetTrailer(md metadata.MD) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.trailer = md
}

func (w *headerWriter) close() error {
	if w.header != nil {
		for k, v := range w.header {
			n := "Grpc-Header-" + k
			for _, v := range v {
				w.rw.Header().Add(n, v)
			}
		}
	}

	if w.trailer != nil {
		for k, v := range w.trailer {
			n := "Grpc-Trailer-" + k
			for _, v := range v {
				w.rw.Header().Add(n, v)
			}
		}
	}

	return nil
}

type pagedStreamWriter struct {
	headerWriter

	rw    http.ResponseWriter
	mtx   sync.Mutex
	items []interface{}
	max   int
}

func newPagedStreamWriter(rw http.ResponseWriter, maxItems int) *pagedStreamWriter {
	w := &pagedStreamWriter{rw: rw}
	w.headerWriter.init(rw)

	w.max = maxItems
	if w.max == 0 {
		w.max = 50
	}

	return w
}

func (w *pagedStreamWriter) SendMsg(v interface{}) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.max == 0 {
		w.max = 50
	}

	if len(w.items) >= w.max {
		return io.EOF
	}

	w.items = append(w.items, v)
	return nil
}

func (w *pagedStreamWriter) CloseSend() error {
	err := w.headerWriter.close()
	if err != nil {
		return err
	}

	w.rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.rw.WriteHeader(200)
	return json.NewEncoder(w.rw).Encode(w.items)
}

type singleStreamWriter struct {
	headerWriter

	rw   http.ResponseWriter
	mtx  sync.Mutex
	item interface{}
}

func newSingleStreamWriter(rw http.ResponseWriter) *singleStreamWriter {
	w := &singleStreamWriter{rw: rw}
	w.headerWriter.init(rw)
	return w
}

func (w *singleStreamWriter) SendMsg(v interface{}) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.item != nil {
		return io.EOF
	}

	w.item = v
	return nil
}

func (w *singleStreamWriter) CloseSend() error {
	err := w.headerWriter.close()
	if err != nil {
		return err
	}

	w.rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.rw.WriteHeader(200)
	return json.NewEncoder(w.rw).Encode(w.item)
}

type pagedStreamReader struct {
	req   *http.Request
	items []json.RawMessage

	mtx    sync.Mutex
	unread []json.RawMessage
}

func newPagedStreamReader(req *http.Request) (*pagedStreamReader, error) {
	r := &pagedStreamReader{req: req}

	if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
		err := json.NewDecoder(req.Body).Decode(&r.items)
		if err != nil {
			return nil, err
		}
	}

	r.unread = r.items
	return r, nil
}

func (r *pagedStreamReader) RecvMsg(v interface{}) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if len(r.unread) == 0 {
		return io.EOF
	}

	err := json.Unmarshal(r.unread[0], v)
	if err != nil {
		return err
	}

	r.unread = r.unread[1:]
	return nil
}

type singleStreamReader struct {
	req  *http.Request
	data []byte

	mtx  sync.Mutex
	read bool
}

func newSingleStreamReader(req *http.Request) (*singleStreamReader, error) {
	r := &singleStreamReader{req: req}

	if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		r.data = data
	} else {
		r.read = true
	}

	return r, nil
}

func (r *singleStreamReader) RecvMsg(v interface{}) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.read {
		return io.EOF
	}

	err := json.Unmarshal(r.data, v)
	if err != nil {
		return err
	}

	r.read = true
	return nil
}
