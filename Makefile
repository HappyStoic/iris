PROTOBUF_DIR=`pwd`/pkg/messaging/pb

run-debug:
	GOLOG_LOG_LEVEL="dht=debug,p2pnetwork=debug" go run cmd/peercli.go --conf config.yaml

run:
	GOLOG_LOG_LEVEL="p2pnetwork=debug" go run cmd/peercli.go --conf config.yaml

orggensign:
	go run cmd/orggensign.go

run0:
	GOLOG_LOG_LEVEL="p2pnetwork=debug" go run cmd/peercli.go --conf config0.yaml

run1:
	GOLOG_LOG_LEVEL="p2pnetwork=debug" go run cmd/peercli.go --conf config1.yaml

run-do-something:
	DO_SOMETHING="1" GOLOG_LOG_LEVEL="p2pnetwork=debug" go run cmd/peercli.go --conf config.yaml


build:
	go build cmd/peercli.go

protobuf:
	protoc -I=$(PROTOBUF_DIR) --go_out=. $(PROTOBUF_DIR)/*.proto

network:
	docker-compose up --build --force-recreate
