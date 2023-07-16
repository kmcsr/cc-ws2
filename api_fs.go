
package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

const pluginsDirName = "plugins"

type OSFsAPI struct {
	Base string
}

var _ FsAPI = (*OSFsAPI)(nil)

func NewOSFsAPI(base string)(api *OSFsAPI){
	api = &OSFsAPI{
		Base: base,
	}
	api.provideDir()
	return
}

func (api *OSFsAPI)joinPath(paths ...string)(string){
	paths = append(paths, "")
	copy(paths[1:], paths)
	paths[0] = api.Base
	return filepath.Join(paths...)
}

func (api *OSFsAPI)provideDir(paths ...string)(path string, err error){
	path = api.joinPath(paths...)
	if err = os.MkdirAll(path, 0750); err != nil {
		if !os.IsExist(err) {
			return
		}
		err = nil
	}
	return
}

func (api *OSFsAPI)tryPath(paths ...string)(path string, err error){
	path = api.joinPath(paths...)
	if _, err = os.Stat(path); err != nil {
		return
	}
	return
}

func (api *OSFsAPI)CreatePlugin(plugin WebScriptMeta)(err error){
	path, err := api.provideDir(pluginsDirName, plugin.Id, plugin.Version)
	if err != nil {
		return
	}
	buf, err := json.MarshalIndent(plugin, "", "\t")
	if err = os.WriteFile(filepath.Join(path, "meta.json"), buf, 0644); err != nil {
		return
	}
	return
}

func (api *OSFsAPI)DeletePlugin(plugin WebScriptId)(err error){
	path, err := api.tryPath(pluginsDirName, plugin.Id, plugin.Version)
	if err != nil {
		err = PluginNotExistsErr
		return
	}
	if err = os.RemoveAll(path); err != nil {
		return
	}
	return
}

func (api *OSFsAPI)ListPlugins()(plugins []WebScriptMeta, err error){
	path := api.joinPath(pluginsDirName)
	entries, er := os.ReadDir(path)
	if er != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			pluginId := e.Name()
			pluginDir := filepath.Join(path, pluginId)
			versions, er := os.ReadDir(pluginDir)
			if er == nil {
				for _, v := range versions {
					if v.IsDir() {
						version := v.Name()
						buf, er := os.ReadFile(filepath.Join(pluginDir, version, "meta.json"))
						if er == nil {
							var meta WebScriptMeta
							if er = json.Unmarshal(buf, &meta); er == nil {
								if meta.Id != pluginId || meta.Version != version {
									loger.Errorf("Plugin id dismatch metadata: path=%q version=%s meta=%v", pluginDir, version, meta)
									continue
								}
								plugins = append(plugins, meta)
							}
						}
					}
				}
			}
		}
	}
	return
}

func (api *OSFsAPI)ListPluginFiles(plugin WebScriptId, path string)(files []*FileInfo, err error){
	pth, er := api.tryPath(pluginsDirName, plugin.Id, plugin.Version)
	if er != nil {
		err = PluginNotExistsErr
		return
	}
	entries, err := os.ReadDir(filepath.Join(pth, filepath.FromSlash(path)))
	if err != nil {
		return
	}
	files = make([]*FileInfo, 0, len(entries))
	for _, e := range entries {
		if fi, err := e.Info(); err == nil {
			files = append(files, &FileInfo{
				Name: fi.Name(),
				IsDir: fi.IsDir(),
				ModTime: fi.ModTime(),
			})
		}
	}
	return
}

func (api *OSFsAPI)GetPluginFile(plugin WebScriptId, path string)(r io.ReadSeekCloser, modTime time.Time, err error){
	if _, er := api.tryPath(pluginsDirName, plugin.Id, plugin.Version); er != nil {
		err = PluginNotExistsErr
		return
	}
	fd, err := os.Open(api.joinPath(pluginsDirName, plugin.Id, plugin.Version, filepath.FromSlash(path)))
	if err != nil {
		return
	}
	r = fd
	stat, err := fd.Stat()
	if err != nil {
		fd.Close()
		return
	}
	if stat.IsDir() {
		fd.Close()
		err = ErrIsDir
		return
	}
	modTime = stat.ModTime()
	return	
}

func (api *OSFsAPI)PutPluginFile(plugin WebScriptId, path string, r io.Reader)(err error){
	if _, er := api.tryPath(pluginsDirName, plugin.Id, plugin.Version); er != nil {
		err = PluginNotExistsErr
		return
	}
	path, name := splitByteR(path, '/')
	if path, err = api.provideDir(pluginsDirName, plugin.Id, plugin.Version, filepath.FromSlash(path)); err != nil {
		return
	}
	if err = safeDownload(r, filepath.Join(path, name)); err != nil {
		return
	}
	return
}

func (api *OSFsAPI)DelPluginFile(plugin WebScriptId, path string)(err error){
	if _, er := api.tryPath(pluginsDirName, plugin.Id, plugin.Version); er != nil {
		err = PluginNotExistsErr
		return
	}
	fpath, err := api.tryPath(pluginsDirName, plugin.Id, plugin.Version, filepath.FromSlash(path))
	if err != nil {
		return
	}
	if err = os.RemoveAll(fpath); err != nil {
		return
	}
	return
}
