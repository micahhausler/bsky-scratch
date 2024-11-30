package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/spf13/pflag"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

func dumpYaml(obj interface{}) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
	}
	fmt.Println(string(data))
	fmt.Println(strings.Repeat("-", 80))
}

func main() {
	pflag.ErrHelp = errors.New(`Bluesky CLI:
	Usage: bluesky-cli <verb> <resource> [flags]

	Verbs:
	- get - get resource

	Resources:
	- user          get user profile
	- list          get lists. When a --name is specified, it will get the list details
	- starterpack   get starter packs. When a --name is specified, it will get the starter pack details`)

	owner := flag.String("owner", "", "resource owner")
	name := flag.String("name", "", "Name of resource")
	username := flag.String("username", "", "Your username")
	password := flag.String("password", "", "Your password")
	debug := flag.Bool("debug", false, "Debug mode")
	flag.Parse()

	if *username == "" {
		un, ok := os.LookupEnv("BLUESKY_USERNAME")
		if !ok {
			fmt.Println("BLUESKY_USERNAME nor --username was set")
			os.Exit(1)
		}
		username = &un
	}
	if *password == "" {
		pw, ok := os.LookupEnv("BLUESKY_PASSWORD")
		if !ok {
			fmt.Println("BLUESKY_PASSWORD nor --password was set")
			os.Exit(1)
		}
		password = &pw
	}

	if len(os.Args) < 3 {
		fmt.Println("Not enough arguments")
		os.Exit(1)
	}

	verb := os.Args[1]
	resource := os.Args[2]

	ctx := context.Background()
	dir := identity.DefaultDirectory()
	handle, err := dir.LookupHandle(ctx, syntax.Handle(*username))
	if err != nil {
		log.Fatalf("Failed to resolve handler: %v", err)
	}

	xrpcc := &xrpc.Client{Host: handle.PDSEndpoint()}
	session, err := atproto.ServerCreateSession(ctx, xrpcc, &atproto.ServerCreateSession_Input{
		Identifier: *username,
		Password:   *password,
	})
	if err != nil {
		log.Fatalf("UNABLE TO CONNECT: %v", err)
	}
	// Access Token is used to make authenticated requests
	// Refresh Token allows to generate a new Access Token
	xrpcc.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}
	if *debug {
		dumpYaml(xrpcc.Auth)
	}

	switch verb {
	case "get":
		switch resource {
		case "user":
			if *name == "" {
				fmt.Println("No name provided")
				os.Exit(1)
			}
			userHandle, err := dir.LookupHandle(ctx, syntax.Handle(*name))
			if err != nil {
				log.Fatalf("Failed to resolve handler: %v", err)
			}

			profile, err := bsky.ActorGetProfile(ctx, xrpcc, userHandle.DID.String())
			if err != nil {
				log.Fatalf("Failed to get profiles: %v", err)
			}
			dumpYaml(profile)
			return
		case "list", "lists":
			ownerHandle, err := dir.LookupHandle(ctx, syntax.Handle(*owner))
			if err != nil {
				log.Fatalf("Failed to resolve handler: %v", err)
			}
			// todo: paginate
			lists, err := bsky.GraphGetLists(ctx, xrpcc, ownerHandle.DID.String(), "", 10)
			if err != nil {
				log.Fatalf("Failed to get lists: %v", err)
			}
			if *name != "" {
				for _, list := range lists.Lists {
					if list.Name != *name {
						continue
					}
					list, err := bsky.GraphGetList(ctx, xrpcc, ownerHandle.DID.String(), 100, list.Uri)
					if err != nil {
						log.Fatalf("Failed to get starter pack list: %v", err)
					}
					dumpYaml(list)
				}
				return
			}
			dumpYaml(lists)
			return
		case "starterpack", "starterpacks":
			ownerHandle, err := dir.LookupHandle(ctx, syntax.Handle(*owner))
			if err != nil {
				log.Fatalf("Failed to resolve handler: %v", err)
			}
			// todo: paginate
			packs, err := bsky.GraphGetActorStarterPacks(ctx, xrpcc, ownerHandle.DID.String(), "", 10)
			if err != nil {
				log.Fatalf("Failed to get starter packs: %v", err)
			}
			if *name != "" {
				var foundPack *bsky.GraphStarterpack
				for _, packId := range packs.StarterPacks {
					var packData bsky.GraphStarterpack
					buf := &bytes.Buffer{}
					err := packId.Record.Val.MarshalCBOR(buf)
					if err != nil {
						log.Fatalf("Failed to marshal data: %v", err)
					}
					err = packData.UnmarshalCBOR(buf)
					if err != nil {
						log.Fatalf("Failed to unmarshal data: %v", err)
					}
					if packData.Name != *name {
						continue
					}
					foundPack = &packData
					break
				}
				dumpYaml(foundPack)
				return
			}
			dumpYaml(packs)
		default:
			fmt.Printf("Unknown resource %s\n", resource)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown verb %s\n", verb)
		os.Exit(1)
	}

}
