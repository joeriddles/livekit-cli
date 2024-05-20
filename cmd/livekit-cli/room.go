// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

const roomCategory = "Room Server API"

var (
	RoomCommands = []*cli.Command{
		{
			Name:     "create-room",
			Before:   createRoomClient,
			Action:   createRoom,
			Category: roomCategory,
			Flags: withDefaultFlags(
				&cli.StringFlag{
					Name:     "name",
					Usage:    "name of the room",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "room-egress-file",
					Usage: "RoomCompositeRequest json file (see examples/room-composite-file.json)",
				},
				&cli.StringFlag{
					Name:  "participant-egress-file",
					Usage: "ParticipantEgress json file (see examples/auto-participant-egress.json)",
				},
				&cli.StringFlag{
					Name:  "track-egress-file",
					Usage: "AutoTrackEgress json file (see examples/auto-track-egress.json)",
				},
				&cli.UintFlag{
					Name:  "min-playout-delay",
					Usage: "minimum playout delay for video (in ms)",
				},
				&cli.UintFlag{
					Name:  "max-playout-delay",
					Usage: "maximum playout delay for video (in ms)",
				},
				&cli.BoolFlag{
					Name:  "sync-streams",
					Usage: "improve A/V sync by placing them in the same stream. when enabled, transceivers will not be reused",
				},
				&cli.UintFlag{
					Name:  "empty-timeout",
					Usage: "number of seconds to keep the room open before any participant joins",
				},
				&cli.UintFlag{
					Name:  "departure-timeout",
					Usage: "number of seconds to keep the room open after the last participant leaves",
				},
			),
		},
		{
			Name:     "list-rooms",
			Before:   createRoomClient,
			Action:   listRooms,
			Category: roomCategory,
			Flags:    withDefaultFlags(),
		},
		{
			Name:     "list-room",
			Before:   createRoomClient,
			Action:   listRoom,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
			),
		},
		{
			Name:     "delete-room",
			Before:   createRoomClient,
			Action:   deleteRoom,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
			),
		},
		{
			Name:     "update-room-metadata",
			Before:   createRoomClient,
			Action:   updateRoomMetadata,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				&cli.StringFlag{
					Name: "metadata",
				},
			),
		},
		{
			Name:     "list-participants",
			Before:   createRoomClient,
			Action:   listParticipants,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
			),
		},
		{
			Name:     "get-participant",
			Before:   createRoomClient,
			Action:   getParticipant,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				identityFlag,
			),
		},
		{
			Name:     "remove-participant",
			Before:   createRoomClient,
			Action:   removeParticipant,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				identityFlag,
			),
		},
		{
			Name:     "update-participant",
			Before:   createRoomClient,
			Action:   updateParticipant,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				identityFlag,
				&cli.StringFlag{
					Name: "metadata",
				},
				&cli.StringFlag{
					Name:  "permissions",
					Usage: "JSON describing participant permissions (existing values for unset fields)",
				},
			),
		},
		{
			Name:     "mute-track",
			Before:   createRoomClient,
			Action:   muteTrack,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				identityFlag,
				&cli.StringFlag{
					Name:     "track",
					Usage:    "track sid to mute",
					Required: true,
				},
				&cli.BoolFlag{
					Name:  "muted",
					Usage: "set to true to mute, false to unmute",
				},
			),
		},
		{
			Name:     "update-subscriptions",
			Before:   createRoomClient,
			Action:   updateSubscriptions,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				identityFlag,
				&cli.StringSliceFlag{
					Name:     "track",
					Usage:    "track sid to subscribe/unsubscribe",
					Required: true,
				},
				&cli.BoolFlag{
					Name:  "subscribe",
					Usage: "set to true to subscribe, otherwise it'll unsubscribe",
				},
			),
		},
		{
			Name:     "send-data",
			Before:   createRoomClient,
			Action:   sendData,
			Category: roomCategory,
			Flags: withDefaultFlags(
				roomFlag,
				&cli.StringFlag{
					Name:     "data",
					Usage:    "payload to send to client",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "topic",
					Usage: "topic of the message",
				},
				&cli.StringSliceFlag{
					Name:  "participantID",
					Usage: "list of participantID to send the message to",
				},
			),
		},
	}

	roomClient *lksdk.RoomServiceClient
)

