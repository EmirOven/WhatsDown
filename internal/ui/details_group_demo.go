package ui

import (
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDemoGroupExtendedInfo(groupJID types.JID) *groupExtendedInfo {
	selfJID, _ := types.ParseJID(store.DemoSelfJID)
	aliceJID, _ := types.ParseJID("15550100001@s.whatsapp.net")
	marcoJID, _ := types.ParseJID("15550100002@s.whatsapp.net")
	elenaJID, _ := types.ParseJID("15550100003@s.whatsapp.net")
	liamJID, _ := types.ParseJID("15550100004@s.whatsapp.net")

	parts := []types.GroupParticipant{
		{JID: selfJID, PhoneNumber: selfJID, IsAdmin: true, IsSuperAdmin: true},
		{JID: aliceJID, PhoneNumber: aliceJID, IsAdmin: true},
		{JID: marcoJID, PhoneNumber: marcoJID},
		{JID: elenaJID, PhoneNumber: elenaJID},
		{JID: liamJID, PhoneNumber: liamJID},
	}

	info := &types.GroupInfo{
		JID:           groupJID,
		GroupTopic:    types.GroupTopic{Topic: "Demo group chat • Native GTK4 & Libadwaita WhatsApp client"},
		GroupLocked:   types.GroupLocked{IsLocked: false},
		GroupAnnounce: types.GroupAnnounce{IsAnnounce: false},
		MemberAddMode: types.GroupMemberAddModeAllMember,
		Participants:  parts,
	}

	return &groupExtendedInfo{
		Info:               info,
		AllMemberLink:      true,
		HistoryShareAll:    true,
		GroupHistoryToggle: true,
	}
}
