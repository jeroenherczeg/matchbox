package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/coreos/pkg/flagutil"
	"google.golang.org/grpc"

	"github.com/coreos/coreos-baremetal/bootcfg/config"
	bootcfg "github.com/coreos/coreos-baremetal/bootcfg/server"
	pb "github.com/coreos/coreos-baremetal/bootcfg/server/serverpb"
	"github.com/coreos/coreos-baremetal/bootcfg/storage"
)

var (
	// version provided by compile time flag: -ldflags "-X main.version $GIT_SHA"
	version = "was not built properly"
	log     = capnslog.NewPackageLogger("github.com/coreos/coreos-baremetal/cmd/bootcfg-rpc", "main")
)

func main() {
	flags := struct {
		address    string
		configPath string
		dataPath   string
		version    bool
		help       bool
	}{}
	flag.StringVar(&flags.address, "address", "127.0.0.1:8081", "gRPC listen address")
	flag.StringVar(&flags.configPath, "config", "./data/config.yaml", "Path to config file")
	flag.StringVar(&flags.dataPath, "data-path", "./data", "Path to data directory")
	// subcommands
	flag.BoolVar(&flags.version, "version", false, "print version and exit")
	flag.BoolVar(&flags.help, "help", false, "print usage and exit")

	// parse command-line and environment variable arguments
	flag.Parse()
	if err := flagutil.SetFlagsFromEnv(flag.CommandLine, "BOOTCFG"); err != nil {
		log.Fatal(err.Error())
	}

	if flags.version {
		fmt.Println(version)
		return
	}

	if flags.help {
		flag.Usage()
		return
	}

	// validate arguments
	if url, err := url.Parse(flags.address); err != nil || url.String() == "" {
		log.Fatal("A valid HTTP listen address is required")
	}
	if finfo, err := os.Stat(flags.configPath); err != nil || finfo.IsDir() {
		log.Fatal("A path to a config file is required")
	}
	if finfo, err := os.Stat(flags.dataPath); err != nil || !finfo.IsDir() {
		log.Fatal("A path to a data directory is required")
	}

	// load bootstrap config
	cfg, err := config.LoadConfig(flags.configPath)
	if err != nil {
		log.Fatal(err)
	}

	// storage
	store := storage.NewFileStore(&storage.Config{
		Dir:    flags.dataPath,
		Groups: cfg.PBGroups(),
	})

	// gRPC Server
	log.Infof("starting bootcfg gRPC server on %s", flags.address)
	lis, err := net.Listen("tcp", flags.address)
	if err != nil {
		log.Fatalf("failed to start listening: %v", err)
	}
	grpcServer := grpc.NewServer()
	bootcfgServer := bootcfg.NewServer(&bootcfg.Config{
		Store: store,
	})
	pb.RegisterGroupsServer(grpcServer, bootcfgServer)
	pb.RegisterProfilesServer(grpcServer, bootcfgServer)
	grpcServer.Serve(lis)
}