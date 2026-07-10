package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// Member is one entry returned by ExtractMembers.
type Member struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// ExtractMembers lists the members of a public channel/supergroup by
// username (e.g. "@some_channel"). For very large channels Telegram caps
// how much of the member list is visible to non-admin accounts; run this
// worker's account as an admin of the channel for a complete list.
func (c *Client) ExtractMembers(ctx context.Context, channelUsername string) ([]Member, error) {
	channel, err := c.resolveChannel(ctx, channelUsername)
	if err != nil {
		return nil, err
	}

	var members []Member
	seen := make(map[int64]bool)
	const pageSize = 200
	offset := 0

	for {
		resp, err := c.api.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
			Channel: channel,
			Filter:  &tg.ChannelParticipantsRecent{},
			Offset:  offset,
			Limit:   pageSize,
			Hash:    0,
		})
		if err != nil {
			return nil, fmt.Errorf("get participants (offset %d): %w", offset, err)
		}

		page, ok := resp.(*tg.ChannelsChannelParticipants)
		if !ok || len(page.Participants) == 0 {
			break
		}

		users := make(map[int64]*tg.User, len(page.Users))
		for _, u := range page.Users {
			if user, ok := u.(*tg.User); ok {
				users[user.ID] = user
			}
		}

		for _, p := range page.Participants {
			uid, ok := participantUserID(p)
			if !ok || seen[uid] {
				continue
			}
			seen[uid] = true
			if u, found := users[uid]; found {
				members = append(members, Member{
					ID:        u.ID,
					Username:  u.Username,
					FirstName: u.FirstName,
					LastName:  u.LastName,
				})
			}
		}

		offset += len(page.Participants)
		if offset >= page.Count || len(page.Participants) < pageSize {
			break
		}
	}

	return members, nil
}

// participantUserID extracts the member's user ID across the various
// ChannelParticipant variants the API can return.
func participantUserID(p tg.ChannelParticipantClass) (int64, bool) {
	switch v := p.(type) {
	case *tg.ChannelParticipant:
		return v.UserID, true
	case *tg.ChannelParticipantSelf:
		return v.UserID, true
	case *tg.ChannelParticipantAdmin:
		return v.UserID, true
	case *tg.ChannelParticipantCreator:
		return v.UserID, true
	default:
		// Left/banned/other variants key by Peer rather than a plain user
		// ID in some schema versions — skip rather than guess wrong.
		return 0, false
	}
}
