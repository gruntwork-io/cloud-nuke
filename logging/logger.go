package logging

import "github.com/gruntwork-io/go-commons/logging"

// Logger - Global logger variable
var Logger = logging.GetLogger("cloud-nuke", "")

func InitLogger(name string, version string) {
	Logger = logging.GetLogger(name, version)
}
