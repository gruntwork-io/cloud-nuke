package logging

import (
	"fmt"
	"github.com/pterm/pterm"
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

// ParseLogLevel parses the log level from the CLI and sets the log level
func ParseLogLevel(logLevel string) error {
	parsedLogLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level - %s - %s", logLevel, err)
	}

	Logger.SetLevel(parsedLogLevel)
	if parsedLogLevel == logrus.DebugLevel {
		pterm.EnableDebugMessages()
		Logger.Debugf("Setting log level to %s", parsedLogLevel.String())
	}

	return nil
}

func Debug(msg string) {
	pterm.Debug.Println(msg)
}

func Debugf(msg string, args ...interface{}) {
	Debug(fmt.Sprintf(msg, args...))
}

func Info(msg string) {
	pterm.Info.Println(msg)
}

func Infof(msg string, args ...interface{}) {
	Info(fmt.Sprintf(msg, args...))
}

func Error(msg string) {
	pterm.Error.Println(msg)
}

func Errorf(msg string, args ...interface{}) {
	Error(fmt.Sprintf(msg, args...))
}

func Warn(msg string) {
	pterm.Warning.Println(msg)
}

func Warnf(msg string, args ...interface{}) {
	Warn(fmt.Sprintf(msg, args...))
}
