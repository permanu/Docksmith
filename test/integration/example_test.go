package integration_test

import (
	"fmt"

	"github.com/permanu/docksmith"
)

func ExampleDetect() {
	fw, err := docksmith.Detect("../../testdata/fixtures/with-dockerfile")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(fw.Name)
	// Output: dockerfile
}

func ExampleResolveDockerTag() {
	tag := docksmith.ResolveDockerTag("node", "22")
	fmt.Println(tag)
	// Output: node:22-alpine
}

func ExampleFrameworkDefaults() {
	build, start := docksmith.FrameworkDefaults("django")
	fmt.Println(build)
	fmt.Println(start)
	// Output:
	// pip install -r requirements.txt
	// gunicorn config.wsgi:application --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2
}
