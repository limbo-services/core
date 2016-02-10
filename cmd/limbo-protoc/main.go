package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	svcpanic = flag.Bool("panic", true, "Disable panic generator plugin")
	svcauth  = flag.Bool("auth", true, "Disable auth generator plugin")
	svchttp  = flag.Bool("http", true, "Disable http generator plugin")
)

func main() {
	flag.Parse()

	protoc, err := exec.LookPath("protoc")
	if err != nil {
		log.Fatal(err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	plugins := strings.Join(filterEnabled([]string{
		"grpc",
		"builtintypes",
		"jsonschema",
		"validation",
		enabled("svcpanic", svcpanic, true),
		enabled("svcauth", svcauth, true),
		enabled("svchttp", svchttp, true),
		"sql",
	}), "+")

	protocIncludeDir := path.Join(protoc, "..", "..", "include")
	gopathIncludeDir := path.Join(os.Getenv("GOPATH"), "src")
	protoGlob := path.Join(pwd, "*.proto")

	files, err := filepath.Glob(protoGlob)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		return
	}

	var opts []string
	opts = append(opts, "plugins="+plugins)
	opts = append(opts, "Mgoogle/protobuf/descriptor.proto=github.com/limbo-services/protobuf/protoc-gen-gogo/descriptor")
	opts = append(opts, "Mgoogle/protobuf/timestamp.proto=github.com/limbo-services/core/runtime/google/protobuf")

	var args []string
	args = append(args, "-I", protocIncludeDir)
	args = append(args, "-I", gopathIncludeDir)
	args = append(args, files...)
	args = append(args, "--golimbo_out="+strings.Join(opts, ",")+":"+gopathIncludeDir+"/")

	cmd := exec.Command(protoc, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = pwd
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func enabled(name string, vp *bool, def bool) string {
	v := def
	if vp != nil {
		v = *vp
	}
	if v {
		return name
	}
	return ""
}

func filterEnabled(l []string) []string {
	o := l[:0]
	for _, n := range l {
		if n != "" {
			o = append(o, n)
		}
	}
	return o
}
