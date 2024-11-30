package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/micahhausler/bsky-scratch/lists"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
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

type IgnoredUsers []IgnoreUser

type IgnoreUser struct {
	Did    string `json:"did"`
	Handle string `json:"handle"`
}

func main() {
	username := flag.String("username", "", "Your username")
	password := flag.String("password", "", "Your password")
	listName := flag.String("listName", "Principals of Amazon", "List name")
	starterPackName := flag.String("starterPackName", "Principal Engineers of Amazon", "Starter pack name")
	searchTerms := flag.StringSlice("searchTerms", []string{"Principal Engineer Amazon", "Principal Engineer AWS"}, "Search term")
	ignoreFile := flag.String("ignoreFile", "ignored-ids.json", "File of ignored users")
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

	ctx := context.Background()
	dir := identity.DefaultDirectory()
	handle, err := dir.LookupHandle(ctx, syntax.Handle(*username))
	if err != nil {
		log.Fatalf("Failed to resolve handler: %v", err)
	}
	if *debug {
		dumpYaml(handle)
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

	pack, err := lists.StarterPackMembers(ctx, xrpcc, *starterPackName, handle.DID.String())
	if err != nil {
		log.Fatalf("Failed to get starter pack members: %v", err)
	}
	if *debug {
		dumpYaml(pack)
	}

	starterPackList, err := appbsky.GraphGetList(ctx, xrpcc, handle.DID.String(), 100, pack.List)
	if err != nil {
		log.Fatalf("Failed to get starter pack list: %v", err)
	}
	if *debug {
		dumpYaml(starterPackList)
	}
	starterPackDidMap := make(map[string]string)
	starterPackDids := make([]string, 0, *starterPackList.List.ListItemCount)
	for _, item := range starterPackList.Items {
		starterPackDids = append(starterPackDids, item.Subject.Did)
		starterPackDidMap[item.Subject.Did] = item.Subject.Handle
	}

	mainList, err := lists.ListMembers(ctx, xrpcc, *listName, handle.DID.String())
	if err != nil {
		log.Fatalf("Failed to get list members: %v", err)
	}
	mainListDids := make([]string, 0, *mainList.List.ListItemCount)
	listHandles := make([]string, 0, *mainList.List.ListItemCount)
	listDidMap := make(map[string]string)

	for _, item := range mainList.Items {
		listHandles = append(listHandles, item.Subject.Handle)
		mainListDids = append(mainListDids, item.Subject.Did)
		listDidMap[item.Subject.Did] = item.Subject.Handle
	}

	mainListDidSet := sets.New(mainListDids...)
	starterPackDidSet := sets.New(starterPackDids...)

	// Find the difference between the two sets
	onlyInMainList := mainListDidSet.Difference(starterPackDidSet)
	if len(onlyInMainList) > 0 {
		fmt.Println("Missing from Starter Pack:")
		for did := range onlyInMainList {
			fmt.Printf("%s\n", listDidMap[did])
		}
	} else {
		fmt.Printf("Starter pack includes all main list (%d members)\n", len(starterPackDidSet))
	}
	onlyInStarterPack := starterPackDidSet.Difference(mainListDidSet)
	if len(onlyInStarterPack) > 0 {
		fmt.Println("Missing from Main List:")
		for did := range onlyInStarterPack {
			fmt.Printf("%s\n", starterPackDidMap[did])
		}
	} else {
		fmt.Printf("Main list includes all starter pack (%d members)\n", len(mainListDidSet))
	}
	fmt.Println(strings.Repeat("-", 80))
	const padding = 3
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.TabIndent)
	fmt.Fprintf(w, "Handle\tName\t\n")
	for _, item := range mainList.Items {
		fmt.Fprintf(w, "%s\t%s\n", item.Subject.Handle, *item.Subject.DisplayName)

	}
	w.Flush()
	fmt.Println(strings.Repeat("-", 80))

	var f *os.File
	_, err = os.Stat(*ignoreFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to call stat on ignore file: %v", err)
	} else if err != nil && os.IsNotExist(err) {
		f, err = os.Create(*ignoreFile)
		if err != nil {
			log.Fatalf("Failed to create ignore file: %v", err)
		}
	} else {
		f, err = os.OpenFile(*ignoreFile, os.O_RDWR, 0644)
		if err != nil {
			log.Fatalf("Failed to open ignore file: %v", err)
		}
	}
	defer f.Close()

	ignoredUsers := IgnoredUsers{}
	err = json.NewDecoder(f).Decode(&ignoredUsers)
	if err != nil {
		log.Fatalf("Failed to decode ignore file: %v", err)
	}
	f.Seek(0, 0) // reset file read pointer

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	allKnownDids := sets.New(mainListDids...).Insert(starterPackDids...)
	for _, ignored := range ignoredUsers {
		allKnownDids.Insert(ignored.Did)
	}

	fmt.Println(strings.Repeat("#", 80))
	for _, term := range *searchTerms {
		// TODO: Handle pagination
		result, err := appbsky.ActorSearchActors(ctx, xrpcc, "", 50, term, "")
		if err != nil {
			log.Printf("Failed to search actors: %v", err)
			continue
		}
		fmt.Printf("Search term '%s' results: \n", term)
		fmt.Println(strings.Repeat("#", 80))
		for _, actor := range result.Actors {
			if allKnownDids.Has(actor.Did) {
				continue
			}

			dumpYaml(actor)

			ignoreUser, err := HandleUser(mainList, starterPackList, ignoredUsers, xrpcc, actor)
			if err != nil {
				if errors.Is(err, errors.New("User quit")) {
					break
				}
				log.Printf("Failed to handle user: %v", err)
				continue
			}
			if ignoreUser != nil {
				ignoredUsers = append(ignoredUsers, *ignoreUser)
				allKnownDids.Insert(ignoreUser.Did)
			}
			fmt.Println(strings.Repeat("-", 80))
		}

		fmt.Println(strings.Repeat("#", 80))
	}
	err = encoder.Encode(ignoredUsers)
	if err != nil {
		log.Fatalf("Failed to encode ignore file: %v", err)
	}

}

