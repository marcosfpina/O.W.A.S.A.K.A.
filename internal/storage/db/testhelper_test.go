package db

import (
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/config"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

func testLogger() *logging.Logger {
	cfg := &config.LoggingConfig{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	}
	l, _ := logging.NewLogger(cfg)
	return l
}
