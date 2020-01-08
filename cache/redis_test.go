package cache

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var client, _ = NewCommonRedis("test/test", 1024)

func TestTtl(t *testing.T) {
	key := "aaa"
	val := "bbb"
	exp := 2 * time.Hour
	setcmd := client.Set(key, val, exp)
	assert.True(t, setcmd.Err() == nil)
	ttl := client.Ttl(key)
	assert.True(t, ttl.Val() > time.Hour && ttl.Val() <= exp)

	expire := client.Expire(key, time.Hour)
	assert.True(t, expire.Val())
	ttl = client.Ttl(key)
	assert.True(t, ttl.Val() > 0 && ttl.Val() <= time.Hour)
}

func TestGet(t *testing.T) {
	key := "aaa"
	ttl := client.Ttl(key)
	t.Log(ttl.Val().Seconds())
	get := client.Get(key)
	t.Log(get.Val())
}
