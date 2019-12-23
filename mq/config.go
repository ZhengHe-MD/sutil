// Copyright 2014 The mqrouter Author. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mq

import (
	"context"
	"errors"
	"fmt"
	"github.com/shawnfeng/sutil/sconf/center"
	"github.com/shawnfeng/sutil/scontext"
	"github.com/shawnfeng/sutil/slog/slog"
	"strings"
	"sync"
	"time"
)

type MQType int

const (
	MQTypeKafka MQType = iota
)

func (t MQType) String() string {
	switch t {
	case MQTypeKafka:
		return "kafka"
	default:
		return ""
	}
}

type ConfigerType int

const (
	ConfigerTypeSimple ConfigerType = iota
	ConfigerTypeEtcd
	ConfigerTypeApollo
)

func (c ConfigerType) String() string {
	switch c {
	case ConfigerTypeSimple:
		return "simple"
	case ConfigerTypeEtcd:
		return "etcd"
	case ConfigerTypeApollo:
		return "apollo"
	default:
		return "unknown"
	}
}

const (
	defaultTimeout = 3 * time.Second
)

type Config struct {
	MQType         MQType
	MQAddr         []string
	Topic          string
	TimeOut        time.Duration
	CommitInterval time.Duration
	Offset         int64
	OffsetAt       string
}

type KeyParts struct {
	Topic string
	Group string
}

var DefaultConfiger Configer

type Configer interface {
	Init(ctx context.Context) error
	GetConfig(ctx context.Context, topic string) (*Config, error)
	ParseKey(ctx context.Context, k string) (*KeyParts, error)
	Watch(ctx context.Context) <-chan *center.ChangeEvent
}

func NewConfiger(configType ConfigerType) (Configer, error) {
	switch configType {
	case ConfigerTypeSimple:
		return NewSimpleConfiger(), nil
	case ConfigerTypeEtcd:
		return NewEtcdConfiger(), nil
	case ConfigerTypeApollo:
		return NewApolloConfiger(), nil
	default:
		return nil, fmt.Errorf("configType %d error", configType)
	}
}

type SimpleConfig struct {
	mqAddr []string
}

func NewSimpleConfiger() *SimpleConfig {
	return &SimpleConfig{
		mqAddr: []string{"prod.kafka1.ibanyu.com:9092", "prod.kafka2.ibanyu.com:9092", "prod.kafka3.ibanyu.com:9092"},
	}
}

func (m *SimpleConfig) Init(ctx context.Context) error {
	fun := "SimpleConfig.Init-->"
	slog.Infof(ctx, "%s start", fun)
	// noop
	return nil
}

func (m *SimpleConfig) GetConfig(ctx context.Context, topic string) (*Config, error) {
	fun := "SimpleConfig.GetConfig-->"
	slog.Infof(ctx, "%s get simple config topic:%s", fun, topic)

	return &Config{
		MQType:         MQTypeKafka,
		MQAddr:         m.mqAddr,
		Topic:          topic,
		TimeOut:        defaultTimeout,
		CommitInterval: 1 * time.Second,
		Offset:         FirstOffset,
	}, nil
}

func (m *SimpleConfig) ParseKey(ctx context.Context, k string) (*KeyParts, error) {
	fun := "SimpleConfig.ParseKey-->"
	return nil, fmt.Errorf("%s not implemented", fun)
}

func (m *SimpleConfig) Watch(ctx context.Context) <-chan *center.ChangeEvent {
	fun := "SimpleConfig.Watch-->"
	slog.Infof(ctx, "%s start", fun)
	// noop
	return nil
}

type EtcdConfig struct {
	etcdAddr []string
}

func NewEtcdConfiger() *EtcdConfig {
	return &EtcdConfig{
		etcdAddr: []string{}, //todo
	}
}

func (m *EtcdConfig) Init(ctx context.Context) error {
	fun := "EtcdConfig.Init-->"
	slog.Infof(ctx, "%s start", fun)
	// TODO
	return nil
}

func (m *EtcdConfig) GetConfig(ctx context.Context, topic string) (*Config, error) {
	fun := "EtcdConfig.GetConfig-->"
	slog.Infof(ctx, "%s get etcd config topic:%s", fun, topic)
	// TODO
	return nil, fmt.Errorf("%s etcd config not supported", fun)
}

func (m *EtcdConfig) ParseKey(ctx context.Context, k string) (*KeyParts, error) {
	fun := "EtcdConfig.ParseKey-->"
	return nil, fmt.Errorf("%s not implemented", fun)
}

func (m *EtcdConfig) Watch(ctx context.Context) <-chan *center.ChangeEvent {
	fun := "EtcdConfig.Watch-->"
	slog.Infof(ctx, "%s start", fun)
	// TODO:
	return nil
}

const (
	apolloConfigSep   = "."
	apolloBrokersSep  = ","
	apolloBrokersKey  = "brokers"
	apolloOffsetAtKey = "offsetat"
)

