package global

import (
	"github.com/JieWazi/goplugin/http_trigger/middleware"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type Configuration struct {
	MySQL    *middleware.MySQL     `yaml:"mysql"`
	Kafka    *middleware.Kafka     `yaml:"kafka"`
	SkyTrace *middleware.SkyTracer `yaml:"sky_trace"`
}

type MySQL struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	User         string        `yaml:"user"`
	Password     string        `yaml:"password"`
	DBName       string        `yaml:"dbname"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	MaxOpenConns int           `yaml:"max_open_conns"`
	MaxLifetime  time.Duration `yaml:"max_lifetime"`
}

type Kafka struct {
	KafkaBrokers   []string `yaml:"brokers"`
	KafkaDataTopic string   `yaml:"topic"`
}

type SkyTrace struct {
	Endpoint    string `yaml:"endpoint"`
	ServiceName string `yaml:"service_name"`
}

func InitByYamlConfig(path string) (*Configuration, error) {
	conf := &Configuration{}
	if f, err := os.Open(path); err != nil {
		return nil, err
	} else {
		yaml.NewDecoder(f).Decode(conf)
	}
	return conf, nil
}

var Conf *Configuration

func init() {
	pwd, _ := os.Getwd()
	conf, err := InitByYamlConfig(pwd + "/global/conf.yaml")
	if err != nil {
		panic(err)
	}

	err = conf.MySQL.Init()
	if err != nil {
		panic(err)
	}
	err = conf.Kafka.Init()
	if err != nil {
		panic(err)
	}
	err = conf.SkyTrace.Init()
	if err != nil {
		panic(err)
	}
	Conf = conf
}