func createRoomClient(ctx context.Context, c *cli.Command) error {
	pc, err := loadProjectDetails(c)
	if err != nil {
		return err
	}

	roomClient = lksdk.NewRoomServiceClient(pc.URL, pc.APIKey, pc.APISecret, withDefaultClientOpts(pc)...)
	return nil
}

func createRoom(ctx context.Context, c *cli.Command) error {
	req := &livekit.CreateRoomRequest{
		Name: c.String("name"),
	}

	if roomEgressFile := c.String("room-egress-file"); roomEgressFile != "" {
		roomEgress := &livekit.RoomCompositeEgressRequest{}
		b, err := os.ReadFile(roomEgressFile)
		if err != nil {
			return err
		}
		if err = protojson.Unmarshal(b, roomEgress); err != nil {
			return err
		}
		req.Egress = &livekit.RoomEgress{Room: roomEgress}
	}

	if participantEgressFile := c.String("participant-egress-file"); participantEgressFile != "" {
		participantEgress := &livekit.AutoParticipantEgress{}
		b, err := os.ReadFile(participantEgressFile)
		if err != nil {
			return err
		}
		if err = protojson.Unmarshal(b, participantEgress); err != nil {
			return err
		}
		if req.Egress == nil {
			req.Egress = &livekit.RoomEgress{}
		}
		req.Egress.Participant = participantEgress
	}

	if trackEgressFile := c.String("track-egress-file"); trackEgressFile != "" {
		trackEgress := &livekit.AutoTrackEgress{}
		b, err := os.ReadFile(trackEgressFile)
		if err != nil {
			return err
		}
		if err = protojson.Unmarshal(b, trackEgress); err != nil {
			return err
		}
		if req.Egress == nil {
			req.Egress = &livekit.RoomEgress{}
		}
		req.Egress.Tracks = trackEgress
	}

	if c.Uint("min-playout-delay") != 0 {
		fmt.Printf("setting min playout delay: %d\n", c.Uint("min-playout-delay"))
		req.MinPlayoutDelay = uint32(c.Uint("min-playout-delay"))
	}

	if maxPlayoutDelay := c.Uint("max-playout-delay"); maxPlayoutDelay != 0 {
		fmt.Printf("setting max playout delay: %d\n", maxPlayoutDelay)
		req.MaxPlayoutDelay = uint32(maxPlayoutDelay)
	}

	if syncStreams := c.Bool("sync-streams"); syncStreams {
		fmt.Printf("setting sync streams: %t\n", syncStreams)
		req.SyncStreams = syncStreams
	}

	if emptyTimeout := c.Uint("empty-timeout"); emptyTimeout != 0 {
		fmt.Printf("setting empty timeout: %d\n", emptyTimeout)
		req.EmptyTimeout = uint32(emptyTimeout)
	}

	if departureTimeout := c.Uint("departure-timeout"); departureTimeout != 0 {
		fmt.Printf("setting departure timeout: %d\n", departureTimeout)
		req.DepartureTimeout = uint32(departureTimeout)
	}

	room, err := roomClient.CreateRoom(context.Background(), req)
	if err != nil {
		return err
	}

	PrintJSON(room)
	return nil
}

func listRooms(ctx context.Context, c *cli.Command) error {
	res, err := roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err != nil {
		return err
	}
	if len(res.Rooms) == 0 {
		fmt.Println("there are no active rooms")
	}
	for _, rm := range res.Rooms {
		fmt.Printf("%s\t%s\t%d participants\n", rm.Sid, rm.Name, rm.NumParticipants)
	}
	return nil
}

func listRoom(ctx context.Context, c *cli.Command) error {
	res, err := roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{
		Names: []string{c.String("room")},
	})
	if err != nil {
		return err
	}
	if len(res.Rooms) == 0 {
		fmt.Printf("there is no matching room with name: %s\n", c.String("room"))
		return nil
	}
	rm := res.Rooms[0]
	PrintJSON(rm)
	return nil
}

func deleteRoom(ctx context.Context, c *cli.Command) error {
	roomId := c.String("room")
	_, err := roomClient.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
		Room: roomId,
	})
	if err != nil {
		return err
	}

	fmt.Println("deleted room", roomId)
	return nil
}

