# Go Chat with LangChain

This is a web application built with Go and LangChain (langchaingo) that provides a frontend interface for interacting with LLMs via OpenRouter.

## Prerequisites

- Go 1.19 or later
- An OpenRouter API key (get one from [openrouter.ai](https://openrouter.ai/))

## Setup

1. Clone or navigate to the project directory.
2. Install dependencies:
   ```
   go mod tidy
   ```

3. Set your OpenRouter API key as an environment variable:
   ```
   export OPENROUTER_API_KEY="your-api-key-here"
   ```

## Running the Application

Start the web server:
```
go run main.go
```

The server will start on `http://localhost:8080`. Open this URL in your browser to access the frontend.

## Features

- **Web Interface**: Simple HTML frontend for entering prompts and viewing responses.
- **API Endpoint**: `/generate` accepts POST requests with JSON payload `{"prompt": "your prompt"}` and returns `{"response": "generated text"}`.
- **LLM Integration**: Uses OpenRouter to access various LLMs (currently configured for a free model).

## API Usage

You can also interact with the API directly:

```bash
curl -X POST http://localhost:8080/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What is the capital of France?"}'
```

## Model Configuration

The app uses `openai/gpt-oss-120b:free` by default. To change the model, edit the `WithModel` line in `main.go`.

## Dependencies

- `github.com/tmc/langchaingo`: Go port of LangChain for LLM interactions.
- `github.com/gin-gonic/gin`: Web framework for the HTTP server.

## What it does

The program uses LangChain Go to:
- Initialize an OpenAI-compatible client configured for OpenRouter
- Send a prompt to generate a creative company name
- Print the response to the console

## Model Configuration

The code uses `anthropic/claude-3-haiku` as the default model. You can change this in `main.go` to any model supported by OpenRouter (e.g., `openai/gpt-4`, `meta-llama/llama-3-70b-instruct`).

## Dependencies

- `github.com/tmc/langchaingo`: The Go port of LangChain for building applications with LLMs