# Frags (WORK IN PROGRESS!)

**Frags** is a Go library that allows you to define and execute complex, multi-step data extraction plans using AI.
It works by fragmenting a large JSON Schema into smaller, sequential "sessions" and "phases." This enables you to build
robust, iterative interactions with AI models, making it ideal for tasks like structured data extraction from documents,
progressive data collection, or any scenario where you need to manage a complex, stateful conversation with
an AI.

## The Problem

When interacting with AI APIs that support structured JSON output, you may encounter scenarios where:

- The complete output schema is too large, potentially hitting output token limits in a single response
- You want to process information incrementally without maintaining multiple separate schemas
- You want to define multiple, distinct AI interactions that operate on the same overall data model, but without
  polluting a single global context and therefore poisoning the focus.
- Sequential data collection makes more sense than requesting everything at once

## The Solution

Frags allows you:

* Define a single, large schema that can be broken down into smaller, purpose-oriented chunks
* Define multiple, distinct AI **sessions** that operate on the same overall data model, each one with its own context
* Define sub-objectives in each session (phases) which get translated in sequential AI interactions, to reduce the
  the size of each output and ensure JSON schema compliance
* Define dependencies between sessions, allowing you to build workflows that run in sequence or in parallel
* Define custom components that can be reused across sessions
* Define custom AI clients that support file uploads and JSON schema responses
* Load resources from the file system or any other source
* Run multiple sessions in parallel to improve performance

And much more

## Installation

```bash
go get github.com/theirish81/frags
```

## Usage

## Designing the plan

The plan is the central piece of the puzzle. It defines the overall data model, the AI interactions, and the
dependencies between them. You can easily do that by code, but it's way more readable to define it in a YAML file.

### Step 1: Define the schema

The schema defines the overall data model. It's a standard JSON Schema, with the addition of a few tags that allow you
to define the structure of the AI interactions.
**Example:**
```yaml
schema:
  type: object
  required:
    - name
    - email
    - street_address
    - city
    - country
    - product_name
    - rating
    - review_summary
    - full_review_text
  properties:
    # Session 'user_profile'
    name:
      type: string
      description: "The user's full name."
      x-session: user_profile
      x-phase: 0
    email:
      type: string
      description: "The user's email address."
      x-session: user_profile
      x-phase: 0
    street_address:
      type: string
      description: "The user's street address."
      x-session: user_profile
      x-phase: 1
    city:
      type: string
      description: "The city."
      x-session: user_profile
      x-phase: 1
    country:
      type: string
      description: "The country."
      x-session: user_profile
      x-phase: 1

    # Session 'product_review'
    product_name:
      type: string
      description: "The name of the product being reviewed."
      x-session: product_review
      x-phase: 0
    rating:
      type: number
      description: "The user's rating, from 1 to 5."
      minimum: 1
      maximum: 5
      x-session: product_review
      x-phase: 0
    review_summary:
      type: string
      description: "A one-sentence summary of the review."
      x-session: product_review
      x-phase: 1
    full_review_text:
      type: string
      description: "The full text of the product review."
      x-session: product_review
      x-phase: 2
```
**Please Notice:**
- The `x-session` tag is used to define the name of the session.
- The `x-phase` tag is used to define the index of the phase within the session.


### Step 2: Define the Sessions

The sessions define the AI interactions. As before, you can define them by code, but it's much more readable to do it
in YAML. As both the schema and the sessions unmarshal to the same Go struct (SessionManager), you will integrate
the same YAML file.

**Example:**
```yaml
sessions:
  user_profile:
    prompt: "Extract the user's primary details form the provided document"
    nextPhasePrompt: "Also these secondary details"
    resources:
      - identifier: user_text.txt
  product_review:
    prompt: "Extract the required information from the provided document"
    nextPhasePrompt: "Also extract these items"
    resources:
      - identifier: product_details.pdf
```

**Please Notice:**
- The `prompt` field defines the prompt that will be sent to the AI.
- The `nextPhasePrompt` field defines the prompt that will be sent to the AI after each phase.
- The `resources` field defines the files that will be uploaded to the AI.
- The name of the session is the same as the `x-session` tag value.
- This example will not include features like dependencies or context carry-over


### Step 3: Run the plan

The `Runner` is the highest-level abstraction. It automates the entire workflow:

1.  Loads a `SessionManager`.
2.  Takes a `ResourceLoader` to load the files/data required by each session.
3.  Takes an `Ai` interface implementation to make the actual calls to your AI.
4.  Runs each session, automatically iterating through its local phases. It uses the session's `prompt` for the first
    phase and the `nextPhasePrompt` for all subsequent phases.
