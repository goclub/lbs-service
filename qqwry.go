package main

import (
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	"github.com/goclub/tlbs"
	"github.com/xiaoqidun/qqwry"
	"strings"
)

func HandleQQwry(r *xhttp.Router) {
	r.HandleFunc(xhttp.Route{xhttp.GET, "/qqwry/ip"}, func(c *xhttp.Context) (err error) {
		q := c.Request.URL.Query()
		pass := false
		for _, key := range config.AuthKeys {
			if q.Get("key") == key {
				pass = true
				break
			}
		}
		if pass == false {
			return xerr.Reject(1, "key错误", false)
		}
		var reply struct {
			QqwryResult
			xerr.Resp
		}
		ip := q.Get("ip")
		if ip == "" {
			return xerr.Reject(1, "IP不能为空", false)
		}
		if reply.QqwryResult, err = qqwryParse(ip); err != nil {
			return
		}
		return c.WriteJSON(reply)
	})
}

type QqwryResult struct {
	Lbs tlbs.Relationship `json:"lbs"`
	ISP string            `json:"isp"`
	Raw struct {
		City string `json:"city"`
		ISP  string `json:"isp"`
	} `json:"qqwry"`
}

var ispCodeMap = map[string]string{
	"移动": "YD",
	"联通": "LT",
	"电信": "DX",
	"广电": "GD",
}

func qqwryParse(ip string) (result QqwryResult, err error) {
	if ip == "" {
		err = xerr.New("ip为空")
		return
	}
	var isp string
	var city string
	if city, isp, err = qqwry.QueryIP(ip); err != nil {
		return
	}
	result.Raw.ISP = isp
	result.Raw.City = city
	r, hasR := tlbsDistrict.RelationshipByAddress(city)
	if hasR {
		result.Lbs = r
	}
	result.ISP = isp
	for key, value := range ispCodeMap {
		if strings.Contains(city, value) {
			result.ISP = key
			break
		}
	}
	return
}
