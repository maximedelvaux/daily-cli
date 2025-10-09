# Daily Task CLI

[![Go Version](https://img.shields.io/badge/go-1.23%2B-blue)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://makeapullrequest.com)

A simple, interactive CLI tool written in Go to help you track your daily tasks, add quick notes, and review your productivity.

## Installation

Download the latest binary for your platform from the [Releases](https://github.com/maximedelvaux/daily-cli/releases) page:

- **Windows**: `daily-task.exe`
- **Linux (amd64)**: `daily-task-linux`
- **macOS (Intel)**: `daily-task-mac`
- **macOS (Apple Silicon)**: `daily-task-mac-arm64`

### Linux/macOS Setup
After downloading, make the binary executable:
```bash
chmod +x daily-task-linux  # or daily-task-mac / daily-task-mac-arm64
```

Optionally, move it to a directory in your PATH:
```bash
sudo mv daily-task-linux /usr/local/bin/daily-task
```

## Features
- Add, list, edit, and delete daily tasks
- Track estimated and actual time for each task
- Mark tasks as pending, started, done, or cancelled
- Add quick notes for each day (with nano-style editing)
- Review and edit notes for today or any specific day
- Edit yesterday's notes with a single command
- Interactive shell mode with autocomplete and help

## Usage

### Windows
- Use the `daily-task.exe` binary

### Linux
- Use the `daily-task-linux` binary (built for amd64)
- Make it executable: `chmod +x daily-task-linux`
- Run it: `./daily-task-linux add`

### Add a task for today
```
daily-task.exe add
# or on Linux
./daily-task-linux add
```

### Add a task for tomorrow
```
daily-task.exe addt
./daily-task-linux addt
```

### List and edit today's tasks
```
daily-task.exe ls
./daily-task-linux ls
```

### List and edit tomorrow's tasks
```
daily-task.exe lst
./daily-task-linux lst
```

### Add a note for today
```
daily-task.exe note Your note text here
./daily-task-linux note Your note text here
```

### Show today's notes
```
daily-task.exe note
./daily-task-linux note
```

### Edit today's notes in nano
```
daily-task.exe note edit
./daily-task-linux note edit
```

### Edit notes for a specific day
```
daily-task.exe note edit YYYY-MM-DD
./daily-task-linux note edit YYYY-MM-DD
```

### Edit yesterday's notes
```
daily-task.exe note edit-yesterday
./daily-task-linux note edit-yesterday
```

### Interactive Shell Mode
```
daily-task.exe shell
./daily-task-linux shell
```

Type `help` in shell mode for a list of commands and usage examples.

## Why?
This repo is public and does not contain your personal tasks or notes. Use it as a template or starting point for your own daily productivity CLI.

## Requirements
- Go 1.23+ (if you want to build from source)
- [nano](https://www.nano-editor.org/) (for note editing, or change the editor in code)

## License
MIT