5.  Unmarshals the structured JSON results from the AI into a final Go struct.

**Example:**
```go
func main() {
    sessionData, _ := os.ReadFile("test_data/sessions.yaml")
    mgr := NewSessionManager()
    err := mgr.FromYAML(sessionData)
    assert.Nil(t, err)
    ai := NewDummyAi()
    runner := NewRunner[T](mgr, NewDummyResourceLoader(), ai)
    out, err := runner.Run(nil)
	// do something with the output
}
```

**Session Parallelism**
Sessions can run in parallel, which can significantly improve performance. As a default, however, the runner will run
sessions sequentially. To enable parallelism, use the `WithSessionsWorkers(int)` option when creating the runner.

**Reusability**
The same instance of a runner can be used multiple times, however, it can work on a task at a time and will return
an error if you call `Run` before the previous task has completed. Each instance of Runner is totally independent of
each other and thread safe. 

## Advanced Features

### Reusable components
In a plan you can define reusable components. These components can be used in multiple sessions.
The `components` section in the plan YAML file defines `prompts` and `schemas`.
**Example:**
```yaml
components:
  prompts:
    user_profile_prompt: "Extract the user information"
  schemas:
    user_profile_schema:
      type: object
      required:
        - name
        - email
      properties:
        name:
          type: string
        email:
          type: string
```
Prompts can be referenced using a Go template notation, while schemas can be referenced using the `$ref` notation.

### Parametrization / Templatization
Ideally a Frags plan does not solve a single problem, but rather a set of related problems. To make it easier to reuse
the same plan for different scenarios, you can use all sorts of parameters and templating to customize the plan for
each scenario.

#### The Scope
The scope is a map of key-value pairs. Its root level is made of:
* `context`: the accumulated data from all the previous sessions
* `params`: the parameters passed to the runner.
* `vars`: local variables defined in the session
* `components`: reusable components defined in the plan
You can reference these and any sub-property using the dot notation


