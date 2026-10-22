package ui

import (
	"context"
	"fmt"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

type groupExtendedInfo struct {
	Info               *types.GroupInfo
	AllMemberLink      bool
	HistoryShareAll    bool
	GroupHistoryToggle bool
}

func (c *MainController) fetchGroupExtendedInfo(ctx context.Context, jid types.JID) (*groupExtendedInfo, error) {
	di := c.client.DangerousInternals()
	res, err := di.SendGroupIQ(ctx, "get", jid, waBinary.Node{
		Tag:   "query",
		Attrs: waBinary.Attrs{"request": "interactive"},
	})
	if err != nil {
		return nil, err
	}
	groupNode, ok := res.GetOptionalChildByTag("group")
	if !ok {
		return nil, fmt.Errorf("missing group node")
	}
	info, err := di.ParseGroupNode(&groupNode)
	if err != nil || info == nil {
		return nil, err
	}
	lidPairs, redacted := di.CacheGroupInfo(info, true)
	_ = c.client.Store.LIDs.PutManyLIDMappings(ctx, lidPairs)
	_ = c.client.Store.Contacts.PutManyRedactedPhones(ctx, redacted)

	ext := &groupExtendedInfo{
		Info:          info,
		AllMemberLink: info.MemberAddMode == types.GroupMemberAddModeAllMember,
	}
	if n, ok := groupNode.GetOptionalChildByTag("member_link_mode"); ok {
		ext.AllMemberLink = nodeText(n) == "all_member_link"
	}
	if n, ok := groupNode.GetOptionalChildByTag("member_share_group_history_mode"); ok {
		ext.HistoryShareAll = nodeText(n) == "all_member_share"
	}
	if _, ok := groupNode.GetOptionalChildByTag("group_history"); ok {
		ext.GroupHistoryToggle = true
	}
	return ext, nil
}

func nodeText(n waBinary.Node) string {
	switch v := n.Content.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	}
	return ""
}

func (c *MainController) setGroupLinkMode(ctx context.Context, groupJID types.JID, allMembers bool) error {
	val := "ADMIN_LINK"
	if allMembers {
		val = "ALL_MEMBER_LINK"
	}
	return c.mexUpdateGroupProperty(ctx, groupJID, map[string]any{"member_link_mode": val})
}

func (c *MainController) setGroupHistoryShareMode(ctx context.Context, groupJID types.JID, enabled bool) error {
	val, tag := "ADMIN_SHARE", "no_group_history"
	if enabled {
		val, tag = "ALL_MEMBER_SHARE", "group_history"
	}
	errMex := c.mexUpdateGroupProperty(ctx, groupJID, map[string]any{"member_share_group_history_mode": val})
	_, errIQ := c.client.DangerousInternals().SendGroupIQ(ctx, "set", groupJID, waBinary.Node{Tag: tag})
	if errMex == nil || errIQ == nil {
		return nil
	}
	return errMex
}

func (c *MainController) mexUpdateGroupProperty(ctx context.Context, groupJID types.JID, update map[string]any) error {
	vars := map[string]any{"group_id": groupJID.String(), "update": update}
	di := c.client.DangerousInternals()
	if _, err := di.SendMexIQ(ctx, "9418211574894172", vars); err == nil {
		return nil
	}
	_, err := di.SendMexIQ(ctx, "9207174359373176", vars)
	return err
}
