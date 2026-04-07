# go-sb-testclient

[![Open in Visual Studio Code](https://img.shields.io/static/v1?logo=visualstudiocode&label=&message=Open%20in%20Visual%20Studio%20Code&labelColor=2c2c32&color=007acc&logoColor=007acc)](https://open.vscode.dev/danstis/go-sb-testclient)
[![Go Report Card](https://goreportcard.com/badge/github.com/danstis/go-sb-testclient?style=flat-square)](https://goreportcard.com/report/github.com/danstis/go-sb-testclient)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/danstis/go-sb-testclient)](https://pkg.go.dev/github.com/danstis/go-sb-testclient)
[![Release](https://img.shields.io/github/release/danstis/go-sb-testclient.svg?style=flat-square)](https://github.com/danstis/go-sb-testclient/releases/latest)

Azure Service Bus testing client for comparing two subscription receivers side by side.

The binary opens a receiver for `primaryServiceBus` and a receiver for `secondaryServiceBus`, then logs any received messages with `PRI` and `SEC` prefixes until the process is interrupted.

## Setup

Copy the example config into place before running the client:

```bash
cp config.json.example config.json
```

The application expects `config.json` in the repository root.

## Configuration

Each service bus block requires these fields:

- `connectionString`
- `topic`
- `subscription`

Top-level settings:

- `completeMessages`
- `checkInterval`

Example:

```json
{
  "primaryServiceBus": {
    "connectionString": "Endpoint=...",
    "topic": "topicName",
    "subscription": "subscriptionName"
  },
  "secondaryServiceBus": {
    "connectionString": "Endpoint=...",
    "topic": "topicName",
    "subscription": "subscriptionName"
  },
  "completeMessages": false,
  "checkInterval": "5s"
}
```

## Runtime behavior

- `checkInterval` must be a valid Go duration such as `5s`.
- `completeMessages=false` uses peek-lock mode.
- `completeMessages=true` switches to receive-and-delete mode.
- The process continues running until interrupted.

## Run

From the repository root:

```bash
go run ./cmd/go-sb-testclient
```

Once running, the client polls both subscriptions and writes received messages to the log with `PRI` and `SEC` prefixes.

## Commit message style

This repo uses [Conventional Commits](https://www.conventionalcommits.org/) to ensure the build numbering is generated correctly
