package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SkyAPM/go2sky"
	language_agent "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"log"
	"runtime/debug"
	"time"
)

type MySQL struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	User         string        `yaml:"user"`
	Password     string        `yaml:"password"`
	DBName       string        `yaml:"dbname"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	MaxOpenConns int           `yaml:"max_open_conns"`
	MaxLifetime  time.Duration `yaml:"max_lifetime"`
	db           *sql.DB
	tracer       *go2sky.Tracer
	ctx          context.Context
}

func (c *MySQL) Init() error {
	dsn := getDSNWithDB(*c)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		code, ok := getSQLErrCode(err)
		if !ok {
			return errors.WithStack(err)
		}
		if code == 1049 { // Database not exists
			if err := createDatabase(*c); err != nil {
				return err
			}
		}
		db, err = gorm.Open("mysql", dsn)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if c.MaxIdleConns == 0 {
		db.DB().SetMaxIdleConns(3)
	}
	if c.MaxOpenConns == 0 {
		db.DB().SetMaxOpenConns(5)
	}
	if c.MaxLifetime == 0 {
		db.DB().SetConnMaxLifetime(time.Hour)
	}

	c.db = db.DB()
	return nil
}
func (c *MySQL) Get() *sql.DB {
	if c.db == nil {
		c.Init()
	}
	return c.db
}

func getDSNWithDB(conf MySQL) string {
	return fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conf.User, conf.Password, conf.Host, conf.Port, conf.DBName)
}
func getSQLErrCode(err error) (int, bool) {
	mysqlErr, ok := errors.Cause(err).(*mysql.MySQLError)
	if !ok {
		return -1, false
	}

	return int(mysqlErr.Number), true
}
func createDatabase(conf MySQL) error {
	dsn := getBaseDSN(conf)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return errors.WithStack(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + conf.DBName)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
func getBaseDSN(conf MySQL) string {
	return fmt.Sprintf("%s:%s@(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		conf.User, conf.Password, conf.Host, conf.Port)
}

func (c *MySQL) SetTrace(tracer *go2sky.Tracer, ctx context.Context) {
	c.ctx = ctx
	c.tracer = tracer
}

func (c *MySQL) InsertSQL(sql string, args ...interface{}) error {
	span, err := c.tracer.CreateExitSpan(c.ctx, "DB Operation", fmt.Sprintf("%s:%d", c.Host, c.Port), func(header string) error {
		return nil
	})
	if err != nil {
		log.Printf("CreateExitSpan err:%s", err.Error())
		return err
	}
	defer span.End()
	span.Tag(go2sky.TagDBStatement, sql)
	span.Tag(go2sky.TagDBType, "mysql")
	span.SetSpanLayer(language_agent.SpanLayer_Database)
	span.Log(time.Now(), "INFO", fmt.Sprintf("insert sql:%s", sql))
	time.Sleep(1 * time.Second)
	_, err = c.db.Exec(sql, args...)
	if err != nil {
		span.Error(time.Now(), "ERROR", fmt.Sprintf("exec err:%s ,stack: %s", err.Error(), string(debug.Stack())))
		return err
	}
	return nil
}
