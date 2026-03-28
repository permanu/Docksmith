package detect

import (
	"github.com/permanu/docksmith/core"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Most specific first: Laravel > WordPress > Symfony > Slim > plain PHP.
	RegisterDetector("php", detectPlainPHP)
	RegisterDetector("slim", detectSlim)
	RegisterDetector("symfony", detectSymfony)
	RegisterDetector("wordpress", detectWordPress)
	RegisterDetector("laravel", detectLaravel)
}

func detectLaravel(dir string) *core.Framework {
	if hasFile(dir, "artisan") && hasFile(dir, "composer.json") {
		return &core.Framework{
			Name:         "laravel",
			BuildCommand: "composer install --no-dev --optimize-autoloader",
			StartCommand: "php artisan serve --host=0.0.0.0 --port=8000",
			Port:         8000,
			PHPVersion:   detectPHPVersion(dir),
		}
	}
	return nil
}

func detectWordPress(dir string) *core.Framework {
	if hasFile(dir, "wp-config.php") || dirExists(filepath.Join(dir, "wp-content")) {
		return &core.Framework{
			Name:         "wordpress",
			BuildCommand: "",
			StartCommand: "apache2-foreground",
			Port:         80,
			PHPVersion:   detectPHPVersion(dir),
		}
	}
	return nil
}

func detectSymfony(dir string) *core.Framework {
	composerPath := filepath.Join(dir, "composer.json")
	if hasFile(dir, "symfony.lock") || hasFile(dir, "config/bundles.php") ||
		(hasFile(dir, "composer.json") && fileContains(composerPath, "symfony/framework-bundle")) {
		return &core.Framework{
			Name:         "symfony",
			BuildCommand: "composer install --no-dev --optimize-autoloader",
			StartCommand: "php -S 0.0.0.0:8000 -t public",
			Port:         8000,
			PHPVersion:   detectPHPVersion(dir),
		}
	}
	return nil
}

func detectSlim(dir string) *core.Framework {
	composerPath := filepath.Join(dir, "composer.json")
	if hasFile(dir, "composer.json") && fileContains(composerPath, "slim/slim") {
		return &core.Framework{
			Name:         "slim",
			BuildCommand: "composer install --no-dev --optimize-autoloader",
			StartCommand: "php -S 0.0.0.0:8000 -t public",
			Port:         8000,
			PHPVersion:   detectPHPVersion(dir),
		}
	}
	return nil
}

func detectPlainPHP(dir string) *core.Framework {
	if hasFile(dir, "index.php") {
		return &core.Framework{
			Name:         "php",
			BuildCommand: "",
			StartCommand: "apache2-foreground",
			Port:         80,
			PHPVersion:   detectPHPVersion(dir),
		}
	}
	return nil
}

func detectPHPVersion(dir string) string {
	composerPath := filepath.Join(dir, "composer.json")
	if data, err := os.ReadFile(composerPath); err == nil {
		var composer struct {
			Require map[string]string `json:"require"`
		}
		if err := json.Unmarshal(data, &composer); err == nil {
			if constraint, ok := composer.Require["php"]; ok {
				if v := parsePHPConstraint(constraint); v != "" {
					return v
				}
			}
		}
	}
	phpVersionPath := filepath.Join(dir, ".php-version")
	if data, err := os.ReadFile(phpVersionPath); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			parts := strings.SplitN(v, ".", 3)
			if len(parts) >= 2 {
				return parts[0] + "." + parts[1]
			}
			return v
		}
	}
	return "8.3"
}

func parsePHPConstraint(constraint string) string {
	c := strings.TrimSpace(constraint)
	if c == "" || c == "*" {
		return ""
	}
	c = strings.TrimLeft(c, "^~>=<! ")
	c = strings.TrimSuffix(c, ".*")
	for _, sep := range []string{"||", "|", " ", ","} {
		if idx := strings.Index(c, sep); idx > 0 {
			c = c[:idx]
		}
	}
	parts := strings.SplitN(c, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return ""
}
