PROTOBUF_DIR=`pwd`/pkg/messaging/pb

run-dht-debug:
	GOLOG_LOG_LEVEL="dht=debug,iris=debug" go run cmd/peercli.go --conf config.yaml

run:
	GOLOG_LOG_LEVEL="iris=debug" go run cmd/peercli.go --conf config.yaml

orgsig:
	go build cmd/orgsig.go

run-orgsig: orgsig
	go run cmd/orgsig.go


run0:
	GOLOG_LOG_LEVEL="iris=debug" go run cmd/peercli.go --conf config0.yaml

run1:
	GOLOG_LOG_LEVEL="iris=debug" go run cmd/peercli.go --conf config1.yaml

build:
	go build cmd/peercli.go

protobuf:
	protoc -I=$(PROTOBUF_DIR) --go_out=. $(PROTOBUF_DIR)/*.proto

network:
	docker compose up --build --force-recreate
