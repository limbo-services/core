set -e

# /opt/boxen/homebrew/bin/protoc
PROTOC_INCL="$(which protoc | sed 's|bin/protoc|include|')"

rm -f *.pb.go
rm -f *.pb.gw.go
rm -f *.swagger.json
rm -f *.auth.json

protoc \
  -I $PROTOC_INCL \
  -I $GOPATH/src \
  $PWD/*.proto \
  --gofh_out=plugins=grpc+svchttp:$GOPATH/src/
