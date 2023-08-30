package logging

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger = InitLogger()

func InitLogger() *logrus.Logger {
	logger := logrus.New()

	// Set the desired log level (e.g., Debug, Info, Warn, Error, etc.)
	logger.SetLevel(logrus.InfoLevel)

	// You can also set the log output (e.g., os.Stdout, a file, etc.)
	logger.SetOutput(os.Stdout)
	return logger
}
