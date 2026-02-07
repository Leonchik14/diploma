.PHONY: proto
proto:
	@echo "Generating proto files..."
	@mkdir -p proto/gen/go
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto
