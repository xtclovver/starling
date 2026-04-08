module github.com/usedcvnt/microtwitter/user-svc

go 1.25.0

require (
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/jackc/pgx/v5 v5.7.5
	github.com/redis/go-redis/v9 v9.7.3
	github.com/usedcvnt/microtwitter/gen/go v0.0.0
	golang.org/x/crypto v0.48.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)

replace github.com/usedcvnt/microtwitter/gen/go => ../../gen/go
