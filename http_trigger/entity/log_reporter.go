package entity

import (
	"fmt"
	"github.com/SkyAPM/go2sky"
	"log"
	"os"
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
	for i := range spans {
		logs := spans[i].Logs()
		for j := range logs {
			data := logs[j].GetData()
			var log string
			for k := range data {
				log = log + fmt.Sprintf(" %s:%s", data[k].Key, data[k].Value)
			}
			lr.logger.Println(log)
		}
	}
}

func (lr *logReporter) Close() {
}
