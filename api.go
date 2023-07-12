
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

var (
	PermDeniedErr = errors.New("Permission denied")
	TokenNotExistsErr = errors.New("Token not exists")
)

type Token struct {
	Token string `json:"token"`
	Root  bool   `json:"root"`
	Expiration *time.Time `json:"expiration"`
}

type DaemonToken struct {
	Token      string     `json:"token"`
	Server     string     `json:"server"`
	Expiration *time.Time `json:"expiration"`
}

type API interface {
	NewCliToken(expiration *time.Time)(token string, err error)
	NewDaemonToken(server string, expiration *time.Time)(token string, err error)
	RemoveCliToken(token string)(err error)
	RemoveDaemonToken(token string)(err error)
	ListTokens()(tokens []Token, err error)
	ListDaemonTokens()(tokens []DaemonToken, err error)
	AuthCli(token string)(ok bool)
	AuthDaemon(token string, host string)(ok bool)
	CheckRootToken(token string)(ok bool)
	SetRoot(token string, value bool)(err error)
	CreateServer(id string)(err error)
	RemoveServer(id string)(err error)
	ListServers(token string)(servers []string, err error)
	CheckPerm(token string, server string)(ok bool)
	SetPerm(token string, server string, ok bool)(err error)
}

func preProcessCliToken(clitoken string)(token string, ok bool){
	if len(clitoken) != cliTokenLen || clitoken[:len(cliTokenPrefix)] != cliTokenPrefix {
		return "", false
	}
	return clitoken[len(cliTokenPrefix):], true
}
