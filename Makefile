.PHONY: proto

proto:
	./generate_protos.sh

swag-gen:
	swag init -g internal/http/languages.go -o docs --parseDependency --parseInternal
