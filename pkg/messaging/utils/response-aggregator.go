package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type ResponsesProcessor func(string, []proto.Message, *StorageMetadata)

type StorageMetadata struct {
	ResponsesReceiver peer.ID
}

type Storage struct {
	receivingCh chan proto.Message
	metadata    *StorageMetadata
	responses   []proto.Message
}

func NewStorage(maxResp int, metadata *StorageMetadata) *Storage {
	return &Storage{
		receivingCh: make(chan proto.Message),
		responses:   make([]proto.Message, 0, maxResp),
		metadata:    metadata,
	}
}

func (s *Storage) addResp(message proto.Message) error {
	if len(s.responses) == cap(s.responses) {
		return errors.Errorf("putting new response into storage but all responses are already received")
	}
	s.responses = append(s.responses, message)
	return nil
}

func (s *Storage) full() bool {
	return len(s.responses) == cap(s.responses)
}

func (s *Storage) getAggregatedResponses() []proto.Message {
	return s.responses
}

func (s *Storage) getMetadata() *StorageMetadata {
	return s.metadata
}

func (s *Storage) status() string {
	return fmt.Sprintf("%d/%d", len(s.responses), cap(s.responses))
}

type ResponseAggregator struct {
	responseStorage map[string]*Storage
	respProcessor   ResponsesProcessor
}

func NewResponseAggregator(respProcessor ResponsesProcessor) *ResponseAggregator {
	return &ResponseAggregator{
		responseStorage: make(map[string]*Storage),
		respProcessor:   respProcessor,
	}
}

func (rsm *ResponseAggregator) StartWaiting(ctx context.Context, id string, meta *StorageMetadata, maxResp int, timeout time.Duration) error {
	_, exists := rsm.responseStorage[id]
	if exists {
		return errors.Errorf("there is already storage for responses on request id %s", id)
	}
	log.Debugf("starting waiting for %d responses with id %s with timeout %s", maxResp, id, timeout)

	// create storage for this id
	s := NewStorage(maxResp, meta)
	rsm.responseStorage[id] = s
	go func() {
		for {
			select {
			case newMsg := <-s.receivingCh:
				err := s.addResp(newMsg)
				if err != nil {
					log.Errorf("error putting new resp into response storage: %s", err)
				}
				if s.full() {
					log.Infof("aggregated all responses in response storage with id %s", id)
					rsm.finish(id)
					return
				}

			case <-time.After(timeout):
				log.Infof("timeout elapsed waiting for the responses with storage id %s, got %s responses", id,
					s.status())
				rsm.finish(id)
				return

			case <-ctx.Done():
				return
			}

		}
	}()
	return nil
}

func (rsm *ResponseAggregator) finish(id string) {
	// get storage with responses and metadata
	s := rsm.responseStorage[id]
	// delete it, this storage is done now
	delete(rsm.responseStorage, id)
	// process all the responses
	rsm.respProcessor(id, s.getAggregatedResponses(), s.getMetadata())
}

func (rsm *ResponseAggregator) AddResponse(id string, msg proto.Message) error {
	storage, ok := rsm.responseStorage[id]
	if !ok {
		return errors.Errorf("trying to put response to non-existing storage with ID %s", id)
	}
	if storage.full() {
		return errors.Errorf("tried to put new response into already full storage with id %s", id)
	}
	storage.receivingCh <- msg
	return nil
}
