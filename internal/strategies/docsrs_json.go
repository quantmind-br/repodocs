package strategies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func DocsRSJSONEndpoint(crateName, version string) string {
	return fmt.Sprintf("https://docs.rs/crate/%s/%s/json", crateName, version)
}

func ParseRustdocJSON(data []byte) (*RustdocIndex, error) {
	var index RustdocIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse rustdoc JSON: %w", err)
	}
	return &index, nil
}

func (s *DocsRSStrategy) buildJSONEndpoint(crateName, version string) string {
	if s.baseHost != "docs.rs" {
		return fmt.Sprintf("http://%s/crate/%s/%s/json", s.baseHost, crateName, version)
	}
	return DocsRSJSONEndpoint(crateName, version)
}

func (s *DocsRSStrategy) fetchRustdocJSON(ctx context.Context, crateName, version string) (*RustdocIndex, error) {
	endpoint := s.buildJSONEndpoint(crateName, version)

	s.logger.Info().
		Str("crate", crateName).
		Str("version", version).
		Str("endpoint", endpoint).
		Msg("Fetching rustdoc JSON")

	resp, err := s.fetcher.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rustdoc JSON: %w", err)
	}

	var jsonData []byte
	contentType := resp.ContentType

	// Check for zstd compression via content-type OR magic bytes
	// Magic bytes 0x28 0xB5 0x2F 0xFD indicate zstd compression
	// This handles cached responses where content-type may be lost
	isZstd := strings.Contains(contentType, "zstd") ||
		strings.Contains(contentType, "x-zstd") ||
		strings.HasSuffix(endpoint, ".zst") ||
		(len(resp.Body) >= 4 && resp.Body[0] == 0x28 && resp.Body[1] == 0xB5 && resp.Body[2] == 0x2F && resp.Body[3] == 0xFD)

	if isZstd {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
		}
		defer decoder.Close()

		jsonData, err = decoder.DecodeAll(resp.Body, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress zstd: %w", err)
		}
	} else {
		jsonData = resp.Body
	}

	s.logger.Debug().
		Int("compressed_size", len(resp.Body)).
		Int("decompressed_size", len(jsonData)).
		Msg("Processed rustdoc JSON")

	index, err := ParseRustdocJSON(jsonData)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Int("items", len(index.Index)).
		Int("format_version", index.FormatVersion).
		Str("crate_version", index.CrateVersion).
		Msg("Parsed rustdoc index")

	return index, nil
}

const (
	MinFormatVersion = 30
	MaxFormatVersion = 60
)

func (s *DocsRSStrategy) checkFormatVersion(version int) error {
	if version < MinFormatVersion {
		return fmt.Errorf("rustdoc JSON format version %d is too old (min: %d)", version, MinFormatVersion)
	}
	if version > MaxFormatVersion {
		s.logger.Warn().Int("version", version).Msg("Untested format version, proceeding anyway")
	}
	return nil
}

func (s *DocsRSStrategy) getItemByID(index *RustdocIndex, id interface{}) *RustdocItem {
	switch v := id.(type) {
	case string:
		return index.Index[v]
	case float64:
		return index.Index[fmt.Sprintf("%.0f", v)]
	case int:
		return index.Index[fmt.Sprintf("%d", v)]
	default:
		return nil
	}
}

func (s *DocsRSStrategy) collectItems(index *RustdocIndex, opts Options) []*RustdocItem {
	var items []*RustdocItem

	for _, item := range index.Index {
		if item.CrateID != 0 {
			continue
		}

		if item.Name == nil {
			continue
		}

		if item.Docs == nil && !s.hasDocumentableChildren(item) {
			continue
		}

		if !item.IsPublic() {
			continue
		}

		if item.GetUse() != nil {
			continue
		}

		items = append(items, item)
	}

	return items
}

func (s *DocsRSStrategy) hasDocumentableChildren(item *RustdocItem) bool {
	if mod := item.GetModule(); mod != nil {
		return len(mod.Items) > 0
	}
	if trait := item.GetTrait(); trait != nil {
		return len(trait.Items) > 0
	}
	if st := item.GetStruct(); st != nil {
		return len(st.Impls) > 0
	}
	if en := item.GetEnum(); en != nil {
		return len(en.Variants) > 0 || len(en.Impls) > 0
	}
	return false
}