type ApolloConfig struct {
	watchOnce sync.Once
	ch        chan *center.ChangeEvent
	center    center.ConfigCenter
}

func NewApolloConfiger() *ApolloConfig {
	return &ApolloConfig{
		ch: make(chan *center.ChangeEvent),
	}
}

func (m *ApolloConfig) Init(ctx context.Context) (err error) {
	fun := "ApolloConfig.Init-->"
	slog.Infof(ctx, "%s start", fun)
	apolloCenter, err := center.NewConfigCenter(center.ApolloConfigCenter)
	if err != nil {
		slog.Errorf(ctx, "%s create config center err:%v", fun, err)
	}

	err = apolloCenter.Init(ctx, center.DefaultApolloMiddlewareService, []string{center.DefaultApolloMQNamespace})
	if err != nil {
		slog.Errorf(ctx, "%s init config center err:%v", fun, err)
	}

	m.center = apolloCenter
	return
}

type simpleContextControlRouter struct {
	group string
}

func (s simpleContextControlRouter) GetControlRouteGroup() (string, bool) {
	return s.group, true
}

func (s simpleContextControlRouter) SetControlRouteGroup(group string) error {
	s.group = group
	return nil
}

func (m *ApolloConfig) getConfigItemWithFallback(ctx context.Context, topic string, name string) (string, bool) {
	val, ok := m.center.GetStringWithNamespace(ctx, center.DefaultApolloMQNamespace, m.buildKey(ctx, topic, name))
	if !ok {
		defaultCtx := context.WithValue(ctx, scontext.ContextKeyControl, simpleContextControlRouter{defaultRouteGroup})
		val, ok = m.center.GetStringWithNamespace(defaultCtx, center.DefaultApolloMQNamespace, m.buildKey(defaultCtx, topic, name))
	}
	return val, ok
}

func (m *ApolloConfig) GetConfig(ctx context.Context, topic string) (*Config, error) {
	fun := "ApolloConfig.GetConfig-->"
	slog.Infof(ctx, "%s get mq config topic:%s", fun, topic)

	brokersVal, ok := m.getConfigItemWithFallback(ctx, topic, apolloBrokersKey)
	if !ok {
		return nil, fmt.Errorf("%s no brokers config found", fun)
	}

	var brokers []string
	for _, broker := range strings.Split(brokersVal, apolloBrokersSep) {
		if broker != "" {
			brokers = append(brokers, strings.TrimSpace(broker))
		}
	}

	slog.Infof(ctx, "%s got config brokers:%s", fun, brokers)

	offsetAtVal, ok := m.getConfigItemWithFallback(ctx, topic, apolloOffsetAtKey)
	if !ok {
		slog.Infof(ctx, "%s no offsetAtVal config founds", fun)

	}
	slog.Infof(ctx, "%s got config offsetAt:%s", fun, offsetAtVal)

	return &Config{
		MQType:         MQTypeKafka,
		MQAddr:         brokers,
		Topic:          topic,
		TimeOut:        defaultTimeout,
		CommitInterval: 1 * time.Second,
		Offset:         FirstOffset,
		OffsetAt:       offsetAtVal,
	}, nil
}

func (m *ApolloConfig) ParseKey(ctx context.Context, key string) (*KeyParts, error) {
	fun := "ApolloConfig.ParseKey-->"
	parts := strings.Split(key, apolloConfigSep)
	numParts := len(parts)
	if numParts < 4 {
		errMsg := fmt.Sprintf("%s invalid key:%s", fun, key)
		slog.Errorln(ctx, errMsg)
		return nil, errors.New(errMsg)
	}

	return &KeyParts{
		Topic: strings.Join(parts[:numParts-3], apolloConfigSep),
		Group: parts[numParts-3],
	}, nil
}

type apolloObserver struct {
	ch chan<- *center.ChangeEvent
}

func (ob *apolloObserver) HandleChangeEvent(event *center.ChangeEvent) {
	if event.Namespace != center.DefaultApolloMQNamespace {
		return
	}

	// TODO: filter different mq types
	var changes = map[string]*center.Change{}
	for k, ce := range event.Changes {
		if strings.Contains(k, fmt.Sprint(MQTypeKafka)) {
			changes[k] = ce
		}
	}

	event.Changes = changes

	ob.ch <- event
}

func (m *ApolloConfig) Watch(ctx context.Context) <-chan *center.ChangeEvent {
	fun := "ApolloConfig.Watch-->"
	m.watchOnce.Do(func() {
		slog.Infof(ctx, "%s start", fun)
		m.center.StartWatchUpdate(ctx)
		m.center.RegisterObserver(ctx, &apolloObserver{m.ch})
	})
	return m.ch
}

func (m *ApolloConfig) buildKey(ctx context.Context, topic, item string) string {
	return strings.Join([]string{
		topic,
		scontext.GetControlRouteGroupWithDefault(ctx, defaultRouteGroup),
		fmt.Sprint(MQTypeKafka),
		item,
	}, apolloConfigSep)
}
