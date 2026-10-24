package media

/*
#cgo pkg-config: gstreamer-1.0
#include <gst/gst.h>
#include <stdlib.h>

static GstElement* wd_audio_new(const char *uri, const char *sink_name) {
 gst_init(NULL, NULL);
 GstElement *p = gst_element_factory_make("playbin", NULL);
 if (!p) return NULL;
 // scaletempo consumes rate-change segments and preserves the original pitch.
 GstElement *tempo = gst_element_factory_make("scaletempo", NULL);
 if (!tempo) { gst_object_unref(p); return NULL; }
 g_object_set(p, "uri", uri, "audio-filter", tempo, NULL);
 GstElement *video = gst_element_factory_make("fakesink", NULL);
 if (video) g_object_set(p, "video-sink", video, NULL);
 if (sink_name && sink_name[0]) {
  GstElement *sink = gst_element_factory_make(sink_name, NULL);
  if (sink) { g_object_set(sink, "sync", TRUE, NULL); g_object_set(p, "audio-sink", sink, NULL); }
 }
 gst_element_set_state(p, GST_STATE_PAUSED);
 return p;
}
static gint64 wd_audio_position(GstElement *p) {
 gint64 n = 0; gst_element_query_position(p, GST_FORMAT_TIME, &n); return n > 0 ? n : 0;
}
static gint64 wd_audio_duration(GstElement *p) {
 gint64 n = 0; gst_element_query_duration(p, GST_FORMAT_TIME, &n); return n > 0 ? n : 0;
}
static int wd_audio_seek(GstElement *p, double rate, gint64 pos) {
 return gst_element_seek(p, rate, GST_FORMAT_TIME, GST_SEEK_FLAG_FLUSH | GST_SEEK_FLAG_ACCURATE,
  GST_SEEK_TYPE_SET, pos, GST_SEEK_TYPE_NONE, GST_CLOCK_TIME_NONE);
}
static char* wd_audio_event(GstElement *p, int *ended) {
 GstBus *bus = gst_element_get_bus(p);
 GstMessage *msg;
 char *error = NULL;
 while ((msg = gst_bus_pop_filtered(bus, GST_MESSAGE_ERROR | GST_MESSAGE_EOS))) {
  if (GST_MESSAGE_TYPE(msg) == GST_MESSAGE_EOS) { *ended = 1; }
  else {
   GError *err = NULL; gst_message_parse_error(msg, &err, NULL);
   if (!error && err) error = g_strdup(err->message);
   if (err) g_error_free(err);
  }
  gst_message_unref(msg);
 }
 gst_object_unref(bus);
 return error;
}
static void wd_audio_close(GstElement *p) {
 gst_element_set_state(p, GST_STATE_NULL); gst_object_unref(p);
}
*/
import "C"

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

// AudioPlayer uses actual GStreamer timestamps and seeks, including rate changes.
// Methods may be called from the UI while another widget stops playback.
type AudioPlayer struct {
	mu       sync.Mutex
	pipeline *C.GstElement
	playing  bool
	rate     float64
}

type AudioState struct {
	Position, Duration time.Duration
	Playing, Ended     bool
	Err                error
}

func NewAudioPlayer(path string) (*AudioPlayer, error) { return newAudioPlayer(path, "") }
func newAudioPlayer(path, sink string) (*AudioPlayer, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uri := C.CString((&url.URL{Scheme: "file", Path: abs}).String())
	defer C.free(unsafe.Pointer(uri))
	sinkName := C.CString(sink)
	defer C.free(unsafe.Pointer(sinkName))
	pipeline := C.wd_audio_new(uri, sinkName)
	if pipeline == nil {
		return nil, fmt.Errorf("GStreamer playback is unavailable")
	}
	return &AudioPlayer{pipeline: pipeline, rate: 1}, nil
}

func (p *AudioPlayer) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pipeline == nil {
		return fmt.Errorf("audio player is closed")
	}
	if C.gst_element_set_state(p.pipeline, C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_FAILURE {
		return fmt.Errorf("could not play audio")
	}
	p.playing = true
	return nil
}
func (p *AudioPlayer) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pipeline != nil {
		C.gst_element_set_state(p.pipeline, C.GST_STATE_PAUSED)
	}
	p.playing = false
}
func (p *AudioPlayer) Seek(position time.Duration, rate float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pipeline == nil {
		return fmt.Errorf("audio player is closed")
	}
	if rate != 0.5 && rate != 1 && rate != 1.5 && rate != 2 {
		return fmt.Errorf("unsupported playback speed")
	}
	if position < 0 {
		position = 0
	}
	if C.wd_audio_seek(p.pipeline, C.double(rate), C.gint64(position)) == 0 {
		return fmt.Errorf("could not seek this audio")
	}
	p.rate = rate
	return nil
}
func (p *AudioPlayer) State() AudioState {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pipeline == nil {
		return AudioState{}
	}
	var ended C.int
	message := C.wd_audio_event(p.pipeline, &ended)
	var err error
	if message != nil {
		err = fmt.Errorf("%s", C.GoString(message))
		C.g_free(C.gpointer(message))
	}
	if ended != 0 || err != nil {
		p.playing = false
		C.gst_element_set_state(p.pipeline, C.GST_STATE_PAUSED)
	}
	return AudioState{Position: time.Duration(C.wd_audio_position(p.pipeline)), Duration: time.Duration(C.wd_audio_duration(p.pipeline)), Playing: p.playing, Ended: ended != 0, Err: err}
}
func (p *AudioPlayer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pipeline != nil {
		C.wd_audio_close(p.pipeline)
		p.pipeline = nil
	}
	p.playing = false
}

// ActivateAudioPlayer pauses previous voice notes while retaining their position.
func ActivateAudioPlayer(p *AudioPlayer) {
	StopCurrentAudio()
	audioPlayingMu.Lock()
	currentNativeAudio = p
	audioPlayingMu.Unlock()
}
