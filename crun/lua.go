package crun

import (
	"github.com/cjoudrey/gluahttp"
	"github.com/kohkimakimoto/gluaenv"
	"github.com/kohkimakimoto/gluafs"
	"github.com/kohkimakimoto/gluatemplate"
	"github.com/otm/gluash"
	glualibs "github.com/vadv/gopher-lua-libs"
	"github.com/yuin/gluare"
	"github.com/yuin/gopher-lua"
	"net/http"
)

type LuaApp struct {
	LState *lua.LState
}

func NewLuaApp() *LuaApp {
	return &LuaApp{}
}

func (lapp *LuaApp) Run(args []string) error {
	L := lua.NewState()
	defer L.Close()
	lapp.LState = L

	openLibs(L)

	argtb := L.NewTable()
	for i, v := range args {
		L.RawSet(argtb, lua.LNumber(i), lua.LString(v))
	}
	L.SetGlobal("arg", argtb)

	if len(args) > 0 {
		if err := L.DoFile(args[0]); err != nil {
			return err
		}
	}

	return nil
}

func openLibs(L *lua.LState) {
	glualibs.Preload(L)

	L.PreloadModule("fs", gluafs.Loader)
	L.PreloadModule("template", gluatemplate.Loader)
	L.PreloadModule("env", gluaenv.Loader)
	L.PreloadModule("re", gluare.Loader)
	L.PreloadModule("sh", gluash.Loader)
	L.PreloadModule("httpclient", gluahttp.NewHttpModule(&http.Client{}).Loader)

}
