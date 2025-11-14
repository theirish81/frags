# Frags (WORK IN PROGRESS!)

A Go library for fragmenting JSON Schema documents into sessions and phases to optimize AI API interactions.

## Overview

Frags enables you to break down complex JSON Schema structures into smaller, sequential fragments (phases and sessions). This is particularly useful when working with AI APIs that have output token limits, allowing you to process large schemas iteratively while maintaining a single source of truth.

The library is designed to be used in three ways, each building on the last:
1. **Standalone Schema**: Use the `Schema` type to manually fragment a schema by phase.
2. **Schema + SessionManager**: Use the `SessionManager` to define and manage multiple "sessions" against a single schema, loaded from a file (e.g. YAML).
3. **Schema + SessionManager + Runner**: Use the `Runner` to automate the entire process of running sessions and phases, including loading resources and calling an AI API.

## The Problem

When interacting with AI APIs that support structured JSON output, you may encounter scenarios where:

- The complete output schema is too large, potentially hitting output token limits in a single response
- You want to process information incrementally without maintaining multiple separate schemas
- You want to define multiple, distinct AI interactions that operate on the same overall data model, but without polluting a single global context.
- Sequential data collection makes more sense than requesting everything at once

## The Solution

Frags introduces two custom properties to JSON Schema that allows you to:

1. Define a single comprehensive schema for your entire data structure
2. Mark properties with `x-session` to isolate them into a specific context.
3. Mark properties with `x-phase` numbers to control when they should be processed within a session.
4. Extract session and phase-specific sub-schemas at runtime to request data items incrementally.

## Installation

```bash
go get github.com/theirish81/frags
```

## Level 1: Standalone Schema Usage

At its core, `frags` allows you to work with a `Schema` object to manually extract phased portions of a larger schema.
The purpose of such partitioning is to allow you to query the AI API incrementally, in the same conversational context.

### Basic Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/theirish81/frags"
)

func main() {
    // Define a schema with phased properties
    schema := frags.Schema{
        Type: "object",
        Properties: map[string]*frags.Schema{
            "name": {
                Type:   "string",
                XPhase: intPtr(0), // First phase
            },
            "email": {
                Type:   "string",
                Format: "email",
                XPhase: intPtr(0), // First phase
            },
            "address": {
                Type:   "string",
                XPhase: intPtr(1), // Second phase
            },
            "preferences": {
                Type:   "object",
                XPhase: intPtr(2), // Third phase
            },
        },
        Required: []string{"name", "email"},
    }

    // Get available phases
    phases := schema.GetPhaseIndexes()
    fmt.Println("Available phases:", phases) // [0, 1, 2]

    // Extract schema for phase 0
    phase0Schema, err := schema.GetPhase(0)
    if err != nil {
        panic(err)
    }

    // phase0Schema now contains only "name" and "email" properties
    // Use this reduced schema with your AI API
    jsonSchema, _ := json.MarshalIndent(phase0Schema, "", "  ")
    fmt.Println(string(jsonSchema))
}

func intPtr(i int) *int {
    return &i
}
```

## Level 2: Schema + SessionManager

The `SessionManager` introduces the concept of "sessions," which are self-contained, multi-phase conversational tasks.
While a single schema defines the entire data universe, and phases allow you to extract incremental chunks of data,
the context may become polluted with excessive information, reducing the efficacy. 
Sessions are used to create isolated contexts, which improve AI efficacy, while phases still allow you to retrieve
incremental chunks of data.

Here's how it works:
1.  Frags first filters the main schema using the `x-session` tag to create a temporary, session-specific sub-schema.
2.  It then uses the `x-phase` tags within that sub-schema to break the conversation into ordered, incremental steps.

This allows you to define multiple, distinct, and phased AI interactions that operate on the same overall data model
without interfering with one another.

### Example `sessions.yaml`

This example defines two sessions, `user_profile` and `product_review`, each with its own progressive phases.

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

sessions:
  user_profile:
    prompt: "Extract the user's primary details form the provided document"
    nextPhasePrompt: "Also these secondary details"
    resources:
      - user_text.txt
  product_review:
    prompt: "Extract the required information from the provided document"
    nextPhasePrompt: "Also extract these items"
    resources:
      - product_details.pdf
```

### Usage

```go
package main

import (
	"fmt"
	"github.com/theirish81/frags"
	"os"
)

func main() {
	// Load the session manager from a file
	data, _ := os.ReadFile("sessions.yaml")
	sm := frags.NewSessionManager()
	if err := sm.FromYAML(data); err != nil {
		panic(err)
    }
	// Get the schema for the 'user_profile' session
	userProfileSchema, err := sm.Schema.GetSession("user_profile")
	if err != nil {
		panic(err)
	}

	// The phase indexes are local to the session's schema.
	// userProfileSchema now contains only 'name', 'email', 'street_address', 'city', and 'country'.
	phases := userProfileSchema.GetPhaseIndexes()
	fmt.Println("Phases for user_profile session:", phases) // [0, 1]

	// You can now iterate through these phases to build the conversation.
	phase0, _ := userProfileSchema.GetPhase(0)
	fmt.Println("Phase 0 properties:", phase0) // [name, email]

	phase1, _ := userProfileSchema.GetPhase(1)
	fmt.Println("Phase 1 properties:", phase1) // [street_address, city, country]
    
    // Get the schema for the 'product_review' session
    productReviewSchema, err := sm.Schema.GetSession("product_review")
    if err != nil {
        panic(err)
    }
    
    phases = productReviewSchema.GetPhaseIndexes()
    fmt.Println("Phases for product_review session:", phases) // [0, 1, 2]
}
```

## Level 3: Full Automation with the Runner

The `Runner` is the highest-level abstraction. It automates the entire workflow:

1.  Loads a `SessionManager`.
2.  Takes a `ResourceLoader` to load the files/data required by each session.
3.  Takes an `Ai` interface implementation to make the actual calls to your AI.
4.  Runs each session, automatically iterating through its local phases. It uses the session's `prompt` for the first
    phase and the `nextPhasePrompt` for all subsequent phases.
5.  Unmarshals the structured JSON results from the AI into a final Go struct.

### Session Parallelism
Sessions can run in parallel, which can significantly improve performance. As a default, however, the runner will run
sessions sequentially. To enable parallelism, use the `WithSessionsWorkers(int)` option when creating the runner.

### Reusability
The same instance of a runner can be used multiple times, however, it can work on a task at a time and will return
an error if you call `Run` before the previous task has completed.

### Implementing the missing bits
* **ResourceLoader:** loading resources can take various forms based on the needs. The system implements a simple
  FileResourceLoader, but you can implement your own to load resources from any source.
* **Ai:** the runner calls your AI API using the `Call` method. You can implement your own AI API client here. The
  implementation should be stateful to allow the progressive conversation of phases. Ideally the AI implementation
  supports files uploads and JSON schema response formats