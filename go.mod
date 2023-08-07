module github.com/NorthfieldIT/yaml2confluence

go 1.18

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/aybabtme/orderedjson v0.1.0
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/fatih/color v1.15.0
	github.com/flant/libjq-go v1.6.2
	github.com/hoisie/mustache v0.0.0-20160804235033-6375acf62c69
	github.com/mattn/go-colorable v0.1.13
	github.com/mikefarah/yq/v4 v4.34.2
	github.com/nwidger/jsoncolor v0.3.1
	github.com/thanhpk/randstr v1.0.4
	gopkg.in/op/go-logging.v1 v1.0.0-20160211212156-b2cb9fa56473
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/a8m/envsubst v1.4.2 // indirect
	github.com/alecthomas/participle/v2 v2.0.0 // indirect
	github.com/aybabtme/flatjson v0.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/elliotchance/orderedmap v1.5.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.11.0 // indirect
	github.com/jinzhu/copier v0.3.5 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/term v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

// https://github.com/go-yaml/yaml/pull/690
replace gopkg.in/yaml.v3 => github.com/felixfontein/yaml v0.0.0-20210209202929-35d69a41298b
