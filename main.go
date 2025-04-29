package main

import (
	"flag"
	"log"
	"os"

	"github.com/yankeguo/rg"
)

func main() {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Printf("exit with error: %s", err.Error())
		os.Exit(1)
	}()
	defer rg.Guard(&err)

	var (
		optConf string
	)

	flag.StringVar(&optConf, "conf", "config.json", "config file")
	flag.Parse()

	log.Println("loading config from:", optConf)

	opts := rg.Must(LoadOptions(optConf))

	if opts.Aliyun != nil {
		rg.Must0(updateAliyun(opts.Cert, opts.Aliyun))
	}

	if opts.Qcloud != nil {
		rg.Must0(updateQcloud(opts.Cert, opts.Qcloud))
	}
}
