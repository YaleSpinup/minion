package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/YaleSpinup/minion/api"
	"github.com/YaleSpinup/minion/common"

	log "github.com/sirupsen/logrus"
)

const APINAME = "Minion"

var (
	// Version is the main version number
	Version = "0.0.0"

	// Buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	Buildstamp = "No BuildStamp Provided"

	// Githash is the git sha of the built binary, it should be set at buildtime with ldflags
	Githash = "No Git Commit Provided"

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("unable to get working directory")
	}
	log.Infof("Starting %s version %s (%s)", APINAME, Version, cwd)

	config, err := common.ReadConfig(configReader())
	if err != nil {
		log.Fatalf("Unable to read configuration from: %+v", err)
	}

	config.Version = common.Version{
		Version:    Version,
		BuildStamp: Buildstamp,
		GitHash:    Githash,
	}

	// Set the loglevel, info if it's unset
	switch config.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	if config.LogLevel == "debug" {
		log.Debug("Starting profiler on 127.0.0.1:6080")
		go http.ListenAndServe("127.0.0.1:6080", nil)
	}
	log.Debugf("Read config: %+v", config)

	if err := api.NewServer(config); err != nil {
		log.Fatal(err)
	}
}

func configReader() io.Reader {
	if configEnv := os.Getenv("API_CONFIG"); configEnv != "" {
		log.Infof("reading configuration from API_CONFIG environment")

		c, err := base64.StdEncoding.DecodeString(configEnv)
		if err != nil {
			log.Infof("API_CONFIG is not base64 encoded")
			c = []byte(configEnv)
		}

		return bytes.NewReader(c)
	}

	log.Infof("reading configuration from %s", *configFileName)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("unable to open config file", err)
	}

	c, err := ioutil.ReadAll(configFile)
	if err != nil {
		log.Fatalln("unable to read config file", err)
	}

	return bytes.NewReader(c)
}

func vers() {
	fmt.Printf("%s Version: %s\n", APINAME, Version)
	os.Exit(0)
}
