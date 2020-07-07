package relaybaton

import (
	"context"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"relaybaton/pkg/config"
	"relaybaton/pkg/core"
	"time"
)

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "TODO",
	Long:  "TODO",
	Run:   clientExec,
}

func clientExec(cmd *cobra.Command, args []string) {
	for {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var client *core.Client
		app := fx.New(
			fx.Provide(
				core.NewClient,
				config.NewConfClient,
				goroutine.Default,
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
		//<-app.Done()
		err = client.Run()
		if err != nil {
			log.Error(err)
		}
		err = app.Stop(ctx)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(5 * time.Second)
	}
}
