package main

import (
	"context"
	_ "github.com/go-sql-driver/mysql"
	xconv "github.com/goclub/conv"
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	xjson "github.com/goclub/json"
	sl "github.com/goclub/slice"
	sq "github.com/goclub/sql"
	xtime "github.com/goclub/time"
	tlbs "github.com/goclub/tlbs"
	"github.com/xiaoqidun/qqwry"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// ------config

type ConfigKey struct {
	Key   string                  `yaml:"key"`
	Limit uint64                  `yaml:"limit"`
	API   map[string]ConfigKeyAPI `yaml:"api"`
}
type ConfigKeyAPI struct {
	Limit uint64 `yaml:"limit"`
}
type Config struct {
	Keys     []ConfigKey        `yaml:"keys"`
	Mysql    sq.MysqlDataSource `yaml:"mysql"`
	AuthKeys []string           `yaml:"auth_keys"`
}

//--------qqwry

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
		if strings.Contains(city, key) {
			result.ISP = value
			break
		}
	}
	return
}

//--------sql

// IDTlbsKeyUseRecord 用于类型约束
// 比如 userID managerID 都是 uint64,编码的时候如果传错就会出现bug
// 通过 IDTlbsKeyUseRecord 进行类型约束,如果参数不对编译器就会报错
type IDTlbsKeyUseRecord uint32

func NewIDTlbsKeyUseRecord(id uint32) IDTlbsKeyUseRecord {
	return IDTlbsKeyUseRecord(id)
}
func (id IDTlbsKeyUseRecord) Uint32() uint32 {
	return uint32(id)
}
func (id IDTlbsKeyUseRecord) IsZero() bool {
	return id == 0

}
func (id IDTlbsKeyUseRecord) String() string {
	return strconv.FormatUint(uint64(id), 10)
}

// 底层结构体,用于组合出 model
type TableTlbsKeyUseRecord struct {
	sq.WithoutSoftDelete
}

// TableName 给 TableName 加上指针 * 能避免 db.InsertModel(user) 这种错误， 应当使用 db.InsertModel(&user) 或
func (*TableTlbsKeyUseRecord) TableName() string { return "tlbs_key_use_record" }

// User model
type TlbsKeyUseRecord struct {
	Id      IDTlbsKeyUseRecord `db:"id" sq:"ignoreInsert"`
	Key     string             `db:"key"`
	Date    xtime.Date         `db:"date"`
	ApiPath string             `db:"api_path"`
	Count   uint32             `db:"count"`
	TableTlbsKeyUseRecord

	sq.DefaultLifeCycle
}

// AfterInsert 创建后自增字段赋值处理
func (v *TlbsKeyUseRecord) AfterInsert(result sq.Result) (err error) {
	var id uint64
	if id, err = result.LastInsertUint64Id(); err != nil {
		return
	}
	v.Id = IDTlbsKeyUseRecord(uint32(id))
	return
}

// Column dict
func (v TableTlbsKeyUseRecord) Column() (col struct {
	Id      sq.Column
	Key     sq.Column
	Date    sq.Column
	ApiPath sq.Column
	Count   sq.Column
}) {
	col.Id = "id"
	col.Key = "key"
	col.Date = "date"
	col.ApiPath = "api_path"
	col.Count = "count"

	return
}

//---------tlbs_proxy

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

// --util

func writeError(resp http.ResponseWriter, msg string) {
	var body []byte
	var err error
	if body, err = xjson.Marshal(map[string]any{
		"status":  1,
		"message": msg,
	}); err != nil {
		resp.Write([]byte(err.Error()))
		return
	}
	resp.Write(body)
}

//-------type

type TLBSIPReply struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Result    struct {
		IP       string `json:"ip"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		AdInfo struct {
			Nation     string `json:"nation"`
			Province   string `json:"province"`
			City       string `json:"city"`
			District   string `json:"district"`
			Adcode     int    `json:"adcode"`
			NationCode int    `json:"nation_code"`
		} `json:"ad_info"`
	} `json:"result"`
}
type TLBSGeoReply struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Result    struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		Address          string `json:"address"`
		AddressComponent struct {
			Nation       string `json:"nation"`
			Province     string `json:"province"`
			City         string `json:"city"`
			District     string `json:"district"`
			Street       string `json:"street"`
			StreetNumber string `json:"street_number"`
		} `json:"address_component"`
		AdInfo struct {
			NationCode    string `json:"nation_code"`
			Adcode        string `json:"adcode"`
			PhoneAreaCode string `json:"phone_area_code"`
			CityCode      string `json:"city_code"`
			Name          string `json:"name"`
			Location      struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			Nation   string `json:"nation"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Distance int    `json:"_distance"`
		} `json:"ad_info"`
		AddressReference struct {
			Town struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance int    `json:"_distance"`
				DirDesc  string `json:"_dir_desc"`
			} `json:"town"`
			LandmarkL2 struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"landmark_l2"`
			Street struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"street"`
			StreetNumber struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"street_number"`
			Crossroad struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"crossroad"`
		} `json:"address_reference"`
		FormattedAddresses struct {
			Recommend       string `json:"recommend"`
			Rough           string `json:"rough"`
			StandardAddress string `json:"standard_address"`
		} `json:"formatted_addresses"`
	} `json:"result"`
}
