module github.com/codeshop-platform/auth-service

go 1.22.2

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/joho/godotenv v1.5.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/redis/go-redis/v9 v9.5.1
	golang.org/x/crypto v0.24.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
)

// NOTE: this environment's network egress does not allow the golang.org /
// gopkg.in vanity import hosts, only github.com. The replace directives
// below point transitive "golang.org/x/*", "google.golang.org/*" and
// "gopkg.in/*" dependencies to their official GitHub mirrors so that
// `go mod download` / `go build` work without access to proxy.golang.org.
// These are safe, functionally-identical mirrors of the same tagged
// releases and can be removed if the target build environment has normal
// access to the Go module proxy.
replace (
	go.uber.org/goleak => github.com/uber-go/goleak v1.3.0
	golang.org/x/crypto => github.com/golang/crypto v0.24.0
	golang.org/x/net => github.com/golang/net v0.26.0
	golang.org/x/sync => github.com/golang/sync v0.7.0
	golang.org/x/sys => github.com/golang/sys v0.21.0
	golang.org/x/text => github.com/golang/text v0.16.0
	google.golang.org/genproto/googleapis/rpc => github.com/googleapis/go-genproto/googleapis/rpc v0.0.0-20240624140628-dc46fd24d27d
	google.golang.org/grpc => github.com/grpc/grpc-go v1.65.0
	google.golang.org/protobuf => github.com/protocolbuffers/protobuf-go v1.34.2
	gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20200902074654-038fdea0a05b
	gopkg.in/yaml.v3 => github.com/go-yaml/yaml/v3 v3.0.1
)
