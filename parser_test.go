package viper

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestNew_ok(t *testing.T) {
	configContents := []struct {
		format  string
		path    string
		content []byte
	}{
		{"json", "/tmp/ok.json", jsonCfg},
		{"toml", "/tmp/ok.toml", tomlCfg},
		{"yaml", "/tmp/ok.yml", yamlCfg},
	}
	for _, configContent := range configContents {
		t.Run(configContent.format, func(t *testing.T) {

			if err := ioutil.WriteFile(configContent.path, configContent.content, 0644); err != nil {
				t.FailNow()
			}

			serviceConfig, err := New().Parse(configContent.path)
			if err != nil {
				t.Error("Unexpected error. Got", err.Error())
			}

			endpoint := serviceConfig.Endpoints[0]
			endpointExtraConfiguration := endpoint.ExtraConfig

			if endpointExtraConfiguration != nil {
				testExtraConfig(endpointExtraConfiguration, t)
			} else {
				t.Error("Extra config is not present in EndpointConfig")
			}

			backend := endpoint.Backend[0]
			backendExtraConfiguration := backend.ExtraConfig
			if backendExtraConfiguration != nil {
				testExtraConfig(backendExtraConfiguration, t)
			} else {
				t.Error("Extra config is not present in BackendConfig")
			}

			if err := os.Remove(configContent.path); err != nil {
				t.FailNow()
			}
		})
	}
}

func TestNew_errorMessages(t *testing.T) {
	for _, configContent := range []struct {
		name    string
		path    string
		content []byte
		expErr  string
	}{
		{
			name:    "case0",
			path:    "/tmp/ok.json",
			content: []byte(`{`),
			expErr:  "'/tmp/ok.json': unexpected end of JSON input, offset: 1, row: 0, col: 1",
		},
		{
			name:    "case1",
			path:    "/tmp/ok.json",
			content: []byte(`>`),
			expErr:  "'/tmp/ok.json': invalid character '>' looking for beginning of value, offset: 1, row: 0, col: 1",
		},
		{
			name:    "case2",
			path:    "/tmp/ok.json",
			content: []byte(`"`),
			expErr:  "'/tmp/ok.json': unexpected end of JSON input, offset: 1, row: 0, col: 1",
		},
		{
			name:    "case3",
			path:    "/tmp/ok.json",
			content: []byte(``),
			expErr:  "'/tmp/ok.json': unexpected end of JSON input, offset: 0, row: 0, col: 0",
		},
		{
			name:    "case4",
			path:    "/tmp/ok.json",
			content: []byte(`[{}]`),
			expErr:  "'/tmp/ok.json': json: cannot unmarshal array into Go value of type map[string]interface {}, offset: 1, row: 0, col: 1",
		},
		{
			name:    "case5",
			path:    "/tmp/ok.json",
			content: []byte(`42`),
			expErr:  "'/tmp/ok.json': json: cannot unmarshal number into Go value of type map[string]interface {}, offset: 2, row: 0, col: 2",
		},
		{
			name:    "case6",
			path:    "/tmp/ok.json",
			content: []byte("\r\n42"),
			expErr:  "'/tmp/ok.json': json: cannot unmarshal number into Go value of type map[string]interface {}, offset: 4, row: 1, col: 2",
		},
		{
			name: "case7",
			path: "/tmp/ok.json",
			content: []byte(`{
	"version": 3,
	"name": "My lovely gateway",
	"port": 8080,
	"cache_ttl": 3600
	"timeout": "3s",
	"endpoints": []
}`),
			expErr: "'/tmp/ok.json': invalid character '\"' after object key:value pair, offset: 83, row: 5, col: 2",
		},
	} {
		t.Run(configContent.name, func(t *testing.T) {
			if err := ioutil.WriteFile(configContent.path, configContent.content, 0644); err != nil {
				t.Error(err)
				return
			}

			_, err := New().Parse(configContent.path)
			if err == nil {
				t.Errorf("%s: Expecting error", configContent.name)
				return
			}
			if errMsg := err.Error(); errMsg != configContent.expErr {
				t.Errorf("%s: Unexpected error. Got '%s' want '%s'", configContent.name, errMsg, configContent.expErr)
				return
			}

			if err := os.Remove(configContent.path); err != nil {
				t.Errorf("%s: %s", err.Error(), configContent.name)
				return
			}
		})
	}
}

func testExtraConfig(extraConfig map[string]interface{}, t *testing.T) {
	userVar := extraConfig["user"]
	if userVar != "test" {
		t.Error("User in extra config is not test")
	}
	parents := extraConfig["parents"].([]interface{})
	if parents[0] != "gomez" {
		t.Error("Parent 0 of user us not gomez")
	}
	if parents[1] != "morticia" {
		t.Error("Parent 1 of user us not morticia")
	}

	testExtraNestedConfigKey(extraConfig, t)
}

func testExtraNestedConfigKey(extraConfig map[string]interface{}, t *testing.T) {
	namespace := "nested_data"
	v, ok := extraConfig[namespace]
	if !ok {
		return
	}

	type nestedConfig struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		t.Error("marshal nested config key error: ", err.Error())
		return
	}

	var cfg nestedConfig
	if err = json.Unmarshal(jsonBytes, &cfg); err != nil {
		t.Error("unmarshal nested config key error: ", err.Error())
		return
	}

	if cfg.Data.Status != "OK" {
		t.Errorf("nested config key parse error: %+v\n", cfg)
	}
}

