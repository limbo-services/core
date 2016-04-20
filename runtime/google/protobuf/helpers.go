package protobuf

import (
	"time"

	"limbo.services/core/runtime/limbo"
)

func (t *Timestamp) Validate() error {
	return nil
}

func (t *Timestamp) extract() (sec int64, nsec int32) {
	if t == nil {
		return 0, 0
	}

	x := *((*time.Time)(t))

	if x.IsZero() {
		return 0, 0
	}

	x = x.UTC()
	return x.Unix(), int32(x.Nanosecond())
}

func (t *Timestamp) inject(sec int64, nsec int32) {
	if t == nil {
		return
	}

	if sec == 0 && nsec == 0 {
		*t = Timestamp(time.Time{})
		return
	}

	*t = Timestamp(time.Unix(sec, int64(nsec)))
}

func init() {
	limbo.RegisterSchemaDefinitions([]limbo.SchemaDefinition{
		{
			Name: "google.protobuf.Timestamp",
			Definition: []byte(`
				{
					"type": "string",
					"format": "date-time"
				}
			`),
		},
	})
}
