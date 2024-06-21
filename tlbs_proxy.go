package main

import (
	"context"
	xconv "github.com/goclub/conv"
	xerr "github.com/goclub/error"
	xjson "github.com/goclub/json"
	sl "github.com/goclub/slice"
	sq "github.com/goclub/sql"
	xtime "github.com/goclub/time"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func proxyRequest(resp http.ResponseWriter, req *http.Request) (apiPath string, canUseKey string, err error) {
	ctx := context.Background()
	apiPath = matchAPIPath(req)
	today := xtime.Today(xtime.LocChina)
	tempKey := []ConfigKey{}
	for _, k := range config.Keys {
		tempKey = append(tempKey, k)
	}
	sl.Shuffle(tempKey)
	for _, v := range tempKey {
		limit := v.API[apiPath].Limit
		if limit == 0 {
			limit = v.Limit
		}
		col := TlbsKeyUseRecord{}.Column()
		var hasRecord bool
		if hasRecord, err = db.Has(ctx, &TlbsKeyUseRecord{}, sq.QB{
			Where: sq.
				And(col.Key, sq.Equal(v.Key)).
				And(col.Date, sq.Equal(today)).
				And(col.ApiPath, sq.Equal(apiPath)),
		}); err != nil {
			return
		}
		if hasRecord == false {
			if err = db.InsertModel(ctx, &TlbsKeyUseRecord{
				Key:     v.Key,
				Date:    today,
				ApiPath: apiPath,
				Count:   0,
			}, sq.QB{
				UseInsertIgnoreInto: true,
			}); err != nil {
				return
			}
		}
		var aff int64
		if aff, err = db.UpdateAffected(ctx, &TlbsKeyUseRecord{}, sq.QB{
			Where: sq.
				And(col.Key, sq.Equal(v.Key)).
				And(col.Date, sq.Equal(today)).
				And(col.ApiPath, sq.Equal(apiPath)).
				And(col.Count, sq.LT(limit)),
			Set:   sq.SetRaw(`count = count +1`),
			Limit: 1,
		}); err != nil {
			return
		}
		if aff == 1 {
			canUseKey = v.Key
			break
		}
	}
	if canUseKey == "" {
		err = xerr.Reject(1, "没有可用的key", true)
		return
	}
	return
}

type Proxy struct{}

func matchAPIPath(req *http.Request) (apiPath string) {
	path := "/" + req.URL.Path
	apiPath = path
	q := req.URL.Query()
	switch path {
	case "/ws/geocoder/v1/":
		switch true {
		case q.Get("location") != "":
			apiPath = path + "?location=*"
		case q.Get("address") != "":
			apiPath = path + "?address=*"
		}
	}
	return apiPath
}

func (p Proxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if sl.Contains(config.AuthKeys, req.URL.Query().Get("key")) == false {
		writeError(resp, "key 错误")
		return
	}
	newURL := &url.URL{
		Scheme:   "https",
		Host:     "apis.map.qq.com",
		Path:     req.URL.Path,     // 使用原始请求的 Path
		RawQuery: req.URL.RawQuery, // 使用原始请求的 Query 参数
	}
	q := newURL.Query()
	var canUseKey string
	var err error
	var apiPath string
	var logValues []any
	if apiPath, canUseKey, err = proxyRequest(resp, req); err != nil {
		var bytes []byte
		if bytes, err = localQuery(apiPath, q); err != nil {
			resp.Write([]byte(err.Error()))
			return
		}
		resp.Write(bytes)
		logValues = []any{"", req.URL.String(), "", string(bytes)}
		return
		//if reject, ok := xerr.AsReject(err); ok {
		//	writeError(resp, reject.Message)
		//	return
		//}
		//xerr.PrintStack(err)
		//writeError(resp, "system error")
		return
	}
	q.Set("key", canUseKey) // 修改 key 参数
	newURL.RawQuery = q.Encode()
	var r *http.Request
	if r, err = http.NewRequest(req.Method, newURL.String(), nil); err != nil {
		return
	}
	logValues = []any{canUseKey, req.URL.String(), newURL.String()}
	defer func() {
		if _, err = db.Exec(context.TODO(), "INSERT INTO tlbs_key_log (`key`, `r_url`, `t_url`, `body`) VALUES (?, ?, ?, ?)", logValues); err != nil {
			xerr.PrintStack(err)
			err = nil
			// 忽略插入错误
		}
	}()
	if proxyResp, bodyClose, statusCode, err := httpClient.Do(r); err != nil {
		logValues = append(logValues, err.Error())
		writeError(resp, err.Error())
		return
	} else {
		defer bodyClose()
		resp.WriteHeader(statusCode)
		var b []byte
		if b, err = ioutil.ReadAll(proxyResp.Body); err != nil {
			return
		}
		body := string(b)
		tlbsReply := struct {
			Status    int    `json:"status"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		}{}
		if err = xjson.Unmarshal(b, &tlbsReply); err != nil {
			return
		}
		if tlbsReply.Status != 0 {
			var bytes []byte
			if bytes, err = localQuery(apiPath, q); err != nil {
				resp.Write([]byte(err.Error()))
				return
			}
			if len(bytes) != 0 {
				b = bytes
				body += "\n\n" + string(b)
			}
		}
		logValues = append(logValues, body)
		resp.Write(b)
		return
	}
	return
}

func localQuery(apiPath string, q url.Values) (bytes []byte, err error) {
	switch apiPath {
	case "/ws/geocoder/v1/?location=*":
		var result QqwryResult
		ip := q.Get("ip")
		if result, err = qqwryParse(ip); err != nil {
			bytes = []byte(err.Error())
			return
		}
		geoReply := TLBSGeoReply{}
		location := result.Lbs.Province.Location
		if result.Lbs.City.Location.Lat != 0 {
			location = result.Lbs.City.Location
		}
		if result.Lbs.District.Location.Lat != 0 {
			location = result.Lbs.District.Location
		}
		geoReply.Result.Location = location
		geoReply.Result.Address = result.Lbs.Province.Fullname + result.Lbs.City.Fullname + result.Lbs.District.Fullname
		geoReply.Result.AddressComponent.Province = result.Lbs.Province.Fullname
		geoReply.Result.AddressComponent.City = result.Lbs.City.Fullname
		geoReply.Result.AddressComponent.District = result.Lbs.District.Fullname
		geoReply.Result.AdInfo.Adcode = result.Lbs.Adcode
		geoReply.Result.AdInfo.Name = strings.Join([]string{result.Lbs.Province.Fullname, result.Lbs.City.Fullname, result.Lbs.District.Fullname}, ",")
		geoReply.Result.AdInfo.Location = location
		geoReply.Result.AdInfo.Province = result.Lbs.Province.Fullname
		geoReply.Result.AdInfo.City = result.Lbs.City.Fullname
		geoReply.Result.AdInfo.District = result.Lbs.District.Fullname

		geoReply.Result.FormattedAddresses.Rough = geoReply.Result.Address
		geoReply.Result.FormattedAddresses.Recommend = geoReply.Result.Address
		geoReply.Result.FormattedAddresses.StandardAddress = geoReply.Result.Address
		if bytes, err = xjson.Marshal(geoReply); err != nil {
			xerr.PrintStack(err)
		}
	case "/ws/location/v1/ip":
		var result QqwryResult
		ip := q.Get("ip")
		if result, err = qqwryParse(ip); err != nil {
			bytes = []byte(err.Error())
			return
		}
		ipReply := TLBSIPReply{}
		ipReply.Result.IP = ip
		location := result.Lbs.Province.Location
		if result.Lbs.City.Location.Lat != 0 {
			location = result.Lbs.City.Location
		}
		if result.Lbs.District.Location.Lat != 0 {
			location = result.Lbs.District.Location
		}
		ipReply.Result.Location = location
		ipReply.Result.AdInfo.Province = result.Lbs.Province.Fullname
		ipReply.Result.AdInfo.City = result.Lbs.City.Fullname
		ipReply.Result.AdInfo.District = result.Lbs.District.Fullname
		if adcode, err := xconv.StringInt(result.Lbs.Adcode); err != nil {
			// 不做处理
			xerr.PrintStack(err)
		} else {
			ipReply.Result.AdInfo.Adcode = adcode
		}
		if bytes, err = xjson.Marshal(ipReply); err != nil {
			xerr.PrintStack(err)
		}
	}
	return
}
