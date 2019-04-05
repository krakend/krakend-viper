// Package viper defines a config parser implementation based on the viper pkg
package viper

import (
	"reflect"
	"unsafe"

	"github.com/devopsfaith/krakend/config"
	"github.com/spf13/viper"
)

// New creates a new parser using the viper library
func New() Parser {
	return Parser{viper.New()}
}

// Parser is a config parser using the viper library
type Parser struct {
	viper *viper.Viper
}

// Parser implements the Parse interface
func (p Parser) Parse(configFile string) (config.ServiceConfig, error) {
	p.viper.SetConfigFile(configFile)
	p.viper.AutomaticEnv()
	var cfg config.ServiceConfig
	if err := p.viper.ReadInConfig(); err != nil {
		return cfg, checkErr(err, configFile)
	}
	if err := p.viper.Unmarshal(&cfg); err != nil {
		return cfg, checkErr(err, configFile)
	}
	if err := cfg.Init(); err != nil {
		return cfg, checkErr(err, configFile)
	}

	return cfg, nil
}

func checkErr(err error, configFile string) error {
	switch e := err.(type) {
	case viper.ConfigParseError:
		var subErr error
		rs := reflect.ValueOf(&e).Elem()
		rf := rs.Field(0)
		ri := reflect.ValueOf(&subErr).Elem()

		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()

		ri.Set(rf)

		return checkErr(subErr, configFile)
	default:
		return config.CheckErr(err, configFile)
	}
}
