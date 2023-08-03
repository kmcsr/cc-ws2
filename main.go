
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var startTime = time.Now() // or maybe build time

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

var defaultConfig = &Config{
	Host: "",
	Port: 80,
}
var config *Config = loadConfig()

func loadConfig()(cfg *Config){
	var configPath = filepath.Join(DataDir, "config.json")
	var data []byte
	var err error
	if data, err = os.ReadFile(configPath); err != nil {
		return defaultConfig
		// loger.Fatalf("Cannot read config at %s: %v", configPath, err)
	}
	cfg = new(Config)
	if err = json.Unmarshal(data, cfg); err != nil {
		loger.Fatalf("Cannot parse config at %s: %v", configPath, err)
	}
	return
}

func main(){
	hooksDir := filepath.Join(DataDir, "hooks")

	username := os.Getenv("DB_USER")
	passwd := os.Getenv("DB_PASSWD")
	dbaddr := os.Getenv("DB_ADDR")
	dbname := os.Getenv("DB_NAME")
	if len(username) == 0 || len(dbaddr) == 0 || len(dbname) == 0 {
		loger.Fatalf("Please set the envs `DB_USER`, `DB_PASSWD`, `DB_ADDR`, `DB_NAME`")
	}
	dtapi, err := NewMySQLAPI(username, passwd, dbaddr, dbname)
	if err != nil {
		loger.Fatalf("Cannot init mysql api: %v", err)
	}
	fsapi := NewOSFsAPI(DataDir)

	handler := NewHandler(dtapi, fsapi)

	{
		loger.Info("Loading hook plugins...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
		errs := handler.HookManager().LoadFromDir(ctx, hooksDir)
		cancel()
		if len(errs) != 0 {
			for _, e := range errs {
				loger.Error(e)
			}
		}
	}

	server := &http.Server{
		Addr: net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Handler: logMiddleWare(handler.NewServeMux()),
	}

	done := make(chan struct{}, 0)
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP)

	go func(){
		defer close(done)
		loger.Infof("Server start at %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			loger.Fatalf("Server exit by: %v", err)
		}
	}()

	for {
		select {
		case sig := <-sigch:
			if sig == syscall.SIGHUP { // reload
				loger.Warnf("Reload signal received [%s]", sig.String())
				loger.Info("Reloading hook plugins...")
				{
					ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
					errs := handler.HookManager().ReloadFromDir(ctx, hooksDir)
					cancel()
					if len(errs) != 0 {
						for _, e := range errs {
							loger.Error(e)
						}
					}
				}
			}else{
				ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
				server.Shutdown(ctx)
				cancel()
				return
			}
		case <-done:
			return
		}
	}
}

func logMiddleWare(next http.Handler)(http.Handler){
	return (http.HandlerFunc)(func(rw http.ResponseWriter, req *http.Request){
		loger.Infof("[%s] %s %s", req.RemoteAddr, req.Method, req.URL.Path)
		next.ServeHTTP(rw, req)
	})
}
