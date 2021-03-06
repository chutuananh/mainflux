package lora

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/mainflux/mainflux/broker"
)

const (
	protocol      = "lora"
	thingSuffix   = "thing"
	channelSuffix = "channel"
	contentType   = "application/json"
)

var (
	// ErrMalformedIdentity indicates malformed identity received (e.g.
	// invalid appID or deviceEUI).
	ErrMalformedIdentity = errors.New("malformed identity received")

	// ErrMalformedMessage indicates malformed LoRa message.
	ErrMalformedMessage = errors.New("malformed message received")

	// ErrNotFoundDev indicates a non-existent route map for a device EUI.
	ErrNotFoundDev = errors.New("route map not found for this device EUI")

	// ErrNotFoundApp indicates a non-existent route map for an application ID.
	ErrNotFoundApp = errors.New("route map not found for this application ID")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThing creates thingID:devEUI route-map
	CreateThing(thingID string, devEUI string) error

	// UpdateThing updates thingID:devEUI route-map
	UpdateThing(thingID string, devEUI string) error

	// RemoveThing removes thingID:devEUI route-map
	RemoveThing(thingID string) error

	// CreateChannel creates channelID:appID route-map
	CreateChannel(chanID string, appID string) error

	// UpdateChannel updates channelID:appID route-map
	UpdateChannel(chanID string, appID string) error

	// RemoveChannel removes channelID:appID route-map
	RemoveChannel(chanID string) error

	// Publish forwards messages from the LoRa MQTT broker to Mainflux NATS broker
	Publish(ctx context.Context, token string, msg Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	broker     broker.Nats
	thingsRM   RouteMapRepository
	channelsRM RouteMapRepository
}

// New instantiates the LoRa adapter implementation.
func New(broker broker.Nats, thingsRM, channelsRM RouteMapRepository) Service {
	return &adapterService{
		broker:     broker,
		thingsRM:   thingsRM,
		channelsRM: channelsRM,
	}
}

// Publish forwards messages from Lora MQTT broker to Mainflux NATS broker
func (as *adapterService) Publish(ctx context.Context, token string, m Message) error {
	// Get route map of lora application
	thing, err := as.thingsRM.Get(m.DevEUI)
	if err != nil {
		return ErrNotFoundDev
	}

	// Get route map of lora application
	channel, err := as.channelsRM.Get(m.ApplicationID)
	if err != nil {
		return ErrNotFoundApp
	}

	// Use the SenML message decoded on LoRa server application if
	// field Object isn't empty. Otherwise, decode standard field Data.
	var payload []byte
	switch m.Object {
	case nil:
		payload, err = base64.StdEncoding.DecodeString(m.Data)
		if err != nil {
			return ErrMalformedMessage
		}
	default:
		jo, err := json.Marshal(m.Object)
		if err != nil {
			return err
		}
		payload = []byte(jo)
	}

	created, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return err
	}

	// Publish on Mainflux NATS broker
	msg := broker.Message{
		Publisher:   thing,
		Protocol:    protocol,
		ContentType: contentType,
		Channel:     channel,
		Payload:     payload,
		Created:     created,
	}

	return as.broker.Publish(ctx, token, msg)
}

func (as *adapterService) CreateThing(thingID string, devEUI string) error {
	return as.thingsRM.Save(thingID, devEUI)
}

func (as *adapterService) UpdateThing(thingID string, devEUI string) error {
	return as.thingsRM.Save(thingID, devEUI)
}

func (as *adapterService) RemoveThing(thingID string) error {
	return as.thingsRM.Remove(thingID)
}

func (as *adapterService) CreateChannel(chanID string, appID string) error {
	return as.channelsRM.Save(chanID, appID)
}

func (as *adapterService) UpdateChannel(chanID string, appID string) error {
	return as.channelsRM.Save(chanID, appID)
}

func (as *adapterService) RemoveChannel(chanID string) error {
	return as.channelsRM.Remove(chanID)
}
