package clients

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"

	"happystoic/p2pnetwork/pkg/config"
)

var log = logging.Logger("p2pnetwork")

type Callback func(data []byte)

type RedisClient struct {
	*redis.Client

	ctx                   context.Context
	channel               string
	messageTypesCallbacks map[string]Callback
}

type RedisBaseMessage struct {
	Type    string      `json:"type"`
	Version uint        `json:"version"`
	Data    interface{} `json:"data"`
}

func NewRedisClient(conf *config.Redis, ctx context.Context) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Addr(),
		Username: conf.Username,
		Password: conf.Password,
		DB:       conf.Db,
	})
	// check connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	rc := &RedisClient{
		Client:                rdb,
		ctx:                   ctx,
		channel:               conf.Tl2NlChannel,
		messageTypesCallbacks: make(map[string]Callback),
	}
	rc.subscribeChannel(conf.Tl2NlChannel)
	return rc, nil
}

func (rc *RedisClient) SubscribeCallback(messageType string, callback Callback) error {
	if _, exists := rc.messageTypesCallbacks[messageType]; exists {
		return errors.Errorf("callback with messageType %s already exists", messageType)
	}
	rc.messageTypesCallbacks[messageType] = callback
	return nil
}

func (rc *RedisClient) subscribeChannel(channel string) {
	pubSub := rc.Subscribe(rc.ctx, channel)
	wg := sync.WaitGroup{}

	// Go channel which receives messages.
	ch := pubSub.Channel()
	go func() {
		// Consume messages.
		for redisMsg := range ch {
			baseMsg := RedisBaseMessage{}
			err := json.Unmarshal([]byte(redisMsg.Payload), &baseMsg)
			if err != nil {
				log.Errorf("error while unmarshalling json RedisBaseMessage: %s", err)
				continue
			}

			// this is my message sent to TL, do not deal with it
			if strings.HasPrefix(baseMsg.Type, "nl2tl") {
				continue
			}

			callback, exists := rc.messageTypesCallbacks[baseMsg.Type]
			if exists {
				log.Debugf("received RedisBaseMessage of type %s, calling its callback...", baseMsg.Type)

				// todo make nicer if time
				bytesData, _ := json.Marshal(baseMsg.Data)
				go func() {
					wg.Add(1)
					callback(bytesData)
					wg.Done()
				}()
			} else {
				log.Errorf("received unknown RedisBaseMessage type '%s' from TL", baseMsg.Type)
			}
		}
		wg.Wait()
	}()
}

func (rc *RedisClient) PublishMessage(msgType string, data interface{}) error {
	baseMsg := RedisBaseMessage{
		Type:    msgType,
		Version: 1,
		Data:    data,
	}
	encoded, err := json.Marshal(baseMsg)
	if err != nil {
		return err
	}
	return rc.Publish(rc.ctx, rc.channel, encoded).Err()
}
