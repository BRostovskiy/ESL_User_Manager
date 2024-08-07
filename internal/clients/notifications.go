package clients

import (
	"context"

	"github.com/sirupsen/logrus"
)

type (
	channelName        string
	ChannelNotificator interface {
		Notify(ctx context.Context, channelName channelName, msg string) error
	}

	ChannelNotificationSvc struct {
		ChannelNotificator
		logger *logrus.Logger
	}
)

var (
	ChannelCreate channelName = "create"
	ChannelUpdate channelName = "update"
	ChannelDelete channelName = "delete"
)

func NewChannelNotificationSvc(l *logrus.Logger) *ChannelNotificationSvc {
	return &ChannelNotificationSvc{logger: l}
}

func (n *ChannelNotificationSvc) Notify(_ context.Context, channelName channelName, msg string) error {
	n.logger.Debugf("send message: %s, to channel: %s", msg, channelName)
	return nil
}
