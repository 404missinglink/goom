package sdl

import (
	"fmt"
	"time"

	"github.com/tinogoehlert/go-sdl2/mix"
	"github.com/tinogoehlert/go-sdl2/sdl"

	"github.com/tinogoehlert/goom/audio/music"
	"github.com/tinogoehlert/goom/audio/sfx"
)

// Audio is the SDL sound and music driver.
type Audio struct {
	// sound driver fields
	sounds *sfx.Sounds
	chunks map[string]*mix.Chunk
	test   bool

	// music driver fields
	tracks       *music.TrackStore
	currentTrack *mix.Music
}

// TestMode silences all sounds and music and sets all delays to 0 for testing.
func (a *Audio) TestMode() {
	a.test = false
}

// InitAudio inits the driver.
func (a *Audio) InitAudio(sounds *sfx.Sounds) error {
	err := initAudio()
	if err != nil {
		return fmt.Errorf("failed to init SDL subsystem: %s", err.Error())
	}

	if _, err := mix.OpenAudioDevice(22050, mix.DEFAULT_FORMAT, 2, 4096, "", sdl.AUDIO_ALLOW_ANY_CHANGE); err != nil {
		return fmt.Errorf("failed to open audio device: %s", err.Error())
	}

	a.sounds = sounds
	a.chunks = make(map[string]*mix.Chunk)

	return nil
}

// InitMusic sets the music tracks.
func (a *Audio) InitMusic(tracks *music.TrackStore) error {
	err := initAudio()
	if err != nil {
		return fmt.Errorf("failed to init SDL subsystem: %s", err.Error())
	}
	if _, err := mix.OpenAudioDevice(22050, mix.DEFAULT_FORMAT, 2, 4096, "", sdl.AUDIO_ALLOW_ANY_CHANGE); err != nil {
		return fmt.Errorf("failed to open audio device: %s", err.Error())
	}
	a.tracks = tracks
	return nil
}

// PlayMusic plays a MUS track.
// For SDL playback the MUS track is converted to a MID file and
// stored in a temp dir unless the target MID file is already present.
func (a *Audio) PlayMusic(track *music.Track) error {
	if track == nil {
		return fmt.Errorf("no track given (nil)")
	}

	rwOps, err := sdl.RWFromMem(track.MidiStream.Bytes())
	if err != nil {
		return err
	}

	a.currentTrack, err = mix.LoadMUSTypeRW(rwOps, mix.MID, 0)
	if err != nil {
		return fmt.Errorf("could not load MIDI: %s", err.Error())
	}
	return a.currentTrack.FadeIn(-1, 1000)
}

// Play simply plays an audio chunk with the given name
func (a *Audio) Play(name string) error {
	chunk, err := a.getChunk(name)
	if err != nil {
		return err
	}
	_, err = chunk.Play(-1, 0)
	return err
}

// PlayAtPosition plays a sound in 2D virtual space using a given distance and angle.
func (a *Audio) PlayAtPosition(name string, distance float32, angle int16) error {
	chunk, err := a.getChunk(name)
	if err != nil {
		return err
	}
	if distance > 255 {
		distance = 255
	}
	channel, err := chunk.Play(-1, 0)
	mix.SetPosition(channel, angle, uint8(distance))
	return err
}

func (a *Audio) getChunk(name string) (*mix.Chunk, error) {
	if chunk, ok := a.chunks[name]; ok {
		return chunk, nil
	}
	return a.createChunk(name)
}

func (a *Audio) createChunk(name string) (*mix.Chunk, error) {
	sound, ok := sfx.Sounds(*a.sounds)[name]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}
	rwOps, err := sdl.RWFromMem(sound.ToWAV())
	if err != nil {
		return nil, err
	}
	chunk, err := mix.LoadWAVRW(rwOps, false)
	// chunk, err := mix.QuickLoadWAV(sound.ToWAV())
	if err != nil {
		return nil, fmt.Errorf("could not load WAV: %s", err.Error())
	}
	a.chunks[name] = chunk
	return chunk, nil
}

// Close closes the mixer and quits the SDL audio driver.
func (a *Audio) Close() {
	defer mix.CloseAudio()
	defer sdl.AudioQuit()
	t := time.Now()
	fadeOutDur := time.Second

	fmt.Printf("waiting for audio channels to stop: #0")
	for {
		n := mix.Playing(-1)
		// Wait up to 500 ms for playing channels when in non-test mode.
		if time.Now().Sub(t) > fadeOutDur || n == 0 || a.test {
			fmt.Printf(", OK\n")
			break
		}
		fmt.Printf("\b%d", n)
		time.Sleep(fadeOutDur / 10)
	}
}
