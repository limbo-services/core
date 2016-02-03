
timestamp.pb.go: timestamp.proto timestamp.pb.go.diff
	protoc -I $(GOPATH)/src \
	  $(shell pwd)/timestamp.proto \
	  --golimbo_out=:$(GOPATH)/src/
	patch ./timestamp.pb.go ./timestamp.pb.go.diff
