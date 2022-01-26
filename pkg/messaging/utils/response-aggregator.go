package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type ResponsesProcessor func(string, []proto.Message)

type Storage struct {
	receivingCh chan proto.Message
	responses   []proto.Message
}

func NewStorage(maxResp int) *Storage {
	return &Storage{
		receivingCh: make(chan proto.Message),
		responses:   make([]proto.Message, 0, maxResp),
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

func (s *Storage) aggregatedResponses() []proto.Message {
	return s.responses
}

func (s *Storage) status() string {
	return fmt.Sprintf("%d/%d", len(s.responses), cap(s.responses))
}

type RespStorageManager struct {
	storageMap    map[string]*Storage
	respProcessor ResponsesProcessor
}

func NewResponseStorage(respProc ResponsesProcessor) *RespStorageManager {
	return &RespStorageManager{
		storageMap:    make(map[string]*Storage),
		respProcessor: respProc,
	}
}

func (rsm *RespStorageManager) StartWaiting(ctx context.Context, id string, maxResp int, ttl time.Duration) error {
	_, exists := rsm.storageMap[id]
	if exists {
		return errors.Errorf("there is already storage for responses on request with id %s", id)
	}

	// create storage for this id
	s := NewStorage(maxResp)
	rsm.storageMap[id] = s
	go func() {
		for {
			select {
			case newMsg := <-s.receivingCh:
				err := s.addResp(newMsg)
				if err != nil {
					log.Errorf("error putting new resp into response storage: %s", err)
				}
				if s.full() {
					rsm.finish(id)
					return
				}

			case <-time.After(ttl):
				log.Infof("Timout elapsed waiting for the responses with storage id %s, got %s responses", id,
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

func (rsm *RespStorageManager) finish(id string) {
	// get storage with responses
	s := rsm.storageMap[id]
	// delete it, this storage is done now
	delete(rsm.storageMap, id)
	// process all the responses
	rsm.respProcessor(id, s.aggregatedResponses())
}

func (rsm *RespStorageManager) AddResponse(id string, msg proto.Message) error {
	storage, ok := rsm.storageMap[id]
	if !ok {
		return errors.Errorf("trying to put response to non-existing storage with ID %s", id)
	}
	storage.receivingCh <- msg
	return nil
}
