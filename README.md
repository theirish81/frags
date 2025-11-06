# Frags

A Go library for fragmenting JSON Schema documents into phases to optimize AI API interactions.

## Overview

Frags enables you to break down complex JSON Schema structures into smaller, sequential fragments (phases). This is particularly useful when working with AI APIs that have output token limits, allowing you to process large schemas iteratively while maintaining a single source of truth.

## The Problem

When interacting with AI APIs that support structured JSON output, you may encounter scenarios where:

- The complete output schema is too large, potentially hitting token limits in a single response
- You want to process information incrementally without maintaining multiple separate schemas
- Sequential data collection makes more sense than requesting everything at once

## The Solution

Frags introduces a custom `x-phase` property to JSON Schema that allows you to:

1. Define a single comprehensive schema for your entire data structure
2. Mark properties with phase numbers to control when they should be processed
3. Extract phase-specific sub-schemas at runtime to request only what you need

## Installation

```bash
go get github.com/theirish81/frags
```

## Usage

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

### Workflow Pattern

```go
// 1. Define your complete schema once
fullSchema := defineYourSchema()

// 2. Iterate through phases
for _, phase := range fullSchema.GetPhaseIndexes() {
    // 3. Get the schema for this phase
    phaseSchema, _ := fullSchema.GetPhase(phase)
    
    // 4. Send to AI API with the phase-specific schema
    response := callAIAPI(phaseSchema)
    
    // 5. Merge response with accumulated data
    accumulateData(response)
}
```
