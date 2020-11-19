package func_plugin

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SkyAPM/go2sky"
	language_agent "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type MySQL struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
	db           *sql.DB
	tracer       *go2sky.Tracer
	ctx          context.Context
}

func (c MySQL) Init() error {
	dsn := getDSNWithDB(c)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		code, ok := getSQLErrCode(err)
		if !ok {
			return errors.WithStack(err)
		}
		if code == 1049 { // Database not exists
			if err := createDatabase(c); err != nil {
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

func (c *MySQL) InsertSQL(sql string) error {
	span, _ := c.tracer.CreateExitSpan(c.ctx, "DB Operation", fmt.Sprintf("%s:%s", c.Host, c.Port), func(header string) error {
		return nil
	})
	span.Tag(go2sky.TagDBStatement, sql)
	span.Tag(go2sky.TagDBType, "mysql")
	span.SetSpanLayer(language_agent.SpanLayer_Database)
	logrus.Infof("insert sql:%s", sql)

	span.End()
	return nil
}
