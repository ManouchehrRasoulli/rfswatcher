package main

import (
	"flag"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/client"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/logger"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/server"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/watcher"
	"log"
	"os"
)

func main() {
	var config string
	flag.StringVar(&config, "config", "config.yml", "specify configuration file for service.")
	flag.StringVar(&config, "c", "config.yml", "specify configuration file for service.")
	flag.Parse()

	lg := log.New(os.Stdout, "rfswatcher --> ", 1|4)
	clg := logger.NewColorLogger(lg)
	clg.Printcf(logger.ColorGreen, "start rfswatcher : with config file %v", config)

	cfg, err := pkg.ReadConfig(config)
	if err != nil {
		clg.Printcf(logger.ColorRed, "error rfswatcher : got error %v on reading configuration file %s", err, config)
		os.Exit(1)
	}

	clg.Printcf(logger.ColorBlue, "config rfswatcher : type: %s, address: %s, path: %s", cfg.ServiceType, cfg.Address, cfg.Path)
	switch cfg.ServiceType {
	case pkg.ServerType:
		{
			handler, err := filehandler.NewHandler(cfg.Path, lg)
			if err != nil {
				clg.Printcf(logger.ColorRed, "server error : got error %v on initiating file handler !", err)
				os.Exit(1)
			}
			server := server.NewServer(cfg.Address, lg, handler)
			defer server.Exit()

			watcher, err := watcher.NewWatcher(cfg.Path,
				watcher.WithCallbackFunction(handler.EventHook),
				watcher.WithCallbackFunction(server.EventHook))

			if err != nil {
				clg.Printcf(logger.ColorRed, "server error : got error %v on watcher !", err)
				os.Exit(1)
			}

			defer watcher.Close()

			err = server.Run()
			if err != nil {
				clg.Printcf(logger.ColorRed, "server error : got error %v running http server !!", err)
				os.Exit(1)
			}
		}
	case pkg.ClientType:
		{
			handler, err := filehandler.NewHandler(cfg.Path, lg)
			if err != nil {
				lg.Printf(logger.ColorRed, "client error : got error %v on initiating file handler !", err)
				os.Exit(1)
			}

			client := client.NewClient(cfg.Address, lg, handler)
			err = client.Run()
			if err != nil {
				lg.Printf(logger.ColorRed, "client error : got error %v on initialize connection with server !!", err)
				os.Exit(1)
			}
		}
	default:
		clg.Printf(logger.ColorRed, "error rfswatcher : exit !! invalid service type %s !", cfg.ServiceType)
		os.Exit(1)
	}
}
