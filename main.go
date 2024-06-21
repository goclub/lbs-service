package main

import (
	"context"
	_ "github.com/go-sql-driver/mysql"
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	xjson "github.com/goclub/json"
	sq "github.com/goclub/sql"
	tlbs "github.com/goclub/tlbs"
	"github.com/xiaoqidun/qqwry"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
)

func main() {
	xerr.PrintStack(run())
}

var db *sq.Database
var config = Config{}
var httpClient *xhttp.Client
var tlbsDistrict = tlbs.District{}

func init() {
	var err error
	if tlbsDistrict, err = tlbs.NewDistrict(tlbs.DataDistrict20220707); err != nil {
		panic(err)
	}
	// 从文件加载IP数据库
	if err := qqwry.LoadFile("qqwry.dat"); err != nil {
		panic(err)
	}
	yamlData, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}
	if err = yaml.Unmarshal(yamlData, &config); err != nil {
		panic(err)
	}
	xjson.PrintIndent("config", config)
	if db, _, err = sq.Open("mysql", config.Mysql.FormatDSN()); err != nil {
		panic(err)
	}
	if err = db.Ping(context.TODO()); err != nil {
		panic(err)
	}
	httpClient = xhttp.NewClient(nil)
}
func run() (err error) {
	r := xhttp.NewRouter(xhttp.RouterOption{
		OnCatchError: func(c *xhttp.Context, err error) error {
			if reject, as := xerr.AsReject(err); as {
				return c.WriteJSON(reject.Resp())
			}
			xerr.PrintStack(err)
			return c.WriteBytes([]byte("system error"))
		},
	})
	HandleQQwry(r)
	r.HandleFunc(xhttp.Route{xhttp.GET, "/favicon.ico"}, func(c *xhttp.Context) (err error) {
		return c.WriteBytes([]byte("/"))
	})
	r.PrefixHandler("/", &Proxy{})
	s := http.Server{
		Addr:    ":4324",
		Handler: r,
	}
	r.LogPatterns(&s)
	return s.ListenAndServe()
}
