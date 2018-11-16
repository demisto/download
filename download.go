package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/download/conf"
	"github.com/demisto/download/repo"
	"github.com/demisto/download/web"
)

var (
	confFile = flag.String("conf", "", "Path to configuration file in JSON format")
	logLevel = flag.String("loglevel", "info", "Specify the log level for output (debug/info/warn/error/fatal/panic) - default is info")
	logFile  = flag.String("logfile", "", "The log file location")
	logoutput   *lumberjack.Logger
)

type closer interface {
	Close() error
}

func run(signalCh chan os.Signal) {
	// If we are on DEV, let's use embedded DB. On test and prod we will use MySQL
	r, err := repo.New()
	if err != nil {
		logrus.Fatal(err)
	}
	serviceChannel := make(chan bool)
	var closers []closer
	closers = append(closers, r)
	appC := web.NewContext(r)
	router := web.New(appC, conf.Options.Static)
	go func() {
		router.Serve()
		serviceChannel <- true
	}()
	// Block until one of the signals above is received
	select {
	case <-signalCh:
		logrus.Infoln("Signal received, initializing clean shutdown...")
	case <-serviceChannel:
		logrus.Infoln("A service went down, shutting down...")
	}
	closeChannel := make(chan bool)
	go func() {
		for i := range closers {
			closers[i].Close()
		}
		closeChannel <- true
	}()
	// Block again until another signal is received, a shutdown timeout elapses,
	// or the Command is gracefully closed
	logrus.Infoln("Waiting for clean shutdown...")
	select {
	case <-signalCh:
		logrus.Infoln("Second signal received, initializing hard shutdown")
	case <-time.After(time.Second * 30):
		logrus.Infoln("Time limit reached, initializing hard shutdown")
	case <-closeChannel:
	}
}

func main() {
	flag.Parse()
	conf.Default()
	if *confFile != "" {
		err := conf.Load(*confFile)
		if err != nil {
			logrus.Fatal(err)
		}
	}
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(level)
	if *logFile != "" {
		logoutput = &lumberjack.Logger{}
		logrus.SetOutput(logoutput)
		logoutput.Filename = *logFile
		logoutput.MaxSize = 10
		logoutput.MaxBackups = 3
		logoutput.MaxAge = 0
	}
	conf.LogWriter = logrus.StandardLogger().Writer()
	defer conf.LogWriter.Close()

	// Handle OS signals to gracefully shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	logrus.Infoln("Listening to OS signals")

	run(signalCh)
	logrus.Infoln("Server shutdown completed")
}
