package test

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

func MustNewLogger() logr.Logger {
	r, err := zap.NewDevelopment()
	if err != nil {
		panic("Failed to make new logger")
	}
	return zapr.NewLogger(r)
}
