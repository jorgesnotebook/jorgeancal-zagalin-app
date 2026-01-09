package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/jorgeancal/zagalin/pkg/plugin"
)

func main() {
	if err := app.Manage("jorgeancal-zagalin", plugin.NewApp, app.ManageOpts{}); err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
