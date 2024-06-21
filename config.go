package main

import sq "github.com/goclub/sql"

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

