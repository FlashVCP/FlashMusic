package local

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"

	"netease-music-mygo/internal/source"
)

// ParseFile inspects a local audio file and extracts metadata, duration, and associated lyrics.
func ParseFile(path string) (*source.LocalTrack, error) {
	ext := strings.ToLower(filepath.Ext(path))
	track := &source.LocalTrack{
		ID:     source.LocalTrackID(path),
		Path:   path,
		Format: strings.TrimPrefix(ext, "."),
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	track.Size = fi.Size()

	// Default title from filename: "Artist - Title.ext" or "Title.ext"
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if parts := strings.SplitN(base, " - ", 2); len(parts) == 2 {
		track.Artist = strings.TrimSpace(parts[0])
		track.Title = strings.TrimSpace(parts[1])
	} else {
		track.Title = base
	}

	switch ext {
	case ".flac":
		parseFLAC(path, track)
	case ".mp3":
		parseMP3(path, track)
	}

	// Fallback to cover image files in the same directory if no embedded cover
	if len(track.Cover) == 0 {
		dir := filepath.Dir(path)
		for _, name := range []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png"} {
			if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil && len(b) > 0 {
				track.Cover = b
				break
			}
		}
	}

	// Look for local lyrics (.lrc) in the same directory
	lrcPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".lrc"
	if b, err := os.ReadFile(lrcPath); err == nil {
		track.Lyrics = string(b)
	}

	return track, nil
}

func parseFLAC(path string, track *source.LocalTrack) {
	stream, err := flac.Open(path)
	if err != nil {
		return
	}
	defer stream.Close()

	info := stream.Info
	if info != nil {
		if info.SampleRate > 0 {
			track.SampleRate = int(info.SampleRate)
			track.Duration = int64((info.NSamples * 1000) / uint64(info.SampleRate))
		}
		track.BitDepth = int(info.BitsPerSample)
		if track.Duration > 0 && track.Size > 0 {
			track.Bitrate = int((track.Size * 8) / track.Duration)
		}
	}

	for _, block := range stream.Blocks {
		switch body := block.Body.(type) {
		case *meta.Picture:
			if len(body.Data) > 0 && len(track.Cover) == 0 {
				track.Cover = body.Data
			}
		case *meta.VorbisComment:
			for _, tag := range body.Tags {
				key := strings.ToUpper(strings.TrimSpace(tag[0]))
				val := strings.TrimSpace(tag[1])
				switch key {
				case "TITLE":
					if val != "" {
						track.Title = val
					}
				case "ARTIST":
					if val != "" {
						track.Artist = val
					}
				case "ALBUM":
					if val != "" {
						track.Album = val
					}
				}
			}
		}
	}
}

var (
	mp3BitratesV1  = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	mp3BitratesV2  = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
	mp3SampleRates = [4]int{44100, 48000, 32000, 0}
)

func parseMP3(path string, track *source.LocalTrack) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	// 1. Try parsing ID3v2 header
	var id3Size int64
	var header [10]byte
	if _, err := io.ReadFull(f, header[:]); err == nil && string(header[:3]) == "ID3" {
		id3Size = int64(syncsafe(binary.BigEndian.Uint32(header[6:10]))) + 10
		tagData := make([]byte, id3Size-10)
		if _, err := io.ReadFull(f, tagData); err == nil {
			parseID3v2Frames(tagData, int(header[3]), track)
		}
	}

	// 2. Parse MPEG frame header to accurately determine SampleRate, Bitrate, and calculate duration if needed
	parseMP3FrameHeader(f, id3Size, track)

	// 3. Try fallback to ID3v1 (last 128 bytes) if tags are empty
	if track.Title == "" || track.Artist == "" {
		fi, err := f.Stat()
		if err == nil && fi.Size() >= 128 {
			var id3v1 [128]byte
			if _, err := f.ReadAt(id3v1[:], fi.Size()-128); err == nil && string(id3v1[:3]) == "TAG" {
				if track.Title == "" {
					track.Title = cleanString(string(id3v1[3:33]))
				}
				if track.Artist == "" {
					track.Artist = cleanString(string(id3v1[33:63]))
				}
				if track.Album == "" {
					track.Album = cleanString(string(id3v1[63:93]))
				}
			}
		}
	}
}

