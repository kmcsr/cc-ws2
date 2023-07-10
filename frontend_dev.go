
//go:build dev

package main

import (
	"net/http"
	"os"
	"os/exec"
)

var webAssetsHandler http.Handler = http.StripPrefix("/main", http.FileServer(http.Dir("vue-project/dist")))

func init(){
	loger.Infof("Starting frontend debug server")
	cmd := exec.Command("npm", "-C", "vue-project", "run", "build_dev")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		loger.Fatalf("Cannot start the frontend server: %v", err)
	}
	go func(){
		if err := cmd.Wait(); err != nil {
			loger.Errorf("npm frontend server exited: %v", err)
		}
	}()
}
