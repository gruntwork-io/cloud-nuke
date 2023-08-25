package logging

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger *logrus.Logger

func InitLogger() {
	Logger = logrus.New()

	// Set the desired log level (e.g., Debug, Info, Warn, Error, etc.)
	Logger.SetLevel(logrus.InfoLevel)

	// You can also set the log output (e.g., os.Stdout, a file, etc.)
	Logger.SetOutput(os.Stdout)
}
