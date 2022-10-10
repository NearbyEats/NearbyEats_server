package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nearby-eats/clienthandler"
	"github.com/nearby-eats/datahub"
	config "github.com/nearby-eats/utils"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	config.PopulateConfig()

	// environment := flag.String("e", "development", "")
	flag.Usage = func() {
		fmt.Println("Usage: server -e {mode}")
		os.Exit(1)
	}
	flag.Parse()
	go datahub.Init()
	clienthandler.Init()
}
