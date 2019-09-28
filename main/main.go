package main

import (
	"github.com/iyouport-org/relaybaton"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")
	if err != nil {
		log.Error(err)
		return
	}
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		log.Error(err)
		return
	}
	var conf relaybaton.Config
	if err := v.Unmarshal(&conf); err != nil {
		log.Error(err)
		return
	}
	file, err := os.OpenFile(conf.LogFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err)
	}
	log.SetOutput(file)
	log.SetLevel(log.TraceLevel)
	log.SetFormatter(&log.JSONFormatter{
		PrettyPrint:     true,
		TimestampFormat: "2006-01-02 15:04:05.0000000",
	})
	log.SetReportCaller(true)

	switch os.Args[1] {
	case "client":
		for {
			client, err := relaybaton.NewClient(conf)
			if err != nil {
				log.Error(err)
				continue
			}
			client.Run()
			time.Sleep(5 * time.Second)
		}
	case "server":
		handler := relaybaton.Handler{
			Conf: conf,
		}
		log.Error(http.ListenAndServe(":"+strconv.Itoa(conf.Server.Port), handler))
	}
}
