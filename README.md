# KrakenD viper

A config parser for the [KrakenD](http://krakend.io/) framework

## How to use it

Import the package

	import "github.com/devopsfaith/krakend-viper"

And you are ready for building a parser and get the config from any format supported by viper

	parser := viper.New()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
