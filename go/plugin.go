package vibrato

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	flutter "github.com/go-flutter-desktop/go-flutter"
	"github.com/go-flutter-desktop/go-flutter/plugin"
	"github.com/google/uuid"
)

const channelName = "vibrato"

// VibratoPlugin implements flutter.Plugin and handles method.
type VibratoPlugin struct {
	Streamers map[uuid.UUID]*VibratoEffects
}

func New() *VibratoPlugin {
	return &VibratoPlugin{
		Streamers: map[uuid.UUID]*VibratoEffects{},
	}
}

var _ flutter.Plugin = &VibratoPlugin{} // compile-time type check

func decodeSource(name string, source io.ReadCloser) (s beep.StreamSeekCloser, fmt beep.Format, err error) {
	n := strings.ToLower(name)
	if strings.HasSuffix(n, "mp3") {
		return mp3.Decode(source)
	} else if strings.HasSuffix(n, "wav") {
		return wav.Decode(source)
	} else if strings.HasSuffix(n, "ogg") {
		return vorbis.Decode(source)
	} else {
		return nil, beep.Format{}, errors.New("unsupported stream")
	}
}

// InitPlugin initializes the plugin.
func (p *VibratoPlugin) InitPlugin(messenger plugin.BinaryMessenger) error {
	speaker.Init(beep.SampleRate(44100), 4410)
	channel := plugin.NewMethodChannel(messenger, channelName, plugin.StandardMethodCodec{})
	channel.HandleFunc("playFile", p.handlePlayFile)
	channel.HandleFunc("playBuffer", p.handlePlayBuffer)
	channel.HandleFunc("listStreams", p.handleListStreams)
	channel.HandleFunc("closeStream", p.handleCloseStream)
	channel.HandleFunc("pauseStream", p.handlePauseStream)
	channel.HandleFunc("seekStream", p.handleSeekStream)
	channel.HandleFunc("streamInfo", p.handleStreamInfo)
	return nil
}

// TODO: return stream ID
// TODO: get decoder by filename
func (p *VibratoPlugin) handlePlayFile(arguments interface{}) (reply interface{}, err error) {
	args := arguments.(map[interface{}]interface{})
	f, err := os.Open(args["file"].(string))
	name := args["name"].(string)

	if err != nil {
		return nil, err
	}
	s, beepFmt, err := decodeSource(args["file"].(string), f)
	if err != nil {
		return nil, err
	}

	var id = uuid.New()
	p.Streamers[id] = NewEffects(id, name, beepFmt.SampleRate, s)

	speaker.Play(p.Streamers[id])
	return id.String(), nil
}

func (p *VibratoPlugin) handlePlayBuffer(arguments interface{}) (reply interface{}, err error) {
	args := arguments.(map[interface{}]interface{})
	f := ioutil.NopCloser(bytes.NewReader(args["buffer"].([]byte)))
	fmt := args["format"].(string)
	name := args["name"].(string)

	s, beepFmt, err := decodeSource(fmt, f)
	if err != nil {
		return nil, err
	}

	var id = uuid.New()
	p.Streamers[id] = NewEffects(id, name, beepFmt.SampleRate, s)

	speaker.Play(p.Streamers[id])
	return id.String(), nil
}

func (p *VibratoPlugin) handleCloseStream(arguments interface{}) (reply interface{}, err error) {
	stream, err := p.getStream(arguments)
	if err != nil {
		return nil, err
	}

	if err := stream.Close(); err != nil {
		return nil, err
	}

	delete(p.Streamers, stream.ID)
	return nil, nil
}

func (p *VibratoPlugin) handlePauseStream(arguments interface{}) (reply interface{}, err error) {
	return nil, errors.New("Not implemented")
}

func (p *VibratoPlugin) handleSeekStream(arguments interface{}) (reply interface{}, err error) {
	stream, err := p.getStream(arguments)

	if err != nil {
		return nil, err
	}

	args := arguments.(map[interface{}]interface{})
	position := int(args["position"].(int32))

	err = stream.Seek(position)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *VibratoPlugin) handleStreamInfo(arguments interface{}) (reply interface{}, err error) {
	stream, err := p.getStream(arguments)
	if err != nil {
		return nil, err
	}

	response := map[interface{}]interface{}{}
	response["name"] = stream.Name
	response["position"] = int64(stream.Position())
	response["length"] = int64(stream.Len())
	response["sampleRate"] = int64(stream.SampleRate)
	return response, nil
}

func (p *VibratoPlugin) getStream(arguments interface{}) (stream *VibratoEffects, err error) {
	args := arguments.(map[interface{}]interface{})
	id, err := uuid.Parse(args["id"].(string))

	if err != nil {
		return nil, err
	}

	if val, ok := p.Streamers[id]; ok {
		return val, nil
	} else {
		return nil, errors.New("stream does not exist")
	}
}

func (p *VibratoPlugin) handleListStreams(arguments interface{}) (reply interface{}, err error) {
	streams := map[interface{}]interface{}{}

	for id, streamer := range p.Streamers {
		streams[id.String()] = streamer.Name
	}

	return streams, nil
}

type VibratoEffects struct {
	ID         uuid.UUID
	Name       string
	SampleRate beep.SampleRate
	streamer   beep.StreamSeekCloser
}

func NewEffects(id uuid.UUID, name string, sampleRate beep.SampleRate, streamer beep.StreamSeekCloser) *VibratoEffects {
	return &VibratoEffects{
		ID:         id,
		Name:       name,
		SampleRate: sampleRate,
		streamer:   streamer,
	}
}

func (e *VibratoEffects) Len() int {
	return e.streamer.Len()
}

func (e *VibratoEffects) Position() int {
	return e.streamer.Position()
}

func (e *VibratoEffects) Seek(p int) error {
	speaker.Lock()
	err := e.streamer.Seek(p)
	speaker.Unlock()
	return err
}

func (e *VibratoEffects) Stream(samples [][2]float64) (n int, ok bool) {
	return e.streamer.Stream(samples)
}

func (e *VibratoEffects) Err() error {
	return e.streamer.Err()
}

func (e *VibratoEffects) Close() error {
	return e.streamer.Close()
}
