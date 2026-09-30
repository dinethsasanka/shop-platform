module shop-platform/order

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/segmentio/kafka-go v0.4.42
)

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
)


// The go.mod file is a configuration file for Go modules, which are used to manage dependencies in Go projects.
// run `go mod tidy` = "go download whatever libraries go.mod says I need" (like npm install).