func TestNew_unknownFile(t *testing.T) {
	_, err := New().Parse("/nowhere/in/the/fs.json")
	if err == nil || err.Error() != "'/nowhere/in/the/fs.json' (open): no such file or directory" {
		t.Errorf("Error expected. Got '%s'", err)
	}
}

func TestNew_readingError(t *testing.T) {
	wrongConfigPath := "/tmp/reading.json"
	wrongConfigContent := []byte("{hello\ngo\n")
	if err := ioutil.WriteFile(wrongConfigPath, wrongConfigContent, 0644); err != nil {
		t.FailNow()
	}

	expected := "'/tmp/reading.json': invalid character 'h' looking for beginning of object key string, offset: 2, row: 0, col: 2"
	_, err := New().Parse(wrongConfigPath)
	if err == nil || err.Error() != expected {
		t.Errorf("Error expected. Got '%s'", err)
	}
	if err = os.Remove(wrongConfigPath); err != nil {
		t.FailNow()
	}
}

func TestNew_initError(t *testing.T) {
	wrongConfigPath := "/tmp/unmarshall.json"
	wrongConfigContent := []byte("{\"a\":42}")
	if err := ioutil.WriteFile(wrongConfigPath, wrongConfigContent, 0644); err != nil {
		t.FailNow()
	}

	_, err := New().Parse(wrongConfigPath)
	if err == nil || err.Error() != "'/tmp/unmarshall.json': unsupported version: 0 (want: 3)" {
		t.Error("Error expected. Got", err)
	}
	if err = os.Remove(wrongConfigPath); err != nil {
		t.FailNow()
	}
}

var (
	jsonCfg = []byte(`{
	"version": 3,
	"name": "My lovely gateway",
	"port": 8080,
	"cache_ttl": 3600,
	"timeout": "3s",
	"endpoints": [
			{
					"endpoint": "/github",
					"method": "GET",
					"extra_config" : {"user":"test","hits":6,"parents":["gomez","morticia"], "nested_data": {"data": {"status": "OK"}}},
					"backend": [
							{
									"host": [
											"https://api.github.com"
									],
									"url_pattern": "/",
									"whitelist": [
											"authorizations_url",
											"code_search_url"
									],
									"extra_config" : {"user":"test","hits":6,"parents":["gomez","morticia"]}
							}
					]
			},
			{
					"endpoint": "/supu",
					"method": "GET",
					"concurrent_calls": 3,
					"backend": [
							{
									"host": [
											"http://127.0.0.1:8080"
									],
									"url_pattern": "/__debug/supu"
							}
					]
			},
			{
					"endpoint": "/combination/{id}",
					"method": "GET",
					"backend": [
							{
									"group": "first_post",
									"host": [
											"https://jsonplaceholder.typicode.com"
									],
									"url_pattern": "/posts/{id}",
									"blacklist": [
											"userId"
									]
							},
							{
									"host": [
											"https://jsonplaceholder.typicode.com"
									],
									"url_pattern": "/users/{id}",
									"mapping": {
											"email": "personal_email"
									}
							}
					]
			}
	]
}`)

	tomlCfg = []byte(`version = 3.0
name = "My lovely gateway"
port = 8080.0
cache_ttl = 3600.0
timeout = "3s"

[[endpoints]]
endpoint = "/github"
method = "GET"

[endpoints.extra_config]
user = "test"
hits = 6.0
parents = [
"gomez",
"morticia"
]

[endpoints.extra_config.nested_data.data]
status = "OK"

[[endpoints.backend]]
host = [
"https://api.github.com"
]
url_pattern = "/"
whitelist = [
"authorizations_url",
"code_search_url"
]

[endpoints.backend.extra_config]
user = "test"
hits = 6.0
parents = [
	"gomez",
	"morticia"
]

[[endpoints]]
endpoint = "/supu"
method = "GET"
concurrent_calls = 3.0

[[endpoints.backend]]
host = [
"http://127.0.0.1:8080"
]
url_pattern = "/__debug/supu"

[[endpoints]]
endpoint = "/combination/{id}"
method = "GET"

[[endpoints.backend]]
group = "first_post"
host = [
"https://jsonplaceholder.typicode.com"
]
url_pattern = "/posts/{id}"
blacklist = [
"userId"
]

[[endpoints.backend]]
host = [
"https://jsonplaceholder.typicode.com"
]
url_pattern = "/users/{id}"

[endpoints.backend.mapping]
email = "personal_email"`)

	yamlCfg = []byte(`version: 3
name: My lovely gateway
port: 8080
cache_ttl: 3600
timeout: 3s
endpoints:
  - endpoint: /github
    method: GET
    extra_config:
      user: test
      hits: 6
      parents:
        - gomez
        - morticia
      nested_data:
        data:
          status: OK
    backend:
      - host:
          - 'https://api.github.com'
        url_pattern: /
        whitelist:
          - authorizations_url
          - code_search_url
        extra_config:
          user: test
          hits: 6
          parents:
            - gomez
            - morticia
  - endpoint: /supu
    method: GET
    concurrent_calls: 3
    backend:
      - host:
          - 'http://127.0.0.1:8080'
        url_pattern: /__debug/supu
  - endpoint: '/combination/{id}'
    method: GET
    backend:
      - group: first_post
        host:
          - 'https://jsonplaceholder.typicode.com'
        url_pattern: '/posts/{id}'
        blacklist:
          - userId
      - host:
          - 'https://jsonplaceholder.typicode.com'
        url_pattern: '/users/{id}'
        mapping:
          email: personal_email
`)
)
