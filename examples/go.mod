module github.com/libi/dcron/examples

go 1.21

toolchain go1.22.0

replace (
	github.com/libi/dcron v0.0.0 => ../
	github.com/libi/dcron/commons v0.0.0 => ../commons
	github.com/libi/dcron/driver/etcddriver v0.0.0 => ../driver/etcddriver
	github.com/libi/dcron/driver/redisdriver v0.0.0 => ../driver/redisdriver
	github.com/libi/dcron/driver/rediszsetdriver v0.0.0 => ../driver/rediszsetdriver
)

require (
	github.com/google/uuid v1.5.0
	github.com/libi/dcron v0.0.0
	github.com/libi/dcron/commons v0.0.0
	github.com/libi/dcron/driver/redisdriver v0.0.0
	github.com/redis/go-redis/v9 v9.3.1
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)
