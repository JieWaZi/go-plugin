package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
)

type Loader struct {
	pluginDir   string
	storageDir  string
	pluginName  string
	handlerName string
}

func NewLoader(pluginName, handlerName string) (*Loader, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("get pwd failed,err:%s", err.Error())
		return nil, err
	}
	pluginDir := filepath.Join(wd, "plugin")
	storageDir := filepath.Join(wd, "storage")

	return &Loader{
		pluginDir:   pluginDir,
		storageDir:  storageDir,
		pluginName:  pluginName,
		handlerName: handlerName,
	}, nil
}

func (l *Loader) Compile() error {
	_, err := os.Stat(filepath.Join(l.storageDir, l.pluginName))
	if err != nil {
		return err
	}

	pluginFile := filepath.Join(l.pluginDir, fmt.Sprintf("%s.so", l.pluginName[:len(l.pluginName)-3]))
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o="+pluginFile, filepath.Join(l.storageDir, l.pluginName))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Printf("cmd run failed,err:%s", err.Error())
		return err
	}
	return nil
}

func (l *Loader) LoadPlugin() (plugin.Symbol, error) {
	pluginPath := filepath.Join(l.pluginDir, fmt.Sprintf("%s.so", l.pluginName[:len(l.pluginName)-3]))
	_, err := os.Stat(pluginPath)
	if err != nil {
		return nil, err
	}

	log.Printf("loading plugin from %v", pluginPath)
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, err
	}
	log.Printf("find handler: %v", l.handlerName)
	sym, err := p.Lookup(l.handlerName)
	if err != nil {
		return nil, err
	}

	return sym, nil
}
