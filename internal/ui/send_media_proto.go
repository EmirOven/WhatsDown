package ui

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func buildMediaProto(kind, caption, filename, mimeType string, upload *whatsmeow.UploadResponse, fileLen uint64, ctx *waE2E.ContextInfo) *waE2E.Message {
	var waMsg waE2E.Message
	switch kind {
	case store.MessageKindImage:
		waMsg.ImageMessage = &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           &upload.URL,
			DirectPath:    &upload.DirectPath,
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    &fileLen,
			ContextInfo:   ctx,
		}
	case store.MessageKindVideo:
		waMsg.VideoMessage = &waE2E.VideoMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           &upload.URL,
			DirectPath:    &upload.DirectPath,
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    &fileLen,
			ContextInfo:   ctx,
		}
	case store.MessageKindAudio:
		waMsg.AudioMessage = &waE2E.AudioMessage{
			Mimetype:      proto.String(mimeType),
			URL:           &upload.URL,
			DirectPath:    &upload.DirectPath,
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    &fileLen,
			ContextInfo:   ctx,
		}
	default:
		waMsg.DocumentMessage = &waE2E.DocumentMessage{
			Caption:       proto.String(caption),
			FileName:      proto.String(filename),
			Mimetype:      proto.String(mimeType),
			URL:           &upload.URL,
			DirectPath:    &upload.DirectPath,
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    &fileLen,
			ContextInfo:   ctx,
		}
	}
	return &waMsg
}
