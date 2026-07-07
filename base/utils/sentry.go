package utils

import (
	"fmt"
	"maps"

	sentry "github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
)

var sentryEnabled bool

func trySetupSentryLogging() {
	sentryEnabled = false

	dsn := CoreCfg.SentryDSN
	if dsn == "" {
		LogInfo("config for Sentry not loaded")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		AttachStacktrace: true,
	})
	if err != nil {
		LogWarn("err", err, "unable to setup Sentry logging")
		return
	}

	sentryEnabled = true
	log.Info("Sentry error monitoring configured")
}

func tryCaptureSentryException(level log.Level, fields log.Fields, msg any) {
	if !sentryEnabled {
		return
	}

	var sentryLevel sentry.Level

	switch level {
	case log.ErrorLevel:
		sentryLevel = sentry.LevelError
	case log.FatalLevel, log.PanicLevel:
		sentryLevel = sentry.LevelFatal
	default:
		return
	}

	err := findErrorInFieldsOrMsg(fields, msg)

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentryLevel)
		scope.SetContext("log", buildSentryLogContext(level, fields, msg))
		if err != nil {
			sentry.CaptureException(err)
		} else {
			sentry.CaptureMessage(sentryLogMessage(msg))
		}
	})
}

func findErrorInFieldsOrMsg(fields log.Fields, msg any) error {
	for _, key := range []string{"err", "error"} {
		if v, ok := fields[key]; ok {
			if err, ok := v.(error); ok {
				return err
			}
		}
	}

	if err, ok := msg.(error); ok {
		return err
	}

	return nil
}

func buildSentryLogContext(level log.Level, fields log.Fields, msg any) sentry.Context {
	context := sentry.Context{"level": level.String()}

	maps.Copy(context, fields)

	if message := sentryLogMessage(msg); message != "" {
		context["message"] = message
	}

	return context
}

func sentryLogMessage(msg any) string {
	if msg == nil {
		return ""
	}

	switch value := msg.(type) {
	case error:
		return value.Error()
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}
