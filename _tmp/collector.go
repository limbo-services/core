package main

import (
	"bufio"
	crand "crypto/rand"
	"log"
	"math/rand"
	"runtime"
)

func main() {
	rnd := bufio.NewReader(crand.Reader)
	for i := range uuidPool {
		rnd.Read(uuidPool[i][:])
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	log.Printf("%d", mem.HeapInuse)

	g := generator()

	for i := 0; i < 20; i++ {
		log.Printf("- %v", <-g)
	}
}

func generator() <-chan Stat {
	out := make(chan Stat)
	go func() {
		defer close(out)

		for {
			flip := uint64(rand.Intn(2))
			out <- Stat{
				UUID:      uuidPool[rand.Intn(N)],
				BytesIn:   uint64(rand.Int63n(4*1024) + 512),
				BytesOut:  uint64(rand.Int63n(512*1024) + 1024),
				ReadReqs:  flip,
				WriteReqs: 1 - flip,
			}
		}
	}()
	return out
}

func collector(in <-chan Stat) {

}

const N = 1 * 1000 * 1000

var uuidPool [N][16]byte

type Stat struct {
	UUID      [16]byte
	BytesIn   uint64
	BytesOut  uint64
	ReadReqs  uint64
	WriteReqs uint64
}
