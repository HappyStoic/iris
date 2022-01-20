PROTOBUF_DIR=`pwd`/pkg/messaging/p2p/pb

run:
	GOLOG_LOG_LEVEL="p2pnetwork=debug" go run cmd/peercli.go --conf config.yaml

run-do-something:
	DO_SOMETHING="1" GOLOG_LOG_LEVEL="p2pnetwork=debug" go run cmd/peercli.go --conf config.yaml


build:
	go build cmd/peercli.go

protobuf:
	protoc -I=$(PROTOBUF_DIR) --go_out=. $(PROTOBUF_DIR)/*.proto

network:
	docker-compose up --build --force-recreate