func HandleUser(list *appbsky.GraphGetList_Output, spList *appbsky.GraphGetList_Output, ignoredUsers IgnoredUsers, xrpcc *xrpc.Client, profileView *appbsky.ActorDefs_ProfileView) (*IgnoreUser, error) {
	reader := bufio.NewReader(os.Stdin)
	helpMessage := "Action: Add to list (a), Add to ignore file (i), Skip (s), Quit (q)"
	fmt.Println(helpMessage)

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)
		switch text {
		case "q":
			fmt.Println("Quitting")
			return nil, errors.New("User quit")
		case "a":
			fmt.Printf("Adding %s to list & starter pack\n", profileView.Handle)
			_, err := atproto.RepoCreateRecord(
				context.Background(),
				xrpcc,
				&atproto.RepoCreateRecord_Input{
					Record: &util.LexiconTypeDecoder{Val: &appbsky.GraphListitem{
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
						List:      list.List.Uri,
						Subject:   *&profileView.Did,
					}},
					Collection: "app.bsky.graph.listitem", // TODO what is the NSID?
					Repo:       list.List.Creator.Did,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("Failed to add user: %v", err)
			}
			fmt.Printf("Successfully added %s to list\n", profileView.Handle)

			_, err = atproto.RepoCreateRecord(
				context.Background(),
				xrpcc,
				&atproto.RepoCreateRecord_Input{
					Record: &util.LexiconTypeDecoder{Val: &appbsky.GraphListitem{
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
						List:      spList.List.Uri,
						Subject:   *&profileView.Did,
					}},
					Collection: "app.bsky.graph.listitem",
					Repo:       spList.List.Creator.Did,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("Failed to add user to starter pack: %v", err)
			}
			fmt.Printf("Successfully added %s to starter pack\n", profileView.Handle)
			return nil, nil
		case "i":
			fmt.Printf("Adding %s to ignore file\n", profileView.Handle)
			return &IgnoreUser{
				Did:    profileView.Did,
				Handle: profileView.Handle,
			}, nil
		case "s":
			fmt.Println("Skipping user")
			return nil, nil
		default:
			fmt.Println(helpMessage)
		}
	}
}
