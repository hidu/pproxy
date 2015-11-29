package serve

// 系统版本
var PproxyVersion string

func init() {
	PproxyVersion = GetVersion()
}
