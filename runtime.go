package router

import (
	"net/http"
	"sort"
	"sync"

	"golang.org/x/net/context"
)

var paramPool sync.Pool
var matchPool sync.Pool
var framePool sync.Pool
var runtimePool sync.Pool

func init() {
	paramPool.New = func() interface{} {
		return &paramBuf{make([]param, 0, 1024)}
	}

	matchPool.New = func() interface{} {
		return &matchBuf{make([]match, 0, 128)}
	}

	framePool.New = func() interface{} {
		return &frameBuf{make([]frame, 0, 128)}
	}

	runtimePool.New = func() interface{} {
		return &runtime{}
	}
}

type runtime struct {
	program   []instruction
	frames    []frame
	lastFrame frame
	params    []param
	matches   []match
	instIdx   int

	path   string
	offset int
	length int
	cur    byte
	end    bool
}

type paramBuf struct {
	items []param
}

type frameBuf struct {
	items []frame
}

type matchBuf struct {
	items []match
}

type instruction interface {
	Frames() int
	Jump() *jumpPointer
	Exec(*runtime)
}

type frame struct {
	beg    int
	end    int
	params []param
}

type param struct {
	name  string
	value string
}

type match struct {
	handler
	params []param
}

type jumpPointer struct {
	inst       int
	keepFrames int
}

func runtimeExec(program []instruction, ctx context.Context, rw http.ResponseWriter, req *http.Request) bool {
	var matches = matchPool.Get().(*matchBuf)
	defer matchPool.Put(matches)

	for _, match := range runtimeMatch(req.URL.Path, program, matches.items) {
		cctx := context.WithValue(ctx, paramsKey, match.params)
		if match.ServeHTTP(cctx, rw, req) {
			return true
		}
	}

	return false
}

func runtimeExecTest(path string, program []instruction) {
	var matches = matchPool.Get().(*matchBuf)
	defer matchPool.Put(matches)
	runtimeMatch(path, program, matches.items)
}

func runtimeMatch(path string, program []instruction, matches []match) []match {
	var (
		params = paramPool.Get().(*paramBuf)
		frames = framePool.Get().(*frameBuf)
		r      = runtimePool.Get().(*runtime)
	)

	*r = runtime{
		program: program,
		path:    path,
		length:  len(path),
		params:  params.items,
		frames:  frames.items,
		matches: matches,
	}

	r.SetOffset(0)
	r.Exec()

	sort.Sort((*sortedMatches)(r))

	matches = r.matches
	*r = runtime{}
	paramPool.Put(params)
	framePool.Put(frames)
	runtimePool.Put(r)

	return matches
}

func (c *runtime) Exec() {
	for c.instIdx >= 0 {
		c.Push()
		inst := c.program[c.instIdx]
		c.instIdx++
		inst.Exec(c)

		c.lastFrame.end = c.offset
	}
}

func (c *runtime) Push() {
	// fmt.Printf("PUSH: @%d %d %q@%d\n  %s\n", c.instIdx, len(c.frames)+1, c.path[c.offset:], c.offset, c.program[c.instIdx])

	params := c.params
	if c.lastFrame.params != nil {
		params = c.lastFrame.params
		c.frames = append(c.frames, c.lastFrame)
	}

	c.lastFrame = frame{
		beg:    c.offset,
		end:    c.offset,
		params: params,
	}
}

func (c *runtime) AddParam(name, val string) {
	// fmt.Printf("Added param: %q: %q\n", name, val)
	c.lastFrame.params = append(c.lastFrame.params, param{name, val})
}

func (c *runtime) Commit(hs []handler) {
	for _, h := range hs {
		c.matches = append(c.matches, match{h, c.lastFrame.params})
	}

	paramsLen := len(c.lastFrame.params)
	params2 := append(c.lastFrame.params, c.lastFrame.params...)
	params2 = params2[paramsLen:]

	c.lastFrame.params = params2
	for i, frame := range c.frames {
		frame.params = params2[:len(frame.params)]
		c.frames[i] = frame
	}
}

func (c *runtime) Jump(p jumpPointer) {
	c.instIdx = p.inst

	if p.keepFrames > 0 {
		c.lastFrame = c.frames[p.keepFrames-1]
		c.frames = c.frames[:p.keepFrames-1]
		c.SetOffset(c.lastFrame.end)

		// fmt.Printf("  JUMP -> %d %q %v\n", c.instIdx, c.path[c.offset:], append(c.frames, c.lastFrame))
	} else {
		c.lastFrame = frame{}
		c.frames = c.frames[:0]
		c.SetOffset(0)
	}
}

func (c *runtime) Next() bool {
	return c.SetOffset(c.offset + 1)
}

func (c *runtime) SetOffset(offset int) bool {
	if offset < 0 {
		offset = c.length + offset
	}

	if offset < c.length {
		c.offset = offset
		c.cur = c.path[c.offset]
		c.end = false
		return true
	}

	c.offset = c.length
	c.cur = 0
	c.end = true
	return false
}

type sortedMatches runtime

func (r *sortedMatches) Len() int           { return len(r.matches) }
func (r *sortedMatches) Less(i, j int) bool { return r.matches[i].id < r.matches[j].id }
func (r *sortedMatches) Swap(i, j int)      { r.matches[i], r.matches[j] = r.matches[j], r.matches[i] }
