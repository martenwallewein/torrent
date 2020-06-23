package mem

import (
	"github.com/spf13/afero"
)

var FS = afero.NewOsFs()
var FS2 = afero.NewOsFs()
