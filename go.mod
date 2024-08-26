module github.com/libi/dcron

go 1.19

require (
	github.com/libi/dcron/commons v0.6.0-dev.2
	github.com/stretchr/testify v1.9.0
)

replace github.com/libi/dcron/commons v0.6.0-dev.2 => ./commons

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
