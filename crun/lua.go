package crun

import (
	"github.com/cjoudrey/gluahttp"
	"github.com/kohkimakimoto/gluaenv"
	"github.com/kohkimakimoto/gluafs"
	"github.com/kohkimakimoto/gluatemplate"
	"github.com/kohkimakimoto/gluayaml"
	"github.com/otm/gluash"
	"github.com/yuin/gluare"
	"github.com/yuin/gopher-lua"
	gluajson "layeh.com/gopher-json"
	"net/http"
)

type LuaProcess struct {
	ScriptFile string
	LState     *lua.LState
}

func NewLuaProcess() *LuaProcess {
	return &LuaProcess{}
}

func (p *LuaProcess) Run(args []string) error {
	L := lua.NewState()
	defer L.Close()
	p.LState = L

	L.PreloadModule("json", gluajson.Loader)
	L.PreloadModule("fs", gluafs.Loader)
	L.PreloadModule("yaml", gluayaml.Loader)
	L.PreloadModule("template", gluatemplate.Loader)
	L.PreloadModule("env", gluaenv.Loader)
	L.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	L.PreloadModule("re", gluare.Loader)
	L.PreloadModule("sh", gluash.Loader)

	argtb := L.NewTable()
	for i, v := range args {
		L.RawSet(argtb, lua.LNumber(i), lua.LString(v))
	}
	L.SetGlobal("arg", argtb)

	err := L.DoFile(p.ScriptFile)
	if err != nil {
		return err
	}

	return nil
}
