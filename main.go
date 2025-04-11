package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/yankeguo/rg"
)

type Options struct {
	CrtFile string `json:"crt_file" default:"/tls/tls.crt" validate:"required,file"`
	KeyFile string `json:"key_file" default:"/tls/tls.key" validate:"required,file"`
	Aliyun  struct {
		AccessKeyId     string `json:"access_key_id" validate:"required"`
		AccessKeySecret string `json:"access_key_secret" validate:"required"`
		RegionId        string `json:"region_id" validate:"required"`
	} `json:"aliyun"`
}

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

	var opts Options
	buf := rg.Must(os.ReadFile("config.json"))
	rg.Must0(json.Unmarshal(buf, &opts))
	rg.Must0(defaults.Set(&opts))
	rg.Must0(validator.New().Struct(opts))
}
