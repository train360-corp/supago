package kong

import (
	_ "embed"
)

//go:embed kong.yml
var ConfigFile []byte
