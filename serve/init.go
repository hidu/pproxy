package serve

import (
	"github.com/hidu/goutils"
)

var PproxyVersion string = ""

func init() {
	utils.ResetDefaultBundle()
	PproxyVersion = GetVersion()
}