#### Templates
All prompts support [Go template](https://pkg.go.dev/text/template) syntax, allowing for dynamic prompts. In combination
with the scope, you can express prompts such as: `Find info about the car {{ .params.car_model }}` or
`Obtain reviews for the product {{ .context.product_name }}`

#### Session vars
Always from the point of view of reusability, you can define session-bound variables. This is particularly useful in
combination with reusable prompts.

**Example:**
```yaml
components:
  prompts:
    user_profile_prompt: "Extract the user information for the user {{ .vars.user_id }}"
sessions:
  user_profile:
    prompt: "{{ .components.Prompts.user_profile_prompt }}"
    vars:
      user_id: 12345
```
**Important:** notice the capital P in Prompts

#### Plan parameters
When running a plan, you can pass parameters to the runner. These parameters are available in the scope as `params`.
```go
    runner := NewRunner[T](mgr, NewDummyResourceLoader(), ai)
	runner.Run(map[string]interface{}{
		"car_model": "BMW",
	})
```
and then refer to them in the prompt: `Find info about the car {{ .params.car_model }}`


### Session Dependencies (`dependsOn`)
You can define dependencies between sessions using the `dependsOn` property. This allows you to create workflows where
a session will only run after other sessions have successfully completed. `dependsOn` is a list of dependencies, and
all dependencies must be met for the session to run.

A dependency can be on another session, or it can be a conditional expression.

-   **Session Dependency**: To make a session wait for another, specify the session name.
-   **Conditional Expression**: You can add a `expression` to a dependency. This expression is evaluated against the
    data extracted so far, and the session will only run if the expression evaluates to `true`. Expressions are written 
    in a simple expression language (see [antonmedv/expr](https://github.com/antonmedv/expr)).

A session may depend on another for two main reasons:
- Whether it will execute or not depends on the outcome of the previous session.
- The data extracted by the previous session may be required by the current session. 
- Context carry-over
- 
In this example, `end_session` depends on `session_one` and `session_two`. It will only run after both are complete,
and only if the `animal1` field has been extracted in `session_one`.

```yaml
sessions:
  session_one:
    prompt: answer the question
  session_two:
    prompt: answer the question
  end_session:
    context: true
    dependsOn:
      - session: session_one
        expression: len(context.animal1) > 0
      - session: session_two
    prompt: answer the question based on the answers to the previous questions
```

### Context Carry-over (`context`)
When a session depends on others, it can be useful to access the data extracted by its dependencies. By setting
`context: true` on a session, you make the LLM aware of the already extracted data.

### Retries (`attempts`)
By default, a phase will be attempted once. You can add resilience to your workflow by specifying the `attempts`
property on a session. This will cause each phase within that session to be retried the specified number of times if
the AI call fails or returns invalid data.

```yaml
sessions:
  robust_session:
    prompt: "This is an important prompt."
    attempts: 3
```

### Pre-Prompt (`prePrompt`)
Each session can optionally define a `prePrompt`. This is a special prompt that runs *before* the first phase of the
session and its purpose is to enrich and prepare the context for the subsequent AI interactions. Unlike regular session
prompts, the `prePrompt` does not contribute to the structured output of the session. It's particularly useful for
guiding LLMs that may struggle with combining tool calling and structured output by priming them with necessary context
or instructions.

For example, you can use a `prePrompt` to instruct the LLM on its persona or specific rules it should follow during the
session without those instructions being part of the final data extraction.

```yaml
sessions:
  my_session:
    prePrompt: "You are an expert data extraction agent. Always respond concisely."
    prompt: "Extract the user's name and email."
```
**Notice:** pre-prompts are also useful to overcome some LLM limitations. For example, some LLMs are incapable of
calling certain tools and provide a structured output in the same interaction.

## Tools
Frags allows LLMs that support tool calling to be used in your plan. Simply define the tools in the plan and reference
them in the prompts.  Tools must be enabled on a per-session basis.

**Example:**
```yaml
sessions:
  my_session:
    prompt: search for "frags" in the database
    tools:
      - name: db_search
        type: function
```
In this specific example, we're enabling a custom tool called `db_search` that was implemented and injected into the
AI client.
But there are other options

### internet_search
Search is a *special tool type* and does not require a coded implementation. Instead it relies on the LLM's built-in
search capabilities (if available). The Gemini AI client supports this out of the box.
To enable it, simply add the internet search tool like this:
```yaml
sessions:
  my_session:
    prePrompt: search for "frags" on the internet
    prompt: adapt the discovered information to the schema
    tools:
      - type: internet_search
```
**NOTICE:** in this example I explicitly placed the internet search tool in the `prePrompt` section, because **Gemini**
does not work well with search and structured output in the same interaction. If your model does not have this problem,
you can place it in the `prompt` section instead.

### MCP
Frags also support connection to MCP (Model Control Protocol) servers. The implementation is still quite naive, but it
should work for many use cases. To enable MCP servers, create an `mcp.json` file in the same directory as the `.env`
file. The format is similar if not identical to the one proposed by Anthropic.

**Example:**
```json
{
  "mcpServers": {
    "bigQuery": {
      "command": "toolbox",
      "args": ["--prebuilt","bigquery","--stdio"],
      "env": {
        "BIGQUERY_PROJECT": "foobar-436917",
        "GOOGLE_APPLICATION_CREDENTIALS": "bigQuery.json"
      }
    },
    "lmx": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

The supported modes are stdio and SSE. You will then need to enable the MCP servers you wish to use in each session.

**Example:**
```yaml
sessions:
  my_session:
    prePrompt: search for "frags" in bigQuery, table "services"
    prompt: adapt the discovered information to the schema
    tools:
      - type: mcp
        serverName: bigQuery
## continues with schema...
```

### Pre-Calls
Not a tool, but a way to employ tools, pre-calls allows you to directly instruct the Runner to call tools at the
start of a session. The invocation is programmatic, requires to plan writer to know exactly what they're doing, and
does not involve the LLM. The output of the pre-calls will be included in the context.

**Example:**
```yaml
sessions:
  my_session:
    preCalls:
      - name: list_files
        description: "File list of the test_data directory"
        args:
          dir: "./test_data/"
    prompt: What file types do we have?
## continues with schema...
```
**Notice:**
* out of the box, there's no `list_files` tool, this is just an example.
* Pre-Calls can access any available tool, regardless of whether they've been enabled for the specific session or not. 

## Implementing Custom Components

### ResourceLoader
Loading resources can take various forms based on the needs. The system implements a simple  `FileResourceLoader`, but
you can implement your own to load resources from any source.

See the [FileResourceLoader](https://github.com/theirish81/frags/blob/main/resource_loader.go) implementation for an
example.

### Ai
Frags is LLM agnostic. You can implement your own AI client by implementing the
[Ai interface](https://github.com/theirish81/frags/blob/main/ai.go#L11). There are currently 2 implementations,
distributed as separate packages:
- [Gemini](https://github.com/theirish81/frags/tree/main/gemini): our default
- [Ollama](https://github.com/theirish81/frags/tree/main/ollama): a simple wrapper around the Ollama API

## CLI 
Frags is also distributed with a [CLI tool](https://github.com/theirish81/frags/tree/main/cli).