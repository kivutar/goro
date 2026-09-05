package res

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	_ "golang.org/x/image/bmp"
)

func LoadImage(manager *Manager, candidates []string) (image.Image, string, error) {
	for _, candidate := range candidates {
		data, err := manager.ReadFile(candidate)
		if err != nil {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			img, err = decodeTGA(data)
			if err != nil {
				return nil, candidate, fmt.Errorf("decode image: %w", err)
			}
		}
		return applyROTransparency(img), candidate, nil
	}
	return nil, "", fmt.Errorf("image not found: %s", strings.Join(candidates, ", "))
}

func LoadImageExact(manager *Manager, candidates []string) (image.Image, string, error) {
	for _, candidate := range candidates {
		data, err := manager.ReadFileExact(candidate)
		if err != nil {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			img, err = decodeTGA(data)
			if err != nil {
				return nil, candidate, fmt.Errorf("decode image: %w", err)
			}
		}
		return applyROTransparency(img), candidate, nil
	}
	return nil, "", fmt.Errorf("image not found: %s", strings.Join(candidates, ", "))
}

func DecodeImageData(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		img, err = decodeTGA(data)
		if err != nil {
			return nil, fmt.Errorf("decode image: %w", err)
		}
	}
	return applyROTransparency(img), nil
}

func applyROTransparency(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	draw.Draw(out, bounds, img, bounds.Min, draw.Src)
	for i := 0; i < len(out.Pix); i += 4 {
		if isROMagenta(out.Pix[i], out.Pix[i+1], out.Pix[i+2]) {
			out.Pix[i+3] = 0
		}
	}
	clearTransparentRGB(out)
	return out
}

func ApplyEffectTransparency(img image.Image) *image.NRGBA {
	return ApplyEffectTransparencyWithBlackKey(img, true)
}

func ApplyEffectTransparencyWithBlackKey(img image.Image, blackKey bool) *image.NRGBA {
	out := applyROTransparency(img)
	if blackKey {
		for i := 0; i < len(out.Pix); i += 4 {
			if out.Pix[i] < 3 && out.Pix[i+1] < 3 && out.Pix[i+2] < 3 {
				out.Pix[i+3] = 0
			}
		}
	}
	clearTransparentRGB(out)
	return out
}

func clearTransparentRGB(img *image.NRGBA) {
	if img == nil {
		return
	}
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] != 0 {
			continue
		}
		img.Pix[i+0] = 0
		img.Pix[i+1] = 0
		img.Pix[i+2] = 0
	}
}

func isROMagenta(r, g, b uint8) bool {
	return r >= 248 && g <= 8 && b >= 248
}

func GroundTextureCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return []string{
		"data\\texture\\" + name,
		"data/texture/" + strings.ReplaceAll(name, "\\", "/"),
		"texture\\" + name,
		"texture/" + strings.ReplaceAll(name, "\\", "/"),
		name,
		strings.ReplaceAll(name, "\\", "/"),
	}
}

func WaterTextureCandidates(waterType, frame int) []string {
	if frame < 0 {
		frame = 0
	}
	frame %= 32
	name := fmt.Sprintf("water%d%02d.jpg", waterType, frame)
	koreanFolder := "data\\texture\\워터\\"
	return []string{
		koreanFolder + name,
		strings.ReplaceAll(koreanFolder, "\\", "/") + name,
		"data\\texture\\water\\" + name,
		"data/texture/water/" + name,
		"texture\\water\\" + name,
		"texture/water/" + name,
		"data\\texture\\" + name,
		"data/texture/" + name,
		"texture\\" + name,
		"texture/" + name,
		name,
	}
}

func EffectTextureCandidates(name string) []string {
	name = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(name, ".tga"), ".bmp"))
	if name == "" {
		return nil
	}
	return []string{
		"data\\texture\\effect\\" + name + ".tga",
		"data\\texture\\effect\\" + name + ".bmp",
		"data/texture/effect/" + name + ".tga",
		"data/texture/effect/" + name + ".bmp",
		"texture\\effect\\" + name + ".tga",
		"texture\\effect\\" + name + ".bmp",
		"texture/effect/" + name + ".tga",
		"texture/effect/" + name + ".bmp",
		"effect\\" + name + ".tga",
		"effect\\" + name + ".bmp",
		"effect/" + name + ".tga",
		"effect/" + name + ".bmp",
		name + ".tga",
		name + ".bmp",
	}
}

func ItemIconTextureCandidates(resource string) []string {
	resource = strings.TrimSpace(strings.TrimSuffix(resource, ".bmp"))
	if resource == "" {
		return nil
	}
	stem := strings.ReplaceAll(resource, "/", "\\")
	filenameOnly := stem
	if pos := strings.LastIndexAny(filenameOnly, `\/`); pos >= 0 && pos+1 < len(filenameOnly) {
		filenameOnly = filenameOnly[pos+1:]
	}

	const uiKorPrefix = "data\\texture\\유저인터페이스\\item\\"
	prefixes := []string{
		uiKorPrefix,
		strings.ReplaceAll(uiKorPrefix, "\\", "/"),
		"texture\\유저인터페이스\\item\\",
		"data\\texture\\item\\",
		"data/texture/item/",
		"texture\\item\\",
		"texture/item/",
		"item\\",
		"item/",
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(prefixes)*2+2)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	for _, prefix := range prefixes {
		add(prefix + stem + ".bmp")
		if filenameOnly != stem {
			add(prefix + filenameOnly + ".bmp")
		}
	}
	add(stem + ".bmp")
	if filenameOnly != stem {
		add(filenameOnly + ".bmp")
	}
	return out
}

