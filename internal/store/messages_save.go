package store

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func (s *MessageStore) SaveLiveMessage(ctx context.Context, evt *events.Message) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := s.SaveMessageTx(ctx, tx, evt); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MessageStore) SaveMessageTx(ctx context.Context, tx execer, evt *events.Message) (bool, error) {
	return s.SaveMessageTxWithStar(ctx, tx, evt, false)
}

func (s *MessageStore) SaveMessageTxWithStar(ctx context.Context, tx execer, evt *events.Message, starred bool, initialStatus ...int) (bool, error) {
	if evt.Info.Chat.Server == "broadcast" {
		return false, nil
	}

	chatJID := evt.Info.Chat.String()
	senderJID := evt.Info.Sender.String()
	ts := evt.Info.Timestamp.Unix()
	msgID := evt.Info.ID

	if strings.HasSuffix(chatJID, "@s.whatsapp.net") && s.IsSelfJID(ctx, chatJID) {
		chatJID = s.GetCanonicalSelfJID(ctx)
	}
	if strings.HasSuffix(chatJID, "@lid") {
		if pn := s.ResolveLIDtoPN(ctx, chatJID); pn != "" {
			chatJID = pn
		}
	}

	msg := evt.Message
	if ephem := msg.GetEphemeralMessage(); ephem != nil && ephem.Message != nil {
		msg = ephem.Message
	}
	if vo := msg.GetViewOnceMessage(); vo != nil && vo.Message != nil {
		msg = vo.Message
	}
	if vo2 := msg.GetViewOnceMessageV2(); vo2 != nil && vo2.Message != nil {
		msg = vo2.Message
	}
	if vo2Ext := msg.GetViewOnceMessageV2Extension(); vo2Ext != nil && vo2Ext.Message != nil {
		msg = vo2Ext.Message
	}

	if protoMsg := msg.GetProtocolMessage(); protoMsg != nil {
		return s.handleProtocolMessage(ctx, tx, chatJID, protoMsg)
	}

	kind, text, ok := DescribeMessage(msg)
	if !ok {
		return false, nil
	}

	raw, _ := proto.Marshal(msg)
	starInt := 0
	if starred {
		starInt = 1
	}

	st := MessageStatusNone
	if evt.Info.IsFromMe {
		st = MessageStatusSent
	}
	if len(initialStatus) > 0 {
		st = initialStatus[0]
	}

	return s.insertOrUpdateMessage(ctx, tx, chatJID, msgID, senderJID, evt.Info.PushName, evt.Info.IsFromMe, ts, kind, text, raw, starInt, st)
}
