module github.com/nickfang/personal-dashboard/services/dashboard-api

go 1.25.6

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/joho/godotenv v1.5.1
	github.com/nickfang/personal-dashboard/services/gen/go v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.79.0
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/nickfang/personal-dashboard/services/gen/go => ../gen/go
