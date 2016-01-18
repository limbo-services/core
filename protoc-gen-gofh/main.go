package main

import (
	"github.com/gogo/protobuf/vanity"
	"github.com/gogo/protobuf/vanity/command"

	_ "github.com/fd/featherhead/tools/plugins/svcauth"
	_ "github.com/fd/featherhead/tools/plugins/svchttp"
	_ "github.com/fd/featherhead/tools/plugins/svcpanic"
)

func main() {
	req := command.Read()
	files := req.GetProtoFile()

	vanity.ForEachFile(files, vanity.TurnOffGogoImport)

	vanity.ForEachFile(files, vanity.TurnOnMarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOnSizerAll)
	vanity.ForEachFile(files, vanity.TurnOnUnmarshalerAll)

	resp := command.Generate(req)
	command.Write(resp)
}
