# Bluesky scratch space

[![Go Reference](https://pkg.go.dev/badge/github.com/micahhausler/bsky-scratch.svg)](https://pkg.go.dev/github.com/micahhausler/bsky-scratch)

This repository contains code to curate a Bluesky list and starter pack.


### List Manager
```
Usage of ./list-manager:
      --debug                    Debug mode
      --ignoreFile string        File of ignored users (default "ignored-ids.json")
      --listName string          List name (default "Principals of Amazon")
      --password string          Your password
      --searchTerms strings      Search term (default [Principal Engineer Amazon,Principal Engineer AWS])
      --starterPackName string   Starter pack name (default "Principal Engineers of Amazon")
      --username string          Your username
```

### CLI
```
Usage of ./bsky-cli:
      --debug             Debug mode
      --name string       Name of resource
      --owner string      resource owner
      --password string   Your password
      --username string   Your username
Bluesky CLI:
	Usage: bluesky-cli <verb> <resource> [flags]

	Verbs:
	- get - get resource

	Resources:
	- user          get user profile
	- list          get lists. When a --name  is specified, it will get the list details
	- starterpack   get starter packs. When a --name is specified, it will get the starter pack details
```

## License

[MIT Licensed](/LICENSE)
