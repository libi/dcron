module github.com/libi/dcron/examples

go 1.19

replace github.com/libi/dcron v0.0.0 => ../

require (
	github.com/dcron-contrib/commons v0.0.2
	github.com/dcron-contrib/redisdriver v0.0.0-20240830125937-ca446326fbd7
	github.com/google/uuid v1.6.0
	github.com/libi/dcron v0.0.0
	github.com/redis/go-redis/v9 v9.3.1
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)
