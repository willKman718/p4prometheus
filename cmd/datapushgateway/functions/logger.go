package functions

import "github.com/sirupsen/logrus"

var logger = logrus.New()

func init() {
	logger.Level = logrus.InfoLevel
}

func SetDebugMode(debug bool) {
	if debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
	}
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}
