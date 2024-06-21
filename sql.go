package main

import (
	sq "github.com/goclub/sql"
	xtime "github.com/goclub/time"
	"strconv"
)

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
