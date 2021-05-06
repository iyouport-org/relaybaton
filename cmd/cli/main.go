package main

import (
	"C"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/iyouport-org/relaybaton/internal/cmd/relaybaton"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	err := relaybaton.RootCmd.Execute()
	if err != nil {
		log.Error(err)
	}
}

func init() {
	debug.SetMaxThreads(1 << 20)
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1,netdns=go")
	if err != nil {
		log.Fatal(err)
		return
	}
	err = viper.BindPFlag("config", relaybaton.RootCmd.PersistentFlags().Lookup("config"))
	if err != nil {
		log.Error(err)
	}
	log.SetReportCaller(true)
	log.SetLevel(log.TraceLevel)
	gin.SetMode(gin.ReleaseMode)
}
