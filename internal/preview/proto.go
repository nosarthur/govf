package preview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/draw"
)

// Protocol: how images reach the terminal.
type Protocol int

const (
	ProtoBlocks Protocol = iota // half-block cells, any truecolor term
	ProtoITerm                  // OSC 1337 inline images (iTerm2, WezTerm)
	ProtoKitty                  // kitty graphics protocol
)

func (p Protocol) String() string {
	switch p {
	case ProtoITerm:
		return "iterm"
	case ProtoKitty:
		return "kitty"
	}
	return "blocks"
}

// DetectProtocol picks a protocol from env. GOVF_IMAGES=blocks|iterm|kitty overrides.
func DetectProtocol(getenv func(string) string) Protocol {
	switch strings.ToLower(getenv("GOVF_IMAGES")) {
	case "blocks", "block":
		return ProtoBlocks
	case "iterm", "iterm2":
		return ProtoITerm
	case "kitty":
		return ProtoKitty
	}
	if getenv("TMUX") != "" { // needs passthrough; not handled
		return ProtoBlocks
	}
	if getenv("KITTY_WINDOW_ID") != "" || strings.Contains(getenv("TERM"), "kitty") {
		return ProtoKitty
	}
	switch getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm":
		return ProtoITerm
	}
	if getenv("LC_TERMINAL") == "iTerm2" || getenv("LC_TERMINAL_VERSION") != "" {
		return ProtoITerm
	}
	return ProtoBlocks
}

const (
	maxSendBytes = 4 << 20 // raw file larger than this gets re-encoded
	maxSendPx    = 1600    // longest side after downscale
)

// Encode returns bytes to send: raw file if small png/jpeg/gif (kitty: png only),
// else downscaled PNG.
func Encode(path string, proto Protocol) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	st, _ := f.Stat()
	rawOK := format == "png" || (proto == ProtoITerm && (format == "jpeg" || format == "gif"))
	if rawOK && st != nil && st.Size() <= maxSendBytes && cfg.Width <= maxSendPx && cfg.Height <= maxSendPx {
		return os.ReadFile(path)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return encodePNG(img, maxSendPx)
}

func encodePNG(img image.Image, maxPx int) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxPx || h > maxPx {
		if w >= h {
			h = h * maxPx / w
			w = maxPx
		} else {
			w = w * maxPx / h
			h = maxPx
		}
		dst := image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
		img = dst
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ITermSeq builds an OSC 1337 inline-image sequence sized cols x rows cells.
func ITermSeq(data []byte, cols, rows int) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=1:", len(data), cols, rows)
	b.WriteString(base64.StdEncoding.EncodeToString(data))
	b.WriteByte('\a')
	return b.Bytes()
}

// KittySeq builds chunked kitty graphics commands (PNG payload) at cursor.
func KittySeq(pngData []byte, cols, rows int) []byte {
	enc := base64.StdEncoding.EncodeToString(pngData)
	var b bytes.Buffer
	const chunk = 4096
	first := true
	for len(enc) > 0 {
		n := min(chunk, len(enc))
		more := 0
		if n < len(enc) {
			more = 1
		}
		b.WriteString("\x1b_G")
		if first {
			fmt.Fprintf(&b, "f=100,a=T,q=2,C=1,c=%d,r=%d,", cols, rows)
			first = false
		}
		fmt.Fprintf(&b, "m=%d;", more)
		b.WriteString(enc[:n])
		b.WriteString("\x1b\\")
		enc = enc[n:]
	}
	return b.Bytes()
}

// KittyDelete clears all visible kitty images.
func KittyDelete() []byte { return []byte("\x1b_Ga=d,d=A,q=2\x1b\\") }

// Native renders path as a protocol payload; falls back to Text on failure.
func Native(path string, proto Protocol) Result {
	if !IsImagePath(path) {
		return Result{Kind: KindNone}
	}
	data, err := Encode(path, proto)
	if err != nil {
		return Result{Kind: KindError, Err: err}
	}
	return Result{Kind: KindImage, Data: data}
}
