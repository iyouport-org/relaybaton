package relaybaton

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"relaybaton-dev/pkg/config"
	"relaybaton-dev/pkg/core"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "TODO",
	Long:  "TODO",
	Run:   serverExec,
}

func serverExec(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var server *core.Server
	app := fx.New(
		fx.Provide(
			core.NewServer,
			config.NewConfServer,
			//goroutine.Default,
		),
		fx.Logger(log.StandardLogger()),
		fx.Populate(&server),
		fx.Invoke(config.InitLog, config.InitDNS),
	)
	defer app.Stop(ctx)
	err := app.Start(ctx)
	if err != nil {
		log.Error(err)
	}
	server.Run()
}
