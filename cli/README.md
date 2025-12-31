# Frags CLI

A command-line interface for [Frags](https://github.com/theirish81/frags) to:
* process session files using AI models from Google Gemini, Ollama or ChatGPT.
* render YAML or JSON data files using a Go template..
* print the current configuration.
* ask questions to the AI using the current Frags settings and tools (for debugging purposes).
* run scripts on the scripting engine in the Frags context. (for debugging purposes).

## Description

This tool allows you to run sessions defined in YAML files. It processes the data through a configured AI model
(Gemini or Ollama) and outputs the results in your chosen format: YAML, JSON, or a custom text file using a Go template.
It also includes a `render` command to format existing JSON/YAML data with a template.

## Configuration

The first time you run the CLI, it will create a `.env` file for you. You will need to fill in the required values
before you can use the tool.

The CLI can be configured using the following environment variables:

-   `AI_ENGINE`: The AI engine to use. Can be `gemini`, `ollama` or `chatgpt`. If not set, the tool will try to guess
    based on the other variables present.

### Gemini Configuration
-   `GEMINI_SERVICE_ACCOUNT_PATH`: Path to your Google Cloud service account JSON file.
-   `GEMINI_PROJECT_ID`: Your Google Cloud project ID.
-   `GEMINI_LOCATION`: The Google Cloud region for the Gemini API (e.g., `us-central1`).

### Ollama Configuration
-   `OLLAMA_BASE_URL`: The base URL for your Ollama instance (e.g., `http://localhost:11434`).
-   `NUM_PREDICT`: The number of tokens to predict.

### ChatGPT Configuration
-   `CHATGPT_API_KEY`: Your OpenAI API key.
-   `CHATGPT_BASE_URL`: The base URL for the ChatGPT API, usually `https://api.openai.com/v1`.

### Model & Runner Configuration
-   `MODEL`: The specific model to use (e.g., `gemini-2.5-flash` for Gemini, `qwen3:latest` for Ollama).
-   `TEMPERATURE`, `TOP_K`, `TOP_P`: Model-specific parameters to control creativity and randomness.
-   `PARALLEL_WORKERS`: The number of parallel workers to use for processing. Defaults to 1.

### Example `.env` file:

```
# The AI engine to use: "gemini", "ollama" or "chatgpt"
# If left blank, the CLI will try to guess based on the filled-in values.
AI_ENGINE="ollama"

# --- Gemini Configuration ---
# GEMINI_SERVICE_ACCOUNT_PATH="/path/to/your/service-account.json"
# GEMINI_PROJECT_ID="your-gcp-project-id"
# GEMINI_LOCATION="us-central1"

# --- Ollama Configuration ---
OLLAMA_BASE_URL="http://localhost:11434"

# --- ChatGPT Configuration ---
# CHATGPT_API_KEY="your-openai-api-key"
# CHATGPT_BASE_URL="https://api.openai.com/v1"

# --- Model & Runner Configuration ---
# For Gemini, e.g., "gemini-2.5-flash"
# For Ollama, e.g., "qwen3:latest"
# For ChatGPT, e.g., "gpt-4-turbo"
MODEL="qwen3:latest"

PARALLEL_WORKERS=4

# TEMPERATURE=0.3
# TOP_K=32
# TOP_P=1.0
# NUM_PREDICT=1024
```

## Download
You can find pre-built binaries for Windows, Linux, and macOS on the
[Releases page](https://github.com/theirish81/frags/releases). Remember these binaries are not signed, so you may need
to disable your security settings to run them.

## Commands

### run

Run a session file through the configured AI model.

**Usage:**
`./cli run <path/to/session.yaml> [flags]`

**Flags:**

-   `--format, -f`: Specifies the output format. Options are `yaml` (default), `json`, or `template`.
-   `--output, -o`: Specifies a file to write the output to. If omitted, the output is printed to the console.
-   `--template, -t`: If `format` is `template`, this flag is required. It specifies the path to the Go template file
    to use for formatting the output.
-   `--param, -p`: Can be used multiple times. Pass key-value pairs (`key=value`) to be used as dynamic variables in 
    your session prompts. These variables will replace placeholders like `{{.key}}` in your prompt.

**Examples:**

1.  **Run a session and print YAML output to the console:**
    ```sh
    ./cli run session.yaml
    ```

2.  **Run a session and save the output as a JSON file:**
    ```sh
    ./cli run session.yaml -f json -o output.json
    ```

3.  **Run a session and format the output using a template:**
    ```sh
    ./cli run session.yaml -f template -t story_template.md -o story.md
    ```

4.  **Run a session with dynamic variables:**
    *Assume `session.yaml` contains a prompt like: `Generate a story about {{.character}} in a {{.setting}}.`*
    ```sh
    ./cli run session.yaml -p character="a brave knight" -p setting="mystical forest"
    ```

### ask

Ask a question to the AI, using the current Frags settings and tools.
**Note:** this is different than asking something to a ChatBot. This command reflects the complexity and limitations
of Frags' structured data methodology. This command exist for debugging purposes.

**Usage:**
`./cli ask <prompt> [flags]`

**Flags:**

-   `--pre-prompt, -p`: A prompt to run before the AI prompt.
-   `--system-prompt, -s`: The system prompt.
-   `--upload, -u`: File path to upload (can be specified multiple times).
-   `--internet-search, -i`: Enable internet search.
-   `--tools, -t`: Enable tools.

### script

Run a script (JavaScript) on the scripting engine in the Frags context. This is useful for debugging purposes.

**Usage:**
`./cli script <path/to/script.js> [flags]`

**Flags:**

-   `--format, -f`: Specifies the output format. Options are `yaml` (default), `json`, or `template`.
-   `--output, -o`: Specifies a file to write the output to. If omitted, the output is printed to the console.

### config

Prints the current configuration.

**Usage:**
`./cli config`

### render

Render a YAML or JSON data file using a Go template. This is useful for re-formatting previous outputs without
re-running the AI session.

**Usage:**
`./cli render <path/to/data.yaml|json> [flags]`

**Flags:**

-   `--template, -t`: **(Required)** Path to the Go template file.
-   `--output, -o`: Specifies a file to write the output to. If omitted, the output is printed to the console.

**Example:**

```sh
./cli render output.json --template my_report.templ.md --output final_report.md
```
