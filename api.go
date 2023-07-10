
package main

import (
	"errors"
	"time"
)

const (
	tokenLen = 64

	cliTokenPrefix = "cli_"
	cliTokenLen = tokenLen + len(cliTokenPrefix)

	daemonTokenPrefix = "D_"
	daemonTokenLen = tokenLen + len(daemonTokenPrefix)
)

var PermDeniedErr = errors.New("Permission denied")

type API interface {
	NewCliToken(rootToken string, expiration *time.Time)(token string, err error)
	AuthCli(token string)(ok bool)
	AuthDaemon(token string, host string)(ok bool)
	ListServers(token string)(servers []string, err error)
	CheckPerm(token string, server string)(ok bool)
}
