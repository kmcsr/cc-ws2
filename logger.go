
package main

import (
	"os"

	"github.com/kmcsr/go-logger"
	"github.com/kmcsr/go-logger/logrus"
)

var loger = getLogger()

func getLogger()(loger logger.Logger){
	loger = logrus.Logger
	if os.Getenv("DEBUG") == "true" {
		loger.SetLevel(logger.TraceLevel)
	}else{
		loger.SetLevel(logger.InfoLevel)
		_, err := logger.OutputToFile(loger, "/var/log/cc_ws2/latest.log", os.Stdout)
		if err != nil {
			panic(err)
		}
	}
	return
}
