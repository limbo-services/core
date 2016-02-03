set -e

PROTOC_INCL="$(which protoc | sed 's|bin/protoc|include|')"

rm -f *.pb.go
rm -f *.swagger.json

protoc \
  -I $PROTOC_INCL \
  -I $GOPATH/src \
  $PWD/*.proto \
  --golimbo_out=plugins=grpc+svchttp:$GOPATH/src/
