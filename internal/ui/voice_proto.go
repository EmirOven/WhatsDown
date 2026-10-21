package ui

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func buildVoiceProto(upload *whatsmeow.UploadResponse, fileLen uint64, seconds uint32, waveform []byte, ctx *waE2E.ContextInfo) *waE2E.Message {
	return &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			Mimetype:      proto.String("audio/ogg; codecs=opus"),
			URL:           &upload.URL,
			DirectPath:    &upload.DirectPath,
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    &fileLen,
			Seconds:       &seconds,
			PTT:           proto.Bool(true),
			Waveform:      waveform,
			ContextInfo:   ctx,
		},
	}
}
