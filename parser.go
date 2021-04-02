// Package viper defines a config parser implementation based on the viper pkg
package viper

import (
	"fmt"
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
	p.viper.SetEnvPrefix("krakend")
	p.viper.AutomaticEnv()
	var cfg config.ServiceConfig
	if err := p.viper.ReadInConfig(); err != nil {
		return cfg, checkErr(err, configFile)
	}
	if err := p.viper.Unmarshal(&cfg); err != nil {
		return cfg, checkErr(err, configFile)
	}
	cleanupServiceConfig(&cfg)
	if err := cfg.Init(); err != nil {
		return cfg, config.CheckErr(err, configFile)
	}

	return cfg, nil
}


// cleanupServiceConfig make sure ExtraConfig type is map[string]interface{}
func cleanupServiceConfig(cfg *config.ServiceConfig) {
	cfg.ExtraConfig = cleanConfigMap(cfg.ExtraConfig)
	for _, endpoint := range cfg.Endpoints {
		endpoint.ExtraConfig = cleanConfigMap(endpoint.ExtraConfig)

		for _, backend := range endpoint.Backend {
			backend.ExtraConfig = cleanConfigMap(backend.ExtraConfig)
		}
	}
}

func cleanConfigMap(cfg map[string]interface{}) map[string]interface{} {
	for k, v := range cfg {
		cfg[k] = cleanupMapValue(v)
	}
	return cfg
}

func cleanupMapValue(input interface{}) interface{} {
	switch data := input.(type) {
	case []interface{}:
		for key, value := range data {
			data[key] = cleanupMapValue(value)
		}
		return data
	case map[string]interface{}:
		for key, value := range data {
			data[key] = cleanupMapValue(value)
		}
		return data
	case map[interface{}]interface{}:
		output := make(map[string]interface{})
		for key, value := range data {
			output[fmt.Sprintf("%v", key)] = cleanupMapValue(value)
		}
		return output
	default:
		return data
	}
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
