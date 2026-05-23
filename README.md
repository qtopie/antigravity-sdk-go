# antigravity-sdk-go

A **community Go port** of the [Google Antigravity Python SDK](https://github.com/google-antigravity/antigravity-sdk-python).

> ⚠️ **Disclaimer**: This project is an **independent, community-developed** Go implementation. It is **NOT** an official Google product and is **not affiliated with or endorsed by Google LLC** in any way. The official SDK is Python-only; this Go port was written independently by [qtopie](https://github.com/qtopie) based on the Python source.

## About

The [Google Antigravity Python SDK](https://github.com/google-antigravity/antigravity-sdk-python) is an open-source SDK (Apache-2.0) for building AI agents powered by the Gemini model. This repository provides a Go-language port of that SDK, enabling Go developers to build agents with the same agentic loop infrastructure.

### What's ported

| Component | Status |
|-----------|--------|
| `Agent` / `AgentConfig` | ✅ |
| `LocalConnectionStrategy` | ✅ |
| `Conversation` / `ConversationManager` | ✅ |
| `ToolRunner` | ✅ |
| `HookRunner` | ✅ |
| `MCPTriggers` | ✅ |
| Proto definitions (`localharness.proto`) | ✅ |

---

## Requirements

- Go 1.25+
- A valid Gemini API key ([Get one here](https://aistudio.google.com/apikey))
- The `localharness` binary from the [Antigravity Python SDK](https://github.com/google-antigravity/antigravity-sdk-python)

## Installation

```bash
go get github.com/qtopie/antigravity-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    antigravity "github.com/qtopie/antigravity-sdk-go"
)

func main() {
    ctx := context.Background()

    config := antigravity.AgentConfig{
        SystemInstructions: "You are a helpful Go assistant.",
        CreateStrategy: func(tools []any, hooks []any) (antigravity.ConnectionStrategy, error) {
            return antigravity.NewLocalConnectionStrategy(antigravity.AgentConfig{
                SystemInstructions: "You are a helpful Go assistant.",
            }), nil
        },
    }

    agent := antigravity.NewAgent(config)
    if err := agent.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer agent.Stop(ctx)

    response, err := agent.Chat(ctx, antigravity.TextContent("Hello!"))
    if err != nil {
        log.Fatal(err)
    }

    for chunk := range response.Chunks() {
        fmt.Printf("Chunk: %v\n", chunk)
    }
}
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GEMINI_API_KEY` | Your Gemini API key **(required)** |
| `ANTIGRAVITY_HARNESS_PATH` | Path to the `localharness` binary (defaults to `localharness` in `PATH`) |
| `HTTPS_PROXY` / `ALL_PROXY` | Proxy for outbound connections (see below) |

---

## Running the Example

### Basic

```bash
export GEMINI_API_KEY=your_api_key_here
export ANTIGRAVITY_HARNESS_PATH=/path/to/localharness

go run examples/hello_world/main.go
```

### With a Network Proxy

The SDK respects standard Go proxy environment variables. This is useful in environments where direct access to `generativelanguage.googleapis.com` is restricted.

#### HTTP Proxy

```bash
export HTTPS_PROXY=http://127.0.0.1:8118
export HTTP_PROXY=http://127.0.0.1:8118
```

#### SOCKS5 Proxy

```bash
# socks5h:// — proxy resolves DNS (recommended if DNS is blocked locally)
export HTTPS_PROXY=socks5h://192.168.1.1:1080
export ALL_PROXY=socks5h://192.168.1.1:1080

# socks5:// — client resolves DNS locally, then connects via SOCKS5
export HTTPS_PROXY=socks5://192.168.1.1:1080
export ALL_PROXY=socks5://192.168.1.1:1080
```

> **`socks5h` vs `socks5`**: Use `socks5h://` when local DNS may be polluted or unavailable — the proxy server handles name resolution. Use `socks5://` when your local DNS works fine.

#### Full Example with Proxy

```bash
export GEMINI_API_KEY=your_api_key_here
export ANTIGRAVITY_HARNESS_PATH=/path/to/localharness
export HTTPS_PROXY=socks5h://192.168.1.1:1080
export ALL_PROXY=socks5h://192.168.1.1:1080

go run examples/hello_world/main.go
```

Expected output:

```
2026/05/23 19:39:22 Initializing conversation with model: gemini-2.5-flash
2026/05/23 19:39:22 Sending InputEvent: user_input="Hello!"
2026/05/23 19:39:24 Received StepUpdate: Text="", State=STATE_ACTIVE
2026/05/23 19:39:24 [Harness Stderr] Connected to real-time API successfully
2026/05/23 19:39:24 [Harness Stderr] Received StepUpdate: text:"Hello!" state:STATE_DONE
```


## License & Attribution

This project is licensed under the **Apache License 2.0** — the same license as the original Python SDK.

- **This Go port**: Copyright 2026 qtopie, Apache-2.0
- **Original Python SDK**: Copyright 2026 Google LLC, Apache-2.0  
  Source: https://github.com/google-antigravity/antigravity-sdk-python

In accordance with the Apache 2.0 license:
- The original `LICENSE` file is included in this repository.
- All source files carry a notice indicating they are a port of the original work.
- Copyright notices from the original work are preserved in file headers.

## Contributing

Contributions are welcome! Please open an issue or pull request on [GitHub](https://github.com/qtopie/antigravity-sdk-go).

## Acknowledgements

- [Google Antigravity Python SDK](https://github.com/google-antigravity/antigravity-sdk-python) — the original work this port is based on.
- Google LLC — authors of the original Apache-2.0 licensed Python SDK.
