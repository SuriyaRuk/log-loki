package main

import (
	"errors"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/lokirus"
)

func Sqrt(f float64) (float64, error) {
	if f < 0 {
		return 0, errors.New("math, square root of negative number")
	}
	return math.Sqrt(f), nil
}

func main() {

	server := os.Getenv("SERVER")
	if server == "" {
		server = "0.0.0.0"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server_loki := os.Getenv("SERVER_LOKI")
	if server_loki == "" {
		server_loki = "127.0.0.1"
	}

	port_loki := os.Getenv("PORT_LOKI")
	if port_loki == "" {
		port_loki = "3100"
	}

	job := os.Getenv("JOB")
	if job == "" {
		job = "example-app"
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		password = "secretpassword"
	}

	e := echo.New()

	opts := lokirus.NewLokiHookOptions().
		// Grafana doesn't have a "panic" level, but it does have a "critical" level
		// https://grafana.com/docs/grafana/latest/explore/logs-integration/
		WithLevelMap(lokirus.LevelMap{logrus.PanicLevel: "critical"}).
		WithFormatter(&logrus.JSONFormatter{}).
		WithStaticLabels(lokirus.Labels{
			"job": job,
		}).
		WithBasicAuth(username, password) // Optional

	hook := lokirus.NewLokiHookWithOpts(
		"http://"+server_loki+":"+port_loki,
		opts,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel)

	// Configure the logger
	logger := logrus.New()
	logger.AddHook(hook)

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:           true,
		LogStatus:        true,
		LogMethod:        true,
		LogLatency:       true,
		LogProtocol:      true,
		LogHost:          true,
		LogUserAgent:     true,
		LogRemoteIP:      true,
		LogError:         true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogURIPath:       true,
		LogReferer:       true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			logger.WithFields(logrus.Fields{
				"URI":           values.URI,
				"status":        values.Status,
				"method":        values.Method,
				"startTime":     values.StartTime,
				"latency":       values.Latency,
				"protocol":      values.Protocol,
				"host":          values.Host,
				"userAgent":     values.UserAgent,
				"remoteIP":      values.RemoteIP,
				"erorr":         values.Error,
				"ContentLength": values.ContentLength,
				"responseSize":  values.ResponseSize,
				"uriPath":       values.URIPath,
				"Referer":       values.Referer,
			}).Info("request")

			return nil
		},
	}))

	e.GET("/test", func(c echo.Context) error {
		logger.WithFields(logrus.Fields{"start": "start handler /test " + time.Now().String()}).Info("FROM /test")
		_, error := Sqrt(-1)
		if error != nil {
			logger.WithFields(logrus.Fields{"error": error}).Error("FROM /test")
		}
		return c.String(http.StatusOK, "Hello, Loki!")
	})

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Healthy!")
	})

	e.Logger.Fatal(e.Start(server + ":" + port))
}
