package pg

import "time"

type Config struct {
	User            string        `yaml:"user"`
	Pwd             string        `yaml:"pwd"`
	Server          string        `yaml:"server"`
	DBName          string        `yaml:"db_name"`
	MaxIdleConns    string        `yaml:"max_idle_conns"`
	MaxOpenConns    string        `yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}
