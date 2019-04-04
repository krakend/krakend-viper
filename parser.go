// Package viper defines a config parser implementation based on the viper pkg
package viper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"unsafe"

	"github.com/devopsfaith/krakend/config"
	"github.com/spf13/viper"
)

// New creates a new parser using the viper library
func New() config.Parser {
	return parser{viper.New()}
}

type parser struct {
	viper *viper.Viper
}

// Parser implements the Parse interface
func (p parser) Parse(configFile string) (config.ServiceConfig, error) {
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
	case *json.SyntaxError:
		return formatErr(err, configFile, e.Offset)
	case *json.UnmarshalTypeError:
		return formatErr(err, configFile, e.Offset)
	case *os.PathError:
		return fmt.Errorf(
			"'%s' (%s): %s",
			configFile,
			e.Op,
			e.Err.Error(),
		)
	case viper.ConfigParseError:
		var subErr error
		rs := reflect.ValueOf(&e).Elem()
		rf := rs.Field(0)
		ri := reflect.ValueOf(&subErr).Elem()

		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()

		ri.Set(rf)

		return checkErr(subErr, configFile)
	default:
		return fmt.Errorf("'%s': %v", configFile, err)
	}
}

func formatErr(err error, configFile string, offset int64) error {
	b, _ := ioutil.ReadFile(configFile)
	row, col := getErrorRowCol(b, offset)
	return fmt.Errorf(
		"'%s': %v, offset: %v, row: %v, col: %v",
		configFile,
		err.Error(),
		offset,
		row,
		col,
	)
}

func getErrorRowCol(source []byte, offset int64) (row, col int) {
	for i := int64(0); i < offset; i++ {
		v := source[i]
		if v == '\r' {
			continue
		}
		if v == '\n' {
			col = 0
			row++
			continue
		}
		col++
	}
	return
}
