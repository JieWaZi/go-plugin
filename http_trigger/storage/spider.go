package main

import (
	"github.com/JieWazi/goplugin/http_trigger/func_plugin"
	"time"
)

func Spider(ctx func_plugin.Context, tool *func_plugin.FunctionTool) {
	ctx.Put("haha", "11")
	time.Sleep(2 * time.Second)
	tool.DataWriter.DB.InsertSQL("ssss")
	tool.LogWriter.SendInfo(time.Now(), "test")
}
