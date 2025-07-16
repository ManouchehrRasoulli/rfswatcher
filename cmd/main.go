package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"

	"github.com/ManouchehrRasoulli/rfswatcher/pkg"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/client"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/logger"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/server"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/user"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/watcher"
)

func main() {
	var config string
	var createUserFlag bool
	var deleteUserFlag bool

	flag.StringVar(&config, "config", "config.yml", "specify configuration file for service.")
	flag.StringVar(&config, "c", "config.yml", "specify configuration file for service.")
	flag.BoolVar(&createUserFlag, "create-user", false, "create user")
	flag.BoolVar(&deleteUserFlag, "delete-user", false, "delete user")
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
			var um *user.UserManager = nil
			if cfg.Server.PwFile != "" {
				um := &user.UserManager{PwFile: cfg.Server.PwFile}

				if err := um.Init(); err != nil {
					clg.Printcf(logger.ColorRed, "server error : failed user manager initiallazation. %v", err)
				}

				if createUserFlag {
					if err := um.CreateUser(nil); err != nil {
						clg.Printcf(logger.ColorRed, "server error : failed to create user. %v", err)
						os.Exit(1)
					}
					os.Exit(0)
				} else if deleteUserFlag {
					if err := um.DeleteUser(""); err != nil {
						clg.Printcf(logger.ColorRed, "server error : failed to delete user. %v", err)
						os.Exit(1)
					}
					os.Exit(0)
				}
			}

			handler, err := filehandler.NewHandler(cfg.Path, lg)
			if err != nil {
				clg.Printcf(logger.ColorRed, "server error : got error %v on initiating file handler !", err)
				os.Exit(1)
			}
			var tls *server.ServerTLS = nil
			if cfg.Server.TLS.Cert != "" || cfg.Server.TLS.Key != "" {
				tls = &server.ServerTLS{Cert: cfg.Server.TLS.Cert, Key: cfg.Server.TLS.Key}
			}
			srv := server.NewServer(cfg.Address, cfg.Path, tls, um, lg, handler)
			defer srv.Exit()

			watch, err := watcher.NewWatcher(cfg.Path,
				watcher.WithCallbackFunction(handler.EventHook),
				watcher.WithCallbackFunction(srv.EventHook))

			if err != nil {
				clg.Printcf(logger.ColorRed, "server error : got error %v on watcher !", err)
				os.Exit(1)
			}

			defer watch.Close()

			err = srv.Run()
			if err != nil {
				clg.Printcf(logger.ColorRed, "server error : got error %v running http server !!", err)
				os.Exit(1)
			}
		}
	case pkg.ClientType:
		{
			handler, err := filehandler.NewHandler(cfg.Path, lg)
			if err != nil {
				clg.Printcf(logger.ColorRed, "client error : got error %v on initiating file handler !", err)
				os.Exit(1)
			}

			var tlsCfg *tls.Config
			if cfg.Client.TLS {
				tlsCfg = &tls.Config{}
			}

			cli := client.NewClient(cfg.Address, cfg.Client.Username, cfg.Client.Password, tlsCfg, lg, handler)
			err = cli.Run()
			if err != nil {
				clg.Printcf(logger.ColorRed, "client error : got error %v on initialize connection with server !!", err)
				os.Exit(1)
			}
		}
	default:
		clg.Printcf(logger.ColorRed, "error rfswatcher : exit !! invalid service type %s !", cfg.ServiceType)
		os.Exit(1)
	}
}
