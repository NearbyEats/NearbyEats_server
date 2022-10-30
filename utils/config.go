package utils

import (
	"fmt"
	"os"
	"reflect"
)

type configType struct {
	ENVIRONMENT         string
	CLIENT_HANDLER_PORT string
	DATA_HUB_PORT       string
	DATABASE_URI		string
	// DOCKER_SERVER_HOST string
	// DATABASE_NAME      string
	// DATABASE_HOST      string
	// DATABASE_PORT      string
	// DATABASE_USER      string
	// DATABASE_PASSWORD  string
}

var Config *configType = &configType{}

func PopulateConfig() {
	v := reflect.ValueOf(Config).Elem()
	typeOfConfig := v.Type()

	for i := 0; i < v.NumField(); i++ {
		key := typeOfConfig.Field(i).Name
		value, present := os.LookupEnv(key)

		if !present {
			panic(fmt.Sprintf("Required Environment Variable '%s' is not set.", key))
		}
		v.Field(i).SetString(value)
	}
}