func updateRoomMetadata(ctx context.Context, c *cli.Command) error {
	roomName := c.String("room")
	res, err := roomClient.UpdateRoomMetadata(context.Background(), &livekit.UpdateRoomMetadataRequest{
		Room:     roomName,
		Metadata: c.String("metadata"),
	})
	if err != nil {
		return err
	}

	fmt.Println("Updated room metadata")
	PrintJSON(res)
	return nil
}

func listParticipants(ctx context.Context, c *cli.Command) error {
	roomName := c.String("room")
	res, err := roomClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
		Room: roomName,
	})
	if err != nil {
		return err
	}

	for _, p := range res.Participants {
		fmt.Printf("%s (%s)\t tracks: %d\n", p.Identity, p.State.String(), len(p.Tracks))
	}
	return nil
}

func getParticipant(ctx context.Context, c *cli.Command) error {
	roomName, identity := participantInfoFromCli(c)
	res, err := roomClient.GetParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
	if err != nil {
		return err
	}

	PrintJSON(res)

	return nil
}

func updateParticipant(ctx context.Context, c *cli.Command) error {
	roomName, identity := participantInfoFromCli(c)
	metadata := c.String("metadata")
	permissions := c.String("permissions")
	if metadata == "" && permissions == "" {
		return fmt.Errorf("either metadata or permissions must be set")
	}

	req := &livekit.UpdateParticipantRequest{
		Room:     roomName,
		Identity: identity,
		Metadata: metadata,
	}
	if permissions != "" {
		// load existing participant
		participant, err := roomClient.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
			Room:     roomName,
			Identity: identity,
		})
		if err != nil {
			return err
		}

		req.Permission = participant.Permission
		if req.Permission != nil {
			if err = json.Unmarshal([]byte(permissions), req.Permission); err != nil {
				return err
			}
		}
	}

	fmt.Println("updating participant...")
	PrintJSON(req)
	if _, err := roomClient.UpdateParticipant(ctx, req); err != nil {
		return err
	}
	fmt.Println("participant updated.")

	return nil
}

func removeParticipant(ctx context.Context, c *cli.Command) error {
	roomName, identity := participantInfoFromCli(c)
	_, err := roomClient.RemoveParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
	if err != nil {
		return err
	}

	fmt.Println("successfully removed participant", identity)

	return nil
}

func muteTrack(ctx context.Context, c *cli.Command) error {
	roomName, identity := participantInfoFromCli(c)
	trackSid := c.String("track")
	_, err := roomClient.MutePublishedTrack(context.Background(), &livekit.MuteRoomTrackRequest{
		Room:     roomName,
		Identity: identity,
		TrackSid: trackSid,
		Muted:    c.Bool("muted"),
	})
	if err != nil {
		return err
	}

	verb := "muted"
	if !c.Bool("muted") {
		verb = "unmuted"
	}
	fmt.Println(verb, "track: ", trackSid)
	return nil
}

func updateSubscriptions(ctx context.Context, c *cli.Command) error {
	roomName, identity := participantInfoFromCli(c)
	trackSids := c.StringSlice("track")
	_, err := roomClient.UpdateSubscriptions(context.Background(), &livekit.UpdateSubscriptionsRequest{
		Room:      roomName,
		Identity:  identity,
		TrackSids: trackSids,
		Subscribe: c.Bool("subscribe"),
	})
	if err != nil {
		return err
	}

	verb := "subscribed to"
	if !c.Bool("subscribe") {
		verb = "unsubscribed from"
	}
	fmt.Println(verb, "tracks: ", trackSids)
	return nil
}

func sendData(ctx context.Context, c *cli.Command) error {
	roomName, _ := participantInfoFromCli(c)
	pIDs := c.StringSlice("participantID")
	data := c.String("data")
	topic := c.String("topic")
	req := &livekit.SendDataRequest{
		Room:            roomName,
		Data:            []byte(data),
		DestinationSids: pIDs,
	}
	if topic != "" {
		req.Topic = &topic
	}
	_, err := roomClient.SendData(ctx, req)
	if err != nil {
		return err
	}

	fmt.Println("successfully sent data to room", roomName)
	return nil
}

func participantInfoFromCli(c *cli.Command) (string, string) {
	return c.String("room"), c.String("identity")
}
