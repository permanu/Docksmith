# detect

Detect the framework of any project directory.

## Usage

```bash
go run . /path/to/your/project
```

## What it does

Analyzes the project files (package.json, go.mod, requirements.txt, etc.) and
reports the detected framework, default port, and build/start commands.

## Example output

```
Framework: express
Port:      3000
Build:     npm run build
Start:     node server.js
```
