
//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate npm -C vue-project run build
//go:embed vue-project/dist
var _assetsFS embed.FS
var webAssetsHandler = func()(http.Handler){
	assetsFS, err := fs.Sub(_assetsFS, "vue-project/dist")
	if err != nil {
		loger.Panic(err)
	}
	return http.StripPrefix("/main", http.FileServer(http.FS(assetsFS)))
}()
