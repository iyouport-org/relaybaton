package relaybaton

import (
	"context"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"relaybaton-dev/pkg/config"
	"relaybaton-dev/pkg/core"
)

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "TODO",
	Long:  "TODO",
	Run:   clientExec,
}

func clientExec(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var router *core.Router
	app := fx.New(
		fx.Provide(
			core.NewRouter,
			config.NewConfClient,
			goroutine.Default,
		),
		fx.Logger(log.StandardLogger()),
		fx.Populate(&router),
		fx.Invoke(config.InitLog, config.InitDNS),
	)
	defer app.Stop(ctx)
	err := app.Start(ctx)
	if err != nil {
		log.Error(err)
	}
	//<-app.Done()
	err = router.Run()
	if err != nil {
		log.Error(err)
	}
}
