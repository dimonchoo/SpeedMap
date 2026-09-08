package wpexport

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
)

//go:embed apply_template.php
var applyTemplate string

//go:embed rollback_template.php
var rollbackTemplate string


func BuildApplyPHP(domain string, cfg config.ScanConfig, images []analytics.AggregatedImage, wordpressPath string) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}

	heavy := CollectHeavyImages(images)
	if len(heavy) == 0 {
		return "", fmt.Errorf("no heavy convertible images in scan (raise threshold or rescan)")
	}
	return BuildApplyPHPFromManifest(domain, cfg, wpPath, heavy)
}

// BuildApplyPHPFromManifest embeds an already-filtered image list.
func BuildApplyPHPFromManifest(domain string, cfg config.ScanConfig, wordpressPath string, images []ManifestImage) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}
	if len(images) == 0 {
		return "", fmt.Errorf("no images for WP apply PHP")
	}

	q := int(cfg.NormalizedWebPQuality())
	m := Manifest{
		Domain:        domain,
		Generated:     time.Now().UTC().Format(time.RFC3339),
		Quality:       q,
		WordPressPath: wpPath,
		Images:        images,
	}

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}

	php := strings.Replace(applyTemplate, "{{SPEEDMAP_MANIFEST_JSON}}", string(raw), 1)
	if php == applyTemplate {
		return "", fmt.Errorf("template placeholder {{SPEEDMAP_MANIFEST_JSON}} not found")
	}
	php = strings.ReplaceAll(php, "{{WORDPRESS_PATH}}", wpPath)
	return php, nil
}

// BuildRollbackPHP returns a standalone rollback eval-file script.
func BuildRollbackPHP(wordpressPath string) string {
	return strings.ReplaceAll(rollbackTemplate, "{{WORDPRESS_PATH}}", strings.TrimSpace(wordpressPath))
}

// ManifestImagesFromWritten strips payloads for PHP embedding.
