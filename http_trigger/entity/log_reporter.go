package entity

import (
	"fmt"
	"github.com/SkyAPM/go2sky"
	language_agent "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"log"
	"os"
	"sort"
	"time"
)

func NewLogReporter() (go2sky.Reporter, error) {
	return &logReporter{logger: log.New(os.Stderr, "", log.LstdFlags)}, nil
}

type logReporter struct {
	logger *log.Logger
}

func (lr *logReporter) Boot(service string, serviceInstance string) {

}

func (lr *logReporter) Send(spans []go2sky.ReportedSpan) {
	if spans == nil {
		return
	}
	var allLogs []*language_agent.Log
	for i := range spans {
		allLogs = append(allLogs, spans[i].Logs()...)
	}
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Time < allLogs[j].Time
	})
	for i := range allLogs {
		data := allLogs[i].GetData()
		var log = fmt.Sprintf("[TIME]:%s ", time.Unix(allLogs[i].Time/1e3, 0).Format("2006-01-02 15:04:05"))
		for k := range data {
			log = log + fmt.Sprintf(" [%s]:%s ", data[k].Key, data[k].Value)
		}
		lr.logger.Println(log)
	}

}

func (lr *logReporter) Close() {
}
