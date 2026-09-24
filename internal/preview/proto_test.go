package preview

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDetectProtocol(t *testing.T) {
	cases := []struct {
		env  map[string]string
		want Protocol
	}{
		{map[string]string{}, ProtoBlocks},
		{map[string]string{"TERM_PROGRAM": "iTerm.app"}, ProtoITerm},
		{map[string]string{"LC_TERMINAL_VERSION": "3.6.11"}, ProtoITerm},
		{map[string]string{"TERM": "xterm-kitty"}, ProtoKitty},
		{map[string]string{"TERM_PROGRAM": "iTerm.app", "TMUX": "/tmp/x"}, ProtoBlocks},
		{map[string]string{"TERM_PROGRAM": "iTerm.app", "GOVF_IMAGES": "blocks"}, ProtoBlocks},
		{map[string]string{"GOVF_IMAGES": "kitty"}, ProtoKitty},
	}
	for _, c := range cases {
		if got := DetectProtocol(envOf(c.env)); got != c.want {
			t.Errorf("%v: got %v want %v", c.env, got, c.want)
		}
	}
}

func TestSeqs(t *testing.T) {
	s := string(ITermSeq([]byte("abc"), 10, 5))
	if !strings.HasPrefix(s, "\x1b]1337;File=inline=1;size=3;width=10;height=5;preserveAspectRatio=1:YWJj\a") {
		t.Errorf("iterm seq %q", s)
	}
	big := bytes.Repeat([]byte{1}, 5000) // base64 > 4096 => 2 chunks
	k := string(KittySeq(big, 3, 4))
	if strings.Count(k, "\x1b_G") != 2 || !strings.Contains(k, "f=100,a=T,q=2,C=1,c=3,r=4,m=1;") || !strings.HasSuffix(k, "\x1b\\") {
		t.Errorf("kitty seq chunks=%d", strings.Count(k, "\x1b_G"))
	}
	if !strings.Contains(k, "\x1b_Gm=0;") {
		t.Error("kitty last chunk m=0")
	}
}

func TestEncodeAndNative(t *testing.T) {
	d := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 2000, 10))
	for x := 0; x < 2000; x++ {
		img.Set(x, 0, color.RGBA{255, 0, 0, 255})
	}
	jp := filepath.Join(d, "wide.jpg")
	f, _ := os.Create(jp)
	jpeg.Encode(f, img, nil)
	f.Close()
	// too wide => re-encoded png, downscaled
	data, err := Encode(jp, ProtoITerm)
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || cfg.Width != 1600 || cfg.Height != 8 {
		t.Errorf("downscale: %v %s %+v", err, format, cfg)
	}
	// small jpeg: raw for iterm, png for kitty
	small := image.NewRGBA(image.Rect(0, 0, 4, 4))
	sp := filepath.Join(d, "s.jpg")
	f, _ = os.Create(sp)
	jpeg.Encode(f, small, nil)
	f.Close()
	raw, _ := os.ReadFile(sp)
	if data, _ := Encode(sp, ProtoITerm); !bytes.Equal(data, raw) {
		t.Error("iterm should send raw jpeg")
	}
	if data, _ := Encode(sp, ProtoKitty); bytes.Equal(data, raw) {
		t.Error("kitty should re-encode jpeg")
	}
	if r := Native(sp, ProtoITerm); r.Kind != KindImage || len(r.Data) == 0 {
		t.Errorf("native: %+v", r.Kind)
	}
	if r := Native(filepath.Join(d, "x.txt"), ProtoITerm); r.Kind != KindNone {
		t.Error("native non-image")
	}
	// svg/pdf: raw bytes on iterm, not an image elsewhere
	svg := filepath.Join(d, "v.svg")
	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"/>`)
	os.WriteFile(svg, svgData, 0o644)
	if r := Native(svg, ProtoITerm); r.Kind != KindImage || !bytes.Equal(r.Data, svgData) {
		t.Errorf("svg iterm: %v", r.Kind)
	}
	if r := Native(svg, ProtoKitty); r.Kind != KindError {
		t.Errorf("svg kitty should fail decode: %v", r.Kind)
	}
	if r := File(svg, 10, 5); r.Kind != KindText {
		t.Errorf("svg fallback text: %v", r.Kind)
	}
	pdf := filepath.Join(d, "p.pdf")
	os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644)
	if r := Native(pdf, ProtoITerm); r.Kind != KindImage {
		t.Errorf("pdf iterm: %v", r.Kind)
	}
	if r := Native(pdf, ProtoBlocks); r.Kind != KindNone {
		t.Errorf("pdf blocks: %v", r.Kind)
	}
	bad := filepath.Join(d, "bad.png")
	os.WriteFile(bad, []byte("nope"), 0o644)
	if r := Native(bad, ProtoITerm); r.Kind != KindError {
		t.Error("native bad image")
	}
}
