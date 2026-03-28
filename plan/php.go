package plan

import (
	"github.com/permanu/docksmith/core"
	"strconv"
	"strings"
)

// planPHP builds a BuildPlan for PHP applications.
// WordPress: single-stage php-apache base.
// Laravel: two-stage builder + fpm runtime.
// Symfony/Slim/plain: single-stage apache with composer install.
func planPHP(fw *core.Framework) (*core.BuildPlan, error) {
	phpVer := fw.PHPVersion
	if phpVer == "" {
		phpVer = "8.3"
	}
	port := fw.Port
	if port == 0 {
		port = 80
	}

	switch fw.Name {
	case "wordpress":
		return planPHPWordPress(phpVer)
	case "laravel":
		return planPHPLaravel(fw, phpVer, port)
	default:
		// symfony, slim, php
		return planPHPComposer(fw, phpVer, port)
	}
}

func planPHPWordPress(phpVer string) (*core.BuildPlan, error) {
	// WordPress uses a dedicated Docker Hub image with its own tag format.
	image := "wordpress:php" + phpVer + "-apache"
	return &core.BuildPlan{
		Framework: "wordpress",
		Expose:    80,
		Stages: []core.Stage{
			{
				Name: "runtime",
				From: image,
				Steps: []core.Step{
					{Type: core.StepCopy, Args: []string{".", "/var/www/html/"}},
					{Type: core.StepExpose, Args: []string{"80"}},
					{Type: core.StepCmd, Args: []string{"apache2-foreground"}},
				},
			},
		},
		Dockerignore: []string{".git", "*.log"},
	}, nil
}

func planPHPLaravel(fw *core.Framework, phpVer string, port int) (*core.BuildPlan, error) {
	fpmImage := ResolveDockerTag("php", phpVer)
	phpExts := "apk add --no-cache libzip-dev icu-dev postgresql-dev oniguruma-dev && " +
		"docker-php-ext-install pdo_mysql pdo_pgsql zip intl bcmath opcache && " +
		"(docker-php-ext-install mbstring 2>/dev/null || true)"

	builder := core.Stage{
		Name: "builder",
		From: fpmImage,
		Steps: []core.Step{
			{Type: core.StepRun, Args: []string{phpExts}},
			{Type: core.StepRun, Args: []string{
				"curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer",
			}},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{Type: core.StepCopy, Args: []string{"composer.json", "composer.lock*", "./"}},
			{
				Type:       core.StepRun,
				Args:       []string{"composer install --no-dev --optimize-autoloader --no-scripts"},
				CacheMount: &core.CacheMount{Target: "/root/.composer/cache"},
			},
			{Type: core.StepCopy, Args: []string{".", "."}},
			{Type: core.StepRun, Args: []string{
				"composer dump-autoload --optimize && " +
					"php artisan config:cache 2>/dev/null; " +
					"php artisan route:cache 2>/dev/null; " +
					"php artisan view:cache 2>/dev/null; true",
			}},
		},
	}

	startCmd := fw.StartCommand
	if startCmd == "" {
		startCmd = "php artisan serve --host=0.0.0.0 --port=" + strconv.Itoa(port)
	}
	startArgs := strings.Fields(startCmd)

	runtime := core.Stage{
		Name: "runtime",
		From: fpmImage,
		Steps: []core.Step{
			{Type: core.StepRun, Args: []string{phpExts}},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/app", Dst: "."},
				Link:     true,
			},
			{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: core.StepCmd, Args: startArgs},
		},
	}

	// Laravel needs writable storage and bootstrap/cache dirs for the non-root user.
	runtime.Steps = append(runtime.Steps, core.Step{
		Type: core.StepRun,
		Args: []string{"chown -R www-data:www-data /app/storage /app/bootstrap/cache"},
	})

	addNonRootUser(&runtime, "www-data")
	addHealthcheck(&runtime, "php", port)

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "vendor", "node_modules", "storage/logs"},
	}, nil
}

func planPHPComposer(fw *core.Framework, phpVer string, port int) (*core.BuildPlan, error) {
	// Symfony, Slim and plain PHP all use the apache image.
	apacheImage := ResolveDockerTag("php-apache", phpVer)
	phpExts := "docker-php-ext-install pdo_mysql pdo_pgsql zip opcache && " +
		"(docker-php-ext-install mbstring intl 2>/dev/null || true)"

	hasComposer := fw.Name == "symfony" || fw.Name == "slim"

	steps := []core.Step{
		{Type: core.StepRun, Args: []string{
			"apt-get update -qq && apt-get install -y --no-install-recommends " +
				"unzip libpq-dev libzip-dev libicu-dev libonig-dev && " +
				"rm -rf /var/lib/apt/lists/* && " + phpExts,
		}},
	}

	if hasComposer {
		steps = append(steps,
			core.Step{Type: core.StepRun, Args: []string{
				"curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer",
			}},
			core.Step{Type: core.StepWorkdir, Args: []string{"/var/www/html"}},
			core.Step{Type: core.StepCopy, Args: []string{"composer.json", "composer.lock*", "./"}},
			core.Step{
				Type:       core.StepRun,
				Args:       []string{"composer install --no-dev --optimize-autoloader --no-scripts"},
				CacheMount: &core.CacheMount{Target: "/root/.composer/cache"},
			},
			core.Step{Type: core.StepCopy, Args: []string{".", "."}},
			core.Step{Type: core.StepRun, Args: []string{"composer dump-autoload --optimize"}},
		)
	} else {
		steps = append(steps,
			core.Step{Type: core.StepWorkdir, Args: []string{"/var/www/html"}},
			core.Step{Type: core.StepCopy, Args: []string{".", "."}},
		)
	}

	startCmd := fw.StartCommand
	if startCmd == "" {
		startCmd = "apache2-foreground"
	}
	startArgs := strings.Fields(startCmd)

	steps = append(steps,
		core.Step{Type: core.StepRun, Args: []string{
			"sed -i 's|/var/www/html|/var/www/html/public|g' " +
				"/etc/apache2/sites-available/000-default.conf && " +
				"a2enmod rewrite || true",
		}},
		core.Step{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
		core.Step{Type: core.StepCmd, Args: startArgs},
	)

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{{Name: "runtime", From: apacheImage, Steps: steps}},
		Expose:       port,
		Dockerignore: []string{".git", "vendor", "*.log"},
	}, nil
}
