package main

// void Hello();
import (
	"C"
	_ "github.com/cloudflare/tls-tris"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"relaybaton/internal/cmd/relaybaton"
	"runtime/debug"
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
}

//export Hello
func Hello() {
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

	err = relaybaton.RootCmd.Execute()
	if err != nil {
		log.Error(err)
	}
}
