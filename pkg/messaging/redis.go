package messaging

import (
	"context"
	"github.com/go-redis/redis/v8"
	"happystoic/p2pnetwork/pkg/config"
)

type RedisClient struct {
	*redis.Client

	ctx      context.Context
	channels *config.RedisChannels
}

func NewRedisClient(conf *config.Redis, ctx context.Context) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Addr(),
		Username: conf.Username,
		Password: conf.Password,
		DB:       conf.Db,
	})
	// check connection
	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}
	return &RedisClient{rdb, ctx, &conf.Channels}, nil
}

func (rc *RedisClient) checkConnection() {

}

type Callback func(payload string)

func (rc *RedisClient) subscribeCallback(channel string, callback Callback) {
	pubSub := rc.Subscribe(rc.ctx, channel)

	// Go channel which receives messages.
	ch := pubSub.Channel()

	go func() {
		// Consume messages.
		for msg := range ch {
			callback(msg.Payload)
		}
	}()
}

func (rc *RedisClient) publishMessage(channel string, message interface{}) error {
	return rc.Publish(rc.ctx, channel, message).Err()
}
