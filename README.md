`baton-linear` is a connector for Linear built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Linear API to sync data about organization, users, teams, and projects.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

Linear API key. 
The API key can be created in Settings -> Account -> API -> Personal API keys. 

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-linear
baton-linear
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_API_KEY=apiKey ghcr.io/conductorone/baton-linear:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-linear/cmd/baton-linear@main

BATON_API_KEY=apiKey
baton resources
```

# Data Model

`baton-linear` pulls down information about the following Linear resources:
- Organization
- Users
- Projects
- Teams

# Contributing, Support, and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-linear` Command Line Usage

```
baton-linear

Usage:
  baton-linear [flags]
  baton-linear [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --api-key string         required: The Linear Personal API key used to connect to the Linear API ($BATON_API_KEY)
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-linear
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning           This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync         This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing              This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                version for baton-linear

Use "baton-linear [command] --help" for more information about a command.

```
