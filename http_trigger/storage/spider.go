package main

import (
	"github.com/JieWazi/goplugin/http_trigger/middleware"
	"time"
)

func Spider(ctx middleware.Context, tool *middleware.FunctionTool) error {
	ctx.Put("haha", "11")
	time.Sleep(2 * time.Second)
	err := tool.DataWriter.DB.InsertSQL("ssss")
	if err != nil {
		return err
	}
	tool.LogWriter.SendInfo(time.Now(), "test")
	return nil
}
