package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"ezpub/internal/lang"
)

type imageAsset struct {
	Path string
	Name string
	Ref  string
}

func scanImageDirs(dirs []string) (map[string]string, []string, error) {
	resolver := map[string]string{}
	var images []string
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !isImagePath(path) {
				return nil
			}
			clean := filepath.Clean(path)
			images = append(images, clean)
			resolver[strings.ToLower(filepath.Base(clean))] = clean
			if rel, err := filepath.Rel(dir, clean); err == nil {
				resolver[strings.ToLower(filepath.ToSlash(rel))] = clean
			}
			return nil
		}); err != nil {
			return nil, nil, err
		}
	}
	return resolver, images, nil
}

func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp":
		return true
	default:
		return false
	}
}

func runImages(args []string) error {
	fs := flag.NewFlagSet("images", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var outPath string
	var copyRef string
	var openImage string
	fs.StringVar(&outPath, "o", "", "write image TSV")
	fs.StringVar(&copyRef, "copy-ref", "", "print and copy raw HTML reference for image basename")
	fs.StringVar(&openImage, "open", "", "open image basename with the system viewer")
	if err := fs.Parse(interspersedFlags(args, nil)); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New(lang.ErrImagesInputCount)
	}

	assets, err := collectImageAssets(fs.Args())
	if err != nil {
		return err
	}
	if outPath != "" {
		if err := writeImageManifest(outPath, assets); err != nil {
			return err
		}
	}
	if copyRef != "" {
		asset, ok := findImageAsset(assets, copyRef)
		if !ok {
			return fmt.Errorf("image not found: %s", copyRef)
		}
		fmt.Println(asset.Ref)
		return copyToClipboard(asset.Ref)
	}
	if openImage != "" {
		asset, ok := findImageAsset(assets, openImage)
		if !ok {
			return fmt.Errorf("image not found: %s", openImage)
		}
		return openImageFile(asset.Path)
	}
	if outPath == "" {
		return writeImageManifest("", assets)
	}
	return nil
}

func collectImageAssets(paths []string) ([]imageAsset, error) {
	var images []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isImagePath(p) {
				images = append(images, filepath.Clean(p))
			}
			continue
		}
		err = filepath.WalkDir(p, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !isImagePath(path) {
				return nil
			}
			images = append(images, filepath.Clean(path))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(images)
	seen := map[string]string{}
	var assets []imageAsset
	for _, path := range images {
		name := filepath.Base(path)
		key := strings.ToLower(name)
		if prev, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate image filename %q: %s and %s", name, prev, path)
		}
		seen[key] = path
		assets = append(assets, imageAsset{
			Path: path,
			Name: name,
			Ref:  imageReference(name),
		})
	}
	return assets, nil
}

func writeImageManifest(path string, assets []imageAsset) error {
	var b strings.Builder
	b.WriteString("# ezpub images v1\n")
	b.WriteString("# path\tname\treference\n")
	for _, asset := range assets {
		b.WriteString(asset.Path)
		b.WriteByte('\t')
		b.WriteString(asset.Name)
		b.WriteByte('\t')
		b.WriteString(asset.Ref)
		b.WriteByte('\n')
	}
	if path == "" {
		fmt.Print(b.String())
		return nil
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func loadImageManifest(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var images []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			continue
		}
		images = append(images, filepath.Clean(strings.TrimSpace(parts[0])))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return images, nil
}

func findImageAsset(assets []imageAsset, name string) (imageAsset, bool) {
	want := strings.ToLower(filepath.Base(name))
	for _, asset := range assets {
		if strings.ToLower(asset.Name) == want || strings.ToLower(filepath.Base(asset.Path)) == want {
			return asset, true
		}
	}
	return imageAsset{}, false
}

func imageReference(name string) string {
	return fmt.Sprintf(`##<div class="centeredimage"><img src="images/%s" alt="%s" class="attpic" /></div>`, name, name)
}

func copyToClipboard(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func openImageFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return openSystemFile(path)
}
