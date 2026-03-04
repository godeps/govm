package offline

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed images/**
var imageFS embed.FS

var ErrImageNotFound = errors.New("offline image not found")

type ImageMetadata struct {
	Name      string
	Archive   string
	SHA256    string
	SizeBytes int64
}

// EnsureRootfs extracts an embedded offline rootfs into cache and returns the rootfs path.
func EnsureRootfs(cacheBaseDir, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("offline image name is empty")
	}
	archivePath := filepath.ToSlash(filepath.Join("images", name, "rootfs.tar.gz"))
	raw, err := fs.ReadFile(imageFS, archivePath)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrImageNotFound, name)
	}

	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:8])
	targetDir := filepath.Join(cacheBaseDir, "offline-rootfs", name+"-"+digest)
	marker := filepath.Join(targetDir, ".ready")
	if _, err := os.Stat(marker); err == nil {
		return targetDir, nil
	}

	if err := os.RemoveAll(targetDir); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	if err := extractTarGz(raw, targetDir); err != nil {
		return "", err
	}
	if err := os.WriteFile(marker, []byte("ok\n"), 0o644); err != nil {
		return "", err
	}
	return targetDir, nil
}

// List returns available embedded offline image names.
func List() ([]string, error) {
	entries, err := fs.ReadDir(imageFS, "images")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// Metadata returns embedded image metadata including sha256 and size.
func Metadata() ([]ImageMetadata, error) {
	names, err := List()
	if err != nil {
		return nil, err
	}
	out := make([]ImageMetadata, 0, len(names))
	for _, name := range names {
		archivePath := filepath.ToSlash(filepath.Join("images", name, "rootfs.tar.gz"))
		raw, err := fs.ReadFile(imageFS, archivePath)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		out = append(out, ImageMetadata{
			Name:      name,
			Archive:   archivePath,
			SHA256:    hex.EncodeToString(sum[:]),
			SizeBytes: int64(len(raw)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func extractTarGz(raw []byte, dst string) error {
	r := bytes.NewReader(raw)
	zr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := writeTarEntry(dst, hdr, tr); err != nil {
			return err
		}
	}
}

func writeTarEntry(dst string, hdr *tar.Header, tr *tar.Reader) error {
	cleanName := filepath.Clean(hdr.Name)
	if cleanName == "." || cleanName == string(filepath.Separator) {
		return nil
	}
	if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
		return fmt.Errorf("unsafe tar entry path: %s", hdr.Name)
	}
	target := filepath.Join(dst, cleanName)
	rel, err := filepath.Rel(dst, target)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("unsafe tar entry target: %s", hdr.Name)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		return os.Chmod(target, os.FileMode(hdr.Mode)&os.ModePerm)
	case tar.TypeReg, tar.TypeRegA:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		return os.Chmod(target, os.FileMode(hdr.Mode)&os.ModePerm)
	case tar.TypeSymlink:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		linkTarget, err := sanitizeTarLinkTarget(dst, target, hdr.Linkname)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Symlink(linkTarget, target)
	default:
		// Skip special files for safety.
		return nil
	}
}

func sanitizeTarLinkTarget(dst, linkPath, link string) (string, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", fmt.Errorf("unsafe symlink target: empty")
	}
	if filepath.IsAbs(link) {
		return link, nil
	}
	clean := filepath.Clean(link)
	if clean == "." {
		return "", fmt.Errorf("unsafe symlink target: %s", link)
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(linkPath), clean))
	rel, err := filepath.Rel(dst, resolved)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("unsafe symlink target: %s", link)
	}
	return clean, nil
}
