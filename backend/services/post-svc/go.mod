module github.com/usedcvnt/microtwitter/post-svc

go 1.25.0

require (
	github.com/usedcvnt/microtwitter/gen/go v0.0.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

replace github.com/usedcvnt/microtwitter/gen/go => ../../gen/go
