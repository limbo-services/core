package main

import (
	"limbo.services/core/generator"
	"limbo.services/protobuf/vanity"
	"limbo.services/protobuf/vanity/command"

	_ "limbo.services/core/plugins/builtintypes"
	_ "limbo.services/core/plugins/jsonschema"
	_ "limbo.services/core/plugins/sql"
	_ "limbo.services/core/plugins/svcauth"
	_ "limbo.services/core/plugins/svchttp"
	_ "limbo.services/core/plugins/validation"
)

func main() {
	req := command.Read()
	files := req.GetProtoFile()

	vanity.ForEachFile(files, vanity.TurnOnMarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOnSizerAll)
	vanity.ForEachFile(files, vanity.TurnOnUnmarshalerAll)
	vanity.ForEachFile(files, generator.MapTimestampToTime)

	resp := command.Generate(req)
	command.Write(resp)
}
