package relaybaton

import (
	"context"

	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/iyouport-org/relaybaton/pkg/core"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
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
	var client *core.Client
	app := fx.New(
		fx.Provide(
			core.NewClient,
			config.NewConfClient,
			goroutine.Default,
			core.NewRouter,
		),
		fx.Logger(log.StandardLogger()),
		fx.Invoke(config.InitLog, config.InitDNS),
		fx.Populate(&client),
	)
	defer app.Stop(ctx)
	err := app.Start(ctx)
	if err != nil {
		log.Error(err)
	}
	err = client.Run()
	if err != nil {
		log.Error(err)
	}
}
