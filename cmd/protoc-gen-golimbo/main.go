package main

import (
	"github.com/limbo-services/core/generator"
	"github.com/limbo-services/protobuf/vanity"
	"github.com/limbo-services/protobuf/vanity/command"

	_ "github.com/limbo-services/core/plugins/builtintypes"
	_ "github.com/limbo-services/core/plugins/jsonschema"
	_ "github.com/limbo-services/core/plugins/sql"
	_ "github.com/limbo-services/core/plugins/svcauth"
	_ "github.com/limbo-services/core/plugins/svchttp"
	_ "github.com/limbo-services/core/plugins/svcpanic"
	_ "github.com/limbo-services/core/plugins/validation"
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
