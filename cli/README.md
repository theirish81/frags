# Frags CLI

A command-line interface for [Frags](https://github.com/theirish81/frags) to process session files using Google's Gemini AI.

## Description

This tool allows you to run sessions defined in YAML files. It processes the data through the Gemini AI model and outputs the results in your chosen format: YAML, JSON, or a custom format using a Go template.

## Configuration

The CLI requires a `.env` file in the same directory with the following variables for connecting to Google Gemini:

-   `GEMINI_SERVICE_ACCOUNT_PATH`: Path to your Google Cloud service account JSON file.
-   `GEMINI_PROJECT_ID`: Your Google Cloud project ID.
-   `GEMINI_LOCATION`: The Google Cloud region for the Gemini API (e.g., `us-central1`).
-   `PARALLEL_WORKERS`: (Optional) The number of parallel workers to use for processing. Defaults to 1.

### Example `.env` file:

```
GEMINI_SERVICE_ACCOUNT_PATH="/path/to/your/service-account.json"
GEMINI_PROJECT_ID="your-gcp-project-id"
GEMINI_LOCATION="us-central1"
PARALLEL_WORKERS=4
```

## Installation

To build the CLI, you need to have Go installed. You can build the executable with:

```sh
go build .
```

## Usage

### Flags

-   `--format, -f`: Specifies the output format. Options are `yaml` (default), `json`, or `template`.
-   `--output, -o`: Specifies a file to write the output to. If omitted, the output is printed to the console.
-   `--template, -t`: If `format` is `template`, this flag is required. It specifies the path to the Go template file to use for formatting the output.

### Examples

**1. Run a session and print YAML output to the console:**

```sh
./cli run session.yaml
```

**2. Run a session and save the output as a JSON file:**

```sh
./cli run session.yaml -f json -o output.json
```

**3. Run a session and format the output using a template:**

```sh
./cli run session.yaml -f template -t story_template.md -o story.md
```