func parseMP3FrameHeader(f *os.File, id3Size int64, track *source.LocalTrack) {
	buf := make([]byte, 16384)
	n, err := f.ReadAt(buf, id3Size)
	if err != nil && n < 4 {
		return
	}
	for i := 0; i <= n-4; i++ {
		b := buf[i:]
		if b[0] == 0xff && (b[1]&0xe0) == 0xe0 {
			version := (b[1] >> 3) & 3 // 3: MPEG 1, 2: MPEG 2, 0: MPEG 2.5
			layer := (b[1] >> 1) & 3   // 1: layer III
			if version == 1 || layer != 1 {
				continue
			}
			bi, ri := b[2]>>4, (b[2]>>2)&3
			if bi == 0 || bi == 15 || ri == 3 {
				continue
			}
			var br int
			if version == 3 {
				br = mp3BitratesV1[bi]
			} else {
				br = mp3BitratesV2[bi]
			}
			track.Bitrate = br
			track.SampleRate = mp3SampleRates[ri]
			track.BitDepth = 16
			if track.Duration == 0 && br > 0 && track.Size > id3Size {
				audioBytes := track.Size - id3Size
				track.Duration = (audioBytes * 8) / int64(br)
			}
			return
		}
	}
}

func parseID3v2Frames(data []byte, version int, track *source.LocalTrack) {
	r := bytes.NewReader(data)
	for r.Len() >= 10 {
		var id [4]byte
		if _, err := io.ReadFull(r, id[:]); err != nil || id[0] == 0 {
			break
		}
		var size uint32
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			break
		}
		if version == 4 {
			size = syncsafe(size)
		}
		var flags uint16
		binary.Read(r, binary.BigEndian, &flags)

		if int(size) > r.Len() || size == 0 {
			break
		}
		frameData := make([]byte, size)
		if _, err := io.ReadFull(r, frameData); err != nil {
			break
		}

		frameID := string(id[:])
		switch frameID {
		case "TIT2": // Title
			if val := decodeText(frameData); val != "" {
				track.Title = val
			}
		case "TPE1": // Artist
			if val := decodeText(frameData); val != "" {
				track.Artist = val
			}
		case "TALB": // Album
			if val := decodeText(frameData); val != "" {
				track.Album = val
			}
		case "TLEN": // Duration in ms
			if val := decodeText(frameData); val != "" {
				var d int64
				for _, c := range val {
					if c >= '0' && c <= '9' {
						d = d*10 + int64(c-'0')
					}
				}
				if d > 0 {
					track.Duration = d
				}
			}
		case "APIC": // Embedded cover
			if len(track.Cover) == 0 {
				track.Cover = extractAPIC(frameData)
			}
		}
	}
}

func extractAPIC(data []byte) []byte {
	if len(data) < 4 {
		return nil
	}
	mimeEnd := bytes.IndexByte(data[1:], 0)
	if mimeEnd < 0 {
		return nil
	}
	idx := 1 + mimeEnd + 1
	if idx >= len(data) {
		return nil
	}
	idx++ // picture type
	if idx >= len(data) {
		return nil
	}
	encoding := data[0]
	if encoding == 1 || encoding == 2 {
		descEnd := bytes.Index(data[idx:], []byte{0, 0})
		if descEnd >= 0 {
			idx += descEnd + 2
		}
	} else {
		descEnd := bytes.IndexByte(data[idx:], 0)
		if descEnd >= 0 {
			idx += descEnd + 1
		}
	}
	if idx < len(data) {
		return data[idx:]
	}
	return nil
}

func decodeText(data []byte) string {
	if len(data) <= 1 {
		return ""
	}
	encoding := data[0]
	payload := data[1:]
	switch encoding {
	case 0: // ISO-8859-1 or GBK in Chinese MP3s
		if utf8, err := simplifiedchinese.GBK.NewDecoder().Bytes(payload); err == nil && len(utf8) > 0 {
			return cleanString(string(utf8))
		}
		return cleanString(string(payload))
	case 1: // UTF-16 with BOM
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
		if res, err := decoder.Bytes(payload); err == nil {
			return cleanString(string(res))
		}
	case 2: // UTF-16BE
		decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		if res, err := decoder.Bytes(payload); err == nil {
			return cleanString(string(res))
		}
	case 3: // UTF-8
		return cleanString(string(payload))
	}
	return cleanString(string(payload))
}

func syncsafe(n uint32) uint32 {
	return (n&0x7F) | ((n>>8)&0x7F)<<7 | ((n>>16)&0x7F)<<14 | ((n>>24)&0x7F)<<21
}

func cleanString(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\x00"))
}