func InterfaceTextureCandidates(resource string) []string {
	resource = strings.TrimSpace(strings.TrimSuffix(resource, ".bmp"))
	if resource == "" {
		return nil
	}
	stem := strings.ReplaceAll(resource, "/", "\\")
	filenameOnly := stem
	if pos := strings.LastIndexAny(filenameOnly, `\/`); pos >= 0 && pos+1 < len(filenameOnly) {
		filenameOnly = filenameOnly[pos+1:]
	}

	const uiKorPrefix = "data\\texture\\유저인터페이스\\"
	prefixes := []string{
		uiKorPrefix,
		strings.ReplaceAll(uiKorPrefix, "\\", "/"),
		"texture\\유저인터페이스\\",
		"data\\texture\\interface\\",
		"data/texture/interface/",
		"texture\\interface\\",
		"texture/interface/",
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(prefixes)*2+2)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	for _, prefix := range prefixes {
		add(prefix + stem + ".bmp")
		if filenameOnly != stem {
			add(prefix + filenameOnly + ".bmp")
		}
	}
	add(stem + ".bmp")
	if filenameOnly != stem {
		add(filenameOnly + ".bmp")
	}
	return out
}

func NPCCutinTextureCandidates(resource string) []string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return nil
	}
	filenameOnly := strings.TrimRight(strings.ReplaceAll(resource, "/", "\\"), "\\")
	if pos := strings.LastIndexAny(filenameOnly, `\/`); pos >= 0 && pos+1 < len(filenameOnly) {
		filenameOnly = filenameOnly[pos+1:]
	}
	if filenameOnly == "" || filenameOnly == "." || filenameOnly == ".." {
		return nil
	}
	if dot := strings.LastIndexByte(filenameOnly, '.'); dot < 0 {
		filenameOnly += ".bmp"
	}

	const uiIllustrationPrefix = "data\\texture\\유저인터페이스\\illust\\"
	prefixes := []string{
		uiIllustrationPrefix,
		strings.ReplaceAll(uiIllustrationPrefix, "\\", "/"),
		"texture\\유저인터페이스\\illust\\",
		"data\\texture\\interface\\illust\\",
		"data/texture/interface/illust/",
		"texture\\interface\\illust\\",
		"texture/interface/illust/",
		"illust\\",
		"illust/",
	}
	seen := make(map[string]struct{}, len(prefixes)+1)
	out := make([]string, 0, len(prefixes)+1)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	for _, prefix := range prefixes {
		add(prefix + filenameOnly)
	}
	add(filenameOnly)
	return out
}

func ItemCollectionTextureCandidates(resource string) []string {
	resource = strings.TrimSpace(strings.TrimSuffix(resource, ".bmp"))
	if resource == "" {
		return nil
	}
	stem := strings.ReplaceAll(resource, "/", "\\")
	filenameOnly := stem
	if pos := strings.LastIndexAny(filenameOnly, `\/`); pos >= 0 && pos+1 < len(filenameOnly) {
		filenameOnly = filenameOnly[pos+1:]
	}

	const uiKorPrefix = "data\\texture\\유저인터페이스\\collection\\"
	prefixes := []string{
		uiKorPrefix,
		strings.ReplaceAll(uiKorPrefix, "\\", "/"),
		"texture\\유저인터페이스\\collection\\",
		"collection\\",
		"texture\\collection\\",
		"texture\\interface\\collection\\",
		"data\\collection\\",
		"data\\texture\\collection\\",
		"data\\texture\\interface\\collection\\",
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(prefixes)*2+4)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	for _, prefix := range prefixes {
		add(prefix + stem)
		add(prefix + stem + ".bmp")
		if filenameOnly != stem {
			add(prefix + filenameOnly)
			add(prefix + filenameOnly + ".bmp")
		}
	}
	return out
}

func CardIllustrationTextureCandidates(resource string) []string {
	resource = strings.TrimSpace(strings.TrimSuffix(resource, ".bmp"))
	if resource == "" {
		return nil
	}
	stem := strings.ReplaceAll(resource, "/", "\\")
	filenameOnly := stem
	if pos := strings.LastIndexAny(filenameOnly, `\/`); pos >= 0 && pos+1 < len(filenameOnly) {
		filenameOnly = filenameOnly[pos+1:]
	}

	const uiKorPrefix = "data\\texture\\유저인터페이스\\cardbmp\\"
	prefixes := []string{
		uiKorPrefix,
		strings.ReplaceAll(uiKorPrefix, "\\", "/"),
		"texture\\유저인터페이스\\cardbmp\\",
		"data\\texture\\interface\\cardbmp\\",
		"data/texture/interface/cardbmp/",
		"texture\\interface\\cardbmp\\",
		"texture/interface/cardbmp/",
		"cardbmp\\",
		"cardbmp/",
	}
	seen := make(map[string]struct{}, len(prefixes)*2+2)
	out := make([]string, 0, len(prefixes)*2+2)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	for _, prefix := range prefixes {
		add(prefix + stem + ".bmp")
		if filenameOnly != stem {
			add(prefix + filenameOnly + ".bmp")
		}
	}
	add(stem + ".bmp")
	if filenameOnly != stem {
		add(filenameOnly + ".bmp")
	}
	return out
}
