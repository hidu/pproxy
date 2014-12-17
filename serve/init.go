package serve

var PproxyVersion string = ""

func init() {
	PproxyVersion = GetVersion()
}
