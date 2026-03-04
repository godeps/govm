package client

import "github.com/godeps/govm/internal/offline"

// ListOfflineImages returns embedded offline image names.
func ListOfflineImages() ([]string, error) {
	return offline.List()
}

type OfflineImageMetadata struct {
	Name      string
	Archive   string
	SHA256    string
	SizeBytes int64
}

// ListOfflineImageMetadata returns metadata for embedded offline image bundles.
func ListOfflineImageMetadata() ([]OfflineImageMetadata, error) {
	m, err := offline.Metadata()
	if err != nil {
		return nil, err
	}
	out := make([]OfflineImageMetadata, 0, len(m))
	for _, v := range m {
		out = append(out, OfflineImageMetadata{
			Name:      v.Name,
			Archive:   v.Archive,
			SHA256:    v.SHA256,
			SizeBytes: v.SizeBytes,
		})
	}
	return out, nil
}
