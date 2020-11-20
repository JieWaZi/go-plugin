package middleware

import (
	"context"
	"database/sql"
	"github.com/SkyAPM/go2sky"
	"net/http"
	"time"
)

type DB interface {
	// DB初始化
	Init() error
	Get() *sql.DB
	InsertSQL(sql string, args ...interface{}) error
	SetTrace(tracer *go2sky.Tracer, ctx context.Context)
}

type MQ interface {
	Init() error
	SendDataMessage(key string, data []byte) error
	Close() error
	SetTrace(tracer *go2sky.Tracer, ctx context.Context)
}

type DataWriter struct {
	DB DB
	MQ MQ
}

type LogWriter interface {
	SendInfo(time.Time, string)
	SendError(time.Time, string)
	End()
}

type FunctionTool struct {
	DataWriter DataWriter
	LogWriter  LogWriter
}

func InitFunctionTool() (*FunctionTool, error) {
	return &FunctionTool{
		DataWriter: DataWriter{
			DB: &MySQL{
				Host:         "",
				Port:         0,
				User:         "",
				Password:     "",
				DBName:       "",
				MaxIdleConns: 0,
				MaxOpenConns: 0,
				MaxLifetime:  0,
			},
			MQ: &Kafka{
				KafkaBrokers:   []string{"localhost:1234"},
				KafkaDataTopic: "kafka-topic",
				DataPubClient:  nil,
			},
		},
	}, nil
}

// 用于爬虫使用的Function
type SpiderFunc func(ctx Context, tool *FunctionTool) error

// 普通http的Function
type HttpFunc http.HandlerFunc
