package lists

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

func StarterPackMembers(ctx context.Context, xrpcc *xrpc.Client, packName, ownerDid string) (*appbsky.GraphStarterpack, error) {
	// TODO paginate/loop
	packs, err := appbsky.GraphGetActorStarterPacks(ctx, xrpcc, ownerDid, "", 10)
	if err != nil {
		return nil, err
	}

	var foundPack *appbsky.GraphStarterpack
	for _, packId := range packs.StarterPacks {
		var packData appbsky.GraphStarterpack
		buf := &bytes.Buffer{}
		err := packId.Record.Val.MarshalCBOR(buf)
		if err != nil {
			return nil, err
		}
		err = packData.UnmarshalCBOR(buf)
		if err != nil {
			return nil, err
		}
		if packData.Name != packName {
			continue
		}
		foundPack = &packData
		break
	}
	if foundPack == nil {
		return nil, fmt.Errorf("Pack '%v' doesn't exists\n", packName)
	}
	return foundPack, nil
}

func ListMembers(ctx context.Context, xrpcc *xrpc.Client, listName, ownerDid string) (*appbsky.GraphGetList_Output, error) {
	// TODO paginate/loop
	lists, err := appbsky.GraphGetLists(ctx, xrpcc, ownerDid, "", 10)
	if err != nil {
		return nil, err
	}

	var foundList *appbsky.GraphGetList_Output
	for _, listId := range lists.Lists {
		if listId.Name != listName {
			continue
		}
		foundList, err = appbsky.GraphGetList(ctx, xrpcc, ownerDid, 100, listId.Uri)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to get list: %v", err))

		}
		break
	}
	if foundList == nil {
		return nil, errors.New(fmt.Sprintf("Couldn't find list %s", listName))
	}
	sort.Slice(foundList.Items, func(i, j int) bool {
		return foundList.Items[i].Subject.Handle < foundList.Items[j].Subject.Handle
	})
	return foundList, nil
}
