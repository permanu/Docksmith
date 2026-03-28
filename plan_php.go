package docksmith

import (
	"strconv"
	"strings"
)

// planPHP builds a BuildPlan for PHP applications.
// WordPress: single-stage php-apache base.
// Laravel: two-stage builder + fpm runtime.
// Symfony/Slim/plain: single-stage apache with composer install.
func planPHP(fw *Framework) (*BuildPlan, error) {
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

func planPHPWordPress(phpVer string) (*BuildPlan, error) {
	// WordPress uses a dedicated Docker Hub image with its own tag format.
	image := "wordpress:php" + phpVer + "-apache"
	return &BuildPlan{
		Framework: "wordpress",
		Expose:    80,
		Stages: []Stage{
			{
				Name: "runtime",
				From: image,
				Steps: []Step{
					{Type: StepCopy, Args: []string{".", "/var/www/html/"}},
					{Type: StepExpose, Args: []string{"80"}},
					{Type: StepCmd, Args: []string{"apache2-foreground"}},
				},
			},
		},
		Dockerignore: []string{".git", "*.log"},
	}, nil
}

func planPHPLaravel(fw *Framework, phpVer string, port int) (*BuildPlan, error) {
	fpmImage := ResolveDockerTag("php", phpVer)
	phpExts := "apk add --no-cache libzip-dev icu-dev postgresql-dev oniguruma-dev && " +
		"docker-php-ext-install pdo_mysql pdo_pgsql zip intl bcmath opcache && " +
		"(docker-php-ext-install mbstring 2>/dev/null || true)"

	builder := Stage{
		Name: "builder",
		From: fpmImage,
		Steps: []Step{
			{Type: StepRun, Args: []string{phpExts}},
			{Type: StepRun, Args: []string{
				"curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer",
			}},
			{Type: StepWorkdir, Args: []string{"/app"}},
			{Type: StepCopy, Args: []string{"composer.json", "composer.lock*", "./"}},
			{
				Type:       StepRun,
				Args:       []string{"composer install --no-dev --optimize-autoloader --no-scripts"},
				CacheMount: &CacheMount{Target: "/root/.composer/cache"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
			{Type: StepRun, Args: []string{
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

	runtime := Stage{
		Name: "runtime",
		From: fpmImage,
		Steps: []Step{
			{Type: StepRun, Args: []string{phpExts}},
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app", Dst: "."},
				Link:     true,
			},
			{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: StepCmd, Args: startArgs},
		},
	}

	addNonRootUser(&runtime, "www-data")
	addHealthcheck(&runtime, "php", port)

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "vendor", "node_modules", "storage/logs"},
	}, nil
}

func planPHPComposer(fw *Framework, phpVer string, port int) (*BuildPlan, error) {
	// Symfony, Slim and plain PHP all use the apache image.
	apacheImage := ResolveDockerTag("php-apache", phpVer)
	phpExts := "docker-php-ext-install pdo_mysql pdo_pgsql zip opcache && " +
		"(docker-php-ext-install mbstring intl 2>/dev/null || true)"

	hasComposer := fw.Name == "symfony" || fw.Name == "slim"

	steps := []Step{
		{Type: StepRun, Args: []string{
			"apt-get update -qq && apt-get install -y --no-install-recommends " +
				"unzip libpq-dev libzip-dev libicu-dev libonig-dev && " +
				"rm -rf /var/lib/apt/lists/* && " + phpExts,
		}},
	}

	if hasComposer {
		steps = append(steps,
			Step{Type: StepRun, Args: []string{
				"curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer",
			}},
			Step{Type: StepWorkdir, Args: []string{"/var/www/html"}},
			Step{Type: StepCopy, Args: []string{"composer.json", "composer.lock*", "./"}},
			Step{
				Type:       StepRun,
				Args:       []string{"composer install --no-dev --optimize-autoloader --no-scripts"},
				CacheMount: &CacheMount{Target: "/root/.composer/cache"},
			},
			Step{Type: StepCopy, Args: []string{".", "."}},
			Step{Type: StepRun, Args: []string{"composer dump-autoload --optimize"}},
		)
	} else {
		steps = append(steps,
			Step{Type: StepWorkdir, Args: []string{"/var/www/html"}},
			Step{Type: StepCopy, Args: []string{".", "."}},
		)
	}

	startCmd := fw.StartCommand
	if startCmd == "" {
		startCmd = "apache2-foreground"
	}
	startArgs := strings.Fields(startCmd)

	steps = append(steps,
		Step{Type: StepRun, Args: []string{
			"sed -i 's|/var/www/html|/var/www/html/public|g' " +
				"/etc/apache2/sites-available/000-default.conf && " +
				"a2enmod rewrite || true",
		}},
		Step{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
		Step{Type: StepCmd, Args: startArgs},
	)

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{{Name: "runtime", From: apacheImage, Steps: steps}},
		Expose:       port,
		Dockerignore: []string{".git", "vendor", "*.log"},
	}, nil
}
