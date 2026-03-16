package mqtt

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"telemetry-system/internal/config"
	"telemetry-system/internal/domain"
)

type MQTTClient struct {
	client      mqtt.Client
	logger      *zap.Logger
	messageCh   chan *domain.Measurement
	msgHandler  mqtt.MessageHandler
}

func NewMQTTClient(cfg *config.MQTTConfig, logger *zap.Logger) (*MQTTClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(cfg.ClientID)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(time.Duration(cfg.ReconnectRetry) * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)

	client := &MQTTClient{
		logger:    logger,
		messageCh: make(chan *domain.Measurement, 1000),
	}

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		logger.Info("MQTT connected successfully")
		for _, topic := range cfg.Topics {
			token := c.Subscribe(topic, byte(cfg.QoS), client.msgHandler)
			token.Wait()
			if token.Error() != nil {
				logger.Error("Failed to subscribe to topic", 
					zap.String("topic", topic), 
					zap.Error(token.Error()))
			} else {
				logger.Info("Subscribed to topic", zap.String("topic", topic))
			}
		}
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		logger.Warn("MQTT connection lost", zap.Error(err))
	})

	client.client = mqtt.NewClient(opts)
	return client, nil
}

func (m *MQTTClient) SetMessageHandler(handler func(*domain.Measurement)) {
	m.msgHandler = func(client mqtt.Client, msg mqtt.Message) {
		m.logger.Debug("Received MQTT message",
			zap.String("topic", msg.Topic()),
			zap.ByteString("payload", msg.Payload()))

		var data domain.Measurement
		if err := json.Unmarshal(msg.Payload(), &data); err != nil {
			m.logger.Error("Failed to unmarshal MQTT message", zap.Error(err))
			return
		}

		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}

		// 🔥 ОТПРАВЛЯЕМ В КАНАЛ для ingestor
		select {
		case m.messageCh <- &data:
			m.logger.Debug("Message sent to ingestor",
				zap.String("sensor_id", data.SensorID))
		default:
			m.logger.Warn("Message channel full, dropping message")
		}
	}
}

func (m *MQTTClient) Connect() error {
	token := m.client.Connect()
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	return nil
}

func (m *MQTTClient) GetMessageChannel() <-chan *domain.Measurement {
	return m.messageCh
}

func (m *MQTTClient) Disconnect() {
	m.client.Disconnect(250)
	m.logger.Info("MQTT client disconnected")
}