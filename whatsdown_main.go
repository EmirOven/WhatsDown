package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type chatRowItem struct {
	isCommunityHeader bool
	communityJID      string
	chat              Chat
}

// whatsDownMainController manages the state and UI interactions of the
// WhatsDownMain window.
type whatsDownMainController struct {
	window            *adw.ApplicationWindow
	splitView         *adw.NavigationSplitView
	sidebarStack      *gtk.Stack
	chatsNavBtn       *gtk.Button
	callsNavBtn       *gtk.Button
	archivedNavBtn    *gtk.Button
	settingsNavBtn    *gtk.Button
	chatsSearchEntry  *gtk.SearchEntry
	chatsList         *gtk.ListBox
	callsPage         *adw.ToolbarView
	callsList         *gtk.ListBox
	callsSearchEntry  *gtk.SearchEntry
	archivedSearchEntry *gtk.SearchEntry
	archivedChatsList *gtk.ListBox
	contentStack      *gtk.Stack
	activeChatBtn     *gtk.Button
	activeChatAvatar  *adw.Avatar
	activeChatTitle   *adw.WindowTitle
	chatSearchBtn          *gtk.ToggleButton
	chatSearchBar          *gtk.SearchBar
	chatSearchEntry        *gtk.SearchEntry
	chatSearchCountLabel   *gtk.Label
	chatSearchPrevBtn      *gtk.Button
	chatSearchNextBtn      *gtk.Button
	scrollToBottomBtn      *gtk.Button
	messagesScrolledWindow *gtk.ScrolledWindow
	messagesList           *gtk.ListBox
	messageEntry           *gtk.Entry
	sendButton             *gtk.Button
	attachButton           *gtk.MenuButton
	attachPopover          *gtk.Popover
	attachPhotoBtn         *gtk.Button
	attachDocBtn           *gtk.Button
	attachPollBtn          *gtk.Button
	emojiButton            *gtk.MenuButton
	emojiChooser           *gtk.EmojiChooser
	stickerButton          *gtk.MenuButton
	stickerPopover         *gtk.Popover
	stickerFlowbox         *gtk.FlowBox
	stickerEmptyBox        *gtk.Box
	stickerScrolledWindow  *gtk.ScrolledWindow
	addStickerBtn          *gtk.Button
	savedStickers          []SavedSticker
	savedStickerHashes     map[string]bool
	suppressChatSelection  bool
	autoScrollBottom       bool
	messageStatusLabels    map[string]*gtk.Label

	// Reply preview widgets & state
	replyPreviewBar    *gtk.Box
	replyPreviewSender *gtk.Label
	replyPreviewText   *gtk.Label
	cancelReplyBtn     *gtk.Button
	replyingTo         *StoredMessage
	messageRowMap      map[string]*gtk.ListBoxRow

	// Markdown preview widgets
	markdownPreviewBar   *gtk.Box
	markdownPreviewLabel *gtk.Label

	// Edit message widgets & state
	editBannerBar  *gtk.Box
	editBannerText *gtk.Label
	cancelEditBtn  *gtk.Button
	editingMessage *StoredMessage

	// Voice message recorder widgets & state
	chatInputStack   *gtk.Stack
	voiceRecordBtn   *gtk.Button
	voiceDeleteBtn   *gtk.Button
	voiceStatusIcon  *gtk.Image
	voiceTimerLabel  *gtk.Label
	voiceMeterArea   *gtk.DrawingArea
	voicePauseBtn    *gtk.Button
	voiceStopBtn     *gtk.Button
	voicePlayBtn     *gtk.Button
	voiceSendBtn     *gtk.Button
	voiceRecorder    *voiceRecorder
	voiceTimerHandle glib.SourceHandle
	voicePreviewPath string
	voiceIsPlaying   bool

	// Settings widgets
	settingsProfileRow    *adw.ActionRow
	settingsProfileAvatar *adw.Avatar
	settingsResyncBtn     *gtk.Button
	settingsDbPathRow     *adw.ActionRow
	settingsStatsRow      *adw.ActionRow
	settingsClearCacheBtn *gtk.Button
	settingsThemeRow      *adw.ComboRow
	settingsLogoutBtn     *gtk.Button

	currentChat             *Chat
	chats                   []Chat
	archivedChats           []Chat
	calls                   []CallRecord
	expandedCommunities    map[string]bool
	displayedChatRows      []chatRowItem
	displayedArchivedRows  []chatRowItem
	currentMessages         []StoredMessage
	chatsSearchFilter       string
	archivedSearchFilter    string
	messagesSearchFilter    string
	searchMatchIndices      []int
	currentSearchMatchIndex int
}

var activeMainWindow *whatsDownMainController

// setupMainWindow initializes the WhatsDownMain window and its controllers.
func setupMainWindow(builder *gtk.Builder) (*adw.ApplicationWindow, error) {
	window, ok1 := castObject[*adw.ApplicationWindow](builder, "main_window")
	splitView, ok2 := castObject[*adw.NavigationSplitView](builder, "split_view")
	chatsList, ok3 := castObject[*gtk.ListBox](builder, "chats_list")
	contentStack, ok4 := castObject[*gtk.Stack](builder, "content_stack")
	activeChatBtn, okBtn := castObject[*gtk.Button](builder, "active_chat_btn")
	activeChatAvatar, ok5 := castObject[*adw.Avatar](builder, "active_chat_avatar")
	activeChatTitle, ok6 := castObject[*adw.WindowTitle](builder, "active_chat_title")
	messagesList, ok7 := castObject[*gtk.ListBox](builder, "messages_list")
	messageEntry, ok8 := castObject[*gtk.Entry](builder, "message_entry")
	sendButton, ok9 := castObject[*gtk.Button](builder, "send_button")
	chatsNavBtn, ok10 := castObject[*gtk.Button](builder, "chats_nav_btn")
	archivedNavBtn, ok11 := castObject[*gtk.Button](builder, "archived_nav_btn")
	sidebarStack, ok12 := castObject[*gtk.Stack](builder, "sidebar_stack")
	archivedChatsList, ok13 := castObject[*gtk.ListBox](builder, "archived_chats_list")
	settingsNavBtn, ok14 := castObject[*gtk.Button](builder, "settings_nav_btn")
	settingsProfileRow, ok15 := castObject[*adw.ActionRow](builder, "settings_profile_row")
	settingsProfileAvatar, ok16 := castObject[*adw.Avatar](builder, "settings_profile_avatar")
	settingsResyncBtn, ok17 := castObject[*gtk.Button](builder, "settings_resync_btn")
	settingsDbPathRow, ok18 := castObject[*adw.ActionRow](builder, "settings_db_path_row")
	settingsStatsRow, ok19 := castObject[*adw.ActionRow](builder, "settings_stats_row")
	settingsClearCacheBtn, ok20 := castObject[*gtk.Button](builder, "settings_clear_cache_btn")
	settingsThemeRow, ok21 := castObject[*adw.ComboRow](builder, "settings_theme_row")
	settingsLogoutBtn, ok22 := castObject[*gtk.Button](builder, "settings_logout_btn")
	chatsSearchEntry, ok23 := castObject[*gtk.SearchEntry](builder, "chats_search_entry")
	archivedSearchEntry, ok24 := castObject[*gtk.SearchEntry](builder, "archived_search_entry")
	chatSearchBtn, ok25 := castObject[*gtk.ToggleButton](builder, "chat_search_btn")
	chatSearchBar, ok26 := castObject[*gtk.SearchBar](builder, "chat_search_bar")
	chatSearchEntry, ok27 := castObject[*gtk.SearchEntry](builder, "chat_search_entry")
	chatSearchCountLabel, _ := castObject[*gtk.Label](builder, "chat_search_count_label")
	chatSearchPrevBtn, _ := castObject[*gtk.Button](builder, "chat_search_prev_btn")
	chatSearchNextBtn, _ := castObject[*gtk.Button](builder, "chat_search_next_btn")
	scrollToBottomBtn, _ := castObject[*gtk.Button](builder, "scroll_to_bottom_btn")
	messagesScrolledWindow, _ := castObject[*gtk.ScrolledWindow](builder, "messages_scrolled_window")
	attachButton, _ := castObject[*gtk.MenuButton](builder, "attach_button")
	attachPopover, _ := castObject[*gtk.Popover](builder, "attach_popover")
	attachPhotoBtn, _ := castObject[*gtk.Button](builder, "attach_photo_btn")
	attachDocBtn, _ := castObject[*gtk.Button](builder, "attach_doc_btn")
	attachPollBtn, _ := castObject[*gtk.Button](builder, "attach_poll_btn")
	emojiButton, _ := castObject[*gtk.MenuButton](builder, "emoji_button")
	emojiChooser, _ := castObject[*gtk.EmojiChooser](builder, "emoji_chooser")
	stickerButton, _ := castObject[*gtk.MenuButton](builder, "sticker_button")
	stickerPopover, _ := castObject[*gtk.Popover](builder, "sticker_popover")
	stickerFlowbox, _ := castObject[*gtk.FlowBox](builder, "sticker_flowbox")
	stickerEmptyBox, _ := castObject[*gtk.Box](builder, "sticker_empty_box")
	stickerScrolledWindow, _ := castObject[*gtk.ScrolledWindow](builder, "sticker_scrolled_window")
	addStickerBtn, _ := castObject[*gtk.Button](builder, "add_sticker_btn")
	replyPreviewBar, _ := castObject[*gtk.Box](builder, "reply_preview_bar")
	replyPreviewSender, _ := castObject[*gtk.Label](builder, "reply_preview_sender")
	replyPreviewText, _ := castObject[*gtk.Label](builder, "reply_preview_text")
	cancelReplyBtn, _ := castObject[*gtk.Button](builder, "cancel_reply_btn")
	markdownPreviewBar, _ := castObject[*gtk.Box](builder, "markdown_preview_bar")
	markdownPreviewLabel, _ := castObject[*gtk.Label](builder, "markdown_preview_label")
	contentNavPage, _ := castObject[*adw.NavigationPage](builder, "content_nav_page")
	chatsPage, _ := castObject[*adw.ToolbarView](builder, "chats_page")
	callsPage, _ := castObject[*adw.ToolbarView](builder, "calls_page")
	callsNavBtn, _ := castObject[*gtk.Button](builder, "calls_nav_btn")
	callsList, _ := castObject[*gtk.ListBox](builder, "calls_list")
	callsSearchEntry, _ := castObject[*gtk.SearchEntry](builder, "calls_search_entry")
	archivedPage, _ := castObject[*adw.ToolbarView](builder, "archived_page")
	settingsPage, _ := castObject[*adw.ToolbarView](builder, "settings_page")
	rightChatView, _ := castObject[*adw.ToolbarView](builder, "right_chat_view")
	editBannerBar, _ := castObject[*gtk.Box](builder, "edit_banner_bar")
	editBannerText, _ := castObject[*gtk.Label](builder, "edit_banner_text")
	cancelEditBtn, _ := castObject[*gtk.Button](builder, "cancel_edit_btn")
	chatInputStack, _ := castObject[*gtk.Stack](builder, "chat_input_stack")
	voiceRecordBtn, _ := castObject[*gtk.Button](builder, "voice_record_btn")
	voiceDeleteBtn, _ := castObject[*gtk.Button](builder, "voice_delete_btn")
	voiceStatusIcon, _ := castObject[*gtk.Image](builder, "voice_status_icon")
	voiceTimerLabel, _ := castObject[*gtk.Label](builder, "voice_timer_label")
	voiceMeterArea, _ := castObject[*gtk.DrawingArea](builder, "voice_meter_area")
	voicePauseBtn, _ := castObject[*gtk.Button](builder, "voice_pause_btn")
	voiceStopBtn, _ := castObject[*gtk.Button](builder, "voice_stop_btn")
	voicePlayBtn, _ := castObject[*gtk.Button](builder, "voice_play_btn")
	voiceSendBtn, _ := castObject[*gtk.Button](builder, "voice_send_btn")

	if !ok1 || !ok2 || !ok3 || !ok4 || !okBtn || !ok5 || !ok6 || !ok7 || !ok8 || !ok9 || !ok10 || !ok11 || !ok12 || !ok13 ||
		!ok14 || !ok15 || !ok16 || !ok17 || !ok18 || !ok19 || !ok20 || !ok21 || !ok22 ||
		!ok23 || !ok24 || !ok25 || !ok26 || !ok27 {
		return nil, errors.New("WhatsDownMain.ui missing required widgets")
	}

	if sidebarStack != nil {
		if chatsPage != nil && sidebarStack.ChildByName("chats") == nil {
			sidebarStack.AddTitled(chatsPage, "chats", "Chats")
		}
		if callsPage != nil && sidebarStack.ChildByName("calls") == nil {
			sidebarStack.AddTitled(callsPage, "calls", "Calls")
		}
		if archivedPage != nil && sidebarStack.ChildByName("archived") == nil {
			sidebarStack.AddTitled(archivedPage, "archived", "Archived")
		}
		if settingsPage != nil && sidebarStack.ChildByName("settings") == nil {
			sidebarStack.AddTitled(settingsPage, "settings", "Settings")
		}
	}
	if contentNavPage != nil && rightChatView != nil {
		contentNavPage.SetChild(rightChatView)
	}

	ctrl := &whatsDownMainController{
		window:                 window,
		splitView:              splitView,
		sidebarStack:           sidebarStack,
		chatsNavBtn:            chatsNavBtn,
		callsNavBtn:            callsNavBtn,
		callsPage:              callsPage,
		callsList:              callsList,
		callsSearchEntry:       callsSearchEntry,
		expandedCommunities:    make(map[string]bool),
		archivedNavBtn:         archivedNavBtn,
		settingsNavBtn:         settingsNavBtn,
		chatsSearchEntry:       chatsSearchEntry,
		chatsList:              chatsList,
		archivedSearchEntry:    archivedSearchEntry,
		archivedChatsList:      archivedChatsList,
		contentStack:           contentStack,
		activeChatBtn:          activeChatBtn,
		activeChatAvatar:       activeChatAvatar,
		activeChatTitle:        activeChatTitle,
		chatSearchBtn:          chatSearchBtn,
		chatSearchBar:          chatSearchBar,
		chatSearchEntry:        chatSearchEntry,
		chatSearchCountLabel:   chatSearchCountLabel,
		chatSearchPrevBtn:      chatSearchPrevBtn,
		chatSearchNextBtn:      chatSearchNextBtn,
		scrollToBottomBtn:      scrollToBottomBtn,
		messagesScrolledWindow: messagesScrolledWindow,
		messagesList:           messagesList,
		messageEntry:           messageEntry,
		sendButton:             sendButton,
		attachButton:           attachButton,
		attachPopover:          attachPopover,
		attachPhotoBtn:         attachPhotoBtn,
		attachDocBtn:           attachDocBtn,
		attachPollBtn:          attachPollBtn,
		emojiButton:            emojiButton,
		emojiChooser:           emojiChooser,
		stickerButton:          stickerButton,
		stickerPopover:         stickerPopover,
		stickerFlowbox:         stickerFlowbox,
		stickerEmptyBox:        stickerEmptyBox,
		stickerScrolledWindow:  stickerScrolledWindow,
		addStickerBtn:          addStickerBtn,
		savedStickerHashes:     make(map[string]bool),
		autoScrollBottom:       true,
		messageStatusLabels:    make(map[string]*gtk.Label),
		replyPreviewBar:        replyPreviewBar,
		replyPreviewSender:     replyPreviewSender,
		replyPreviewText:       replyPreviewText,
		cancelReplyBtn:         cancelReplyBtn,
		markdownPreviewBar:     markdownPreviewBar,
		markdownPreviewLabel:   markdownPreviewLabel,
		editBannerBar:          editBannerBar,
		editBannerText:         editBannerText,
		cancelEditBtn:          cancelEditBtn,
		chatInputStack:         chatInputStack,
		voiceRecordBtn:         voiceRecordBtn,
		voiceDeleteBtn:         voiceDeleteBtn,
		voiceStatusIcon:        voiceStatusIcon,
		voiceTimerLabel:        voiceTimerLabel,
		voiceMeterArea:         voiceMeterArea,
		voicePauseBtn:          voicePauseBtn,
		voiceStopBtn:           voiceStopBtn,
		voicePlayBtn:           voicePlayBtn,
		voiceSendBtn:           voiceSendBtn,
		messageRowMap:          make(map[string]*gtk.ListBoxRow),
		settingsProfileRow:     settingsProfileRow,
		settingsProfileAvatar:  settingsProfileAvatar,
		settingsResyncBtn:      settingsResyncBtn,
		settingsDbPathRow:      settingsDbPathRow,
		settingsStatsRow:       settingsStatsRow,
		settingsClearCacheBtn:  settingsClearCacheBtn,
		settingsThemeRow:       settingsThemeRow,
		settingsLogoutBtn:      settingsLogoutBtn,
	}
	activeMainWindow = ctrl


	// Search entries
	chatsSearchEntry.ConnectSearchChanged(func() {
		ctrl.chatsSearchFilter = strings.TrimSpace(chatsSearchEntry.Text())
		ctrl.loadChats()
	})

	archivedSearchEntry.ConnectSearchChanged(func() {
		ctrl.archivedSearchFilter = strings.TrimSpace(archivedSearchEntry.Text())
		ctrl.loadArchivedChats()
	})

	if callsSearchEntry != nil {
		callsSearchEntry.ConnectSearchChanged(func() {
			ctrl.loadCalls()
		})
	}

	// Message search inside conversation
	chatSearchBtn.SetSensitive(false)
	chatSearchBar.ConnectEntry(chatSearchEntry)
	ctrl.updateSearchUI()

	// Attachment buttons
	if attachPhotoBtn != nil {
		attachPhotoBtn.ConnectClicked(func() {
			if attachPopover != nil {
				attachPopover.Popdown()
			}
			ctrl.openFileChooserForMedia(true)
		})
	}
	if attachDocBtn != nil {
		attachDocBtn.ConnectClicked(func() {
			if attachPopover != nil {
				attachPopover.Popdown()
			}
			ctrl.openFileChooserForMedia(false)
		})
	}
	if attachPollBtn != nil {
		attachPollBtn.ConnectClicked(func() {
			if attachPopover != nil {
				attachPopover.Popdown()
			}
			ctrl.showCreatePollDialog()
		})
	}
	if emojiChooser != nil {
		emojiChooser.ConnectEmojiPicked(func(text string) {
			ctrl.insertEmoji(text)
		})
	}
	if addStickerBtn != nil {
		addStickerBtn.ConnectClicked(func() {
			ctrl.openFileChooserForSticker()
		})
	}
	if stickerPopover != nil {
		stickerPopover.ConnectShow(func() {
			ctrl.loadSavedStickersUI()
		})
	}
	ctrl.loadSavedStickersUI()

	chatSearchBtn.ConnectToggled(func() {
		active := chatSearchBtn.Active()
		chatSearchBar.SetSearchMode(active)
		if active {
			chatSearchEntry.GrabFocus()
		} else {
			chatSearchEntry.SetText("")
			ctrl.messagesSearchFilter = ""
			ctrl.searchMatchIndices = nil
			ctrl.currentSearchMatchIndex = 0
			ctrl.updateSearchUI()
		}
	})

	chatSearchEntry.ConnectSearchChanged(func() {
		ctrl.performMessageSearch(chatSearchEntry.Text())
	})

	chatSearchEntry.ConnectActivate(func() {
		ctrl.navigateSearchMatch(1)
		chatSearchEntry.GrabFocus()
	})

	if chatSearchNextBtn != nil {
		chatSearchNextBtn.ConnectClicked(func() {
			ctrl.navigateSearchMatch(1)
			chatSearchEntry.GrabFocus()
		})
	}
	if chatSearchPrevBtn != nil {
		chatSearchPrevBtn.ConnectClicked(func() {
			ctrl.navigateSearchMatch(-1)
			chatSearchEntry.GrabFocus()
		})
	}

	// Scroll to bottom FAB & adjustment watcher
	if scrollToBottomBtn != nil {
		scrollToBottomBtn.ConnectClicked(func() {
			ctrl.autoScrollBottom = true
			ctrl.scrollToBottom()
		})
	}

	if messagesScrolledWindow != nil {
		if adj := messagesScrolledWindow.VAdjustment(); adj != nil {
			adj.ConnectValueChanged(func() {
				maxVal := adj.Upper() - adj.PageSize()
				isAtBottom := maxVal <= 10 || adj.Value() >= maxVal-60
				ctrl.autoScrollBottom = isAtBottom
				if scrollToBottomBtn != nil {
					scrollToBottomBtn.SetVisible(!isAtBottom)
				}
			})
			adj.ConnectChanged(func() {
				if ctrl.autoScrollBottom {
					ctrl.scrollToBottom()
				}
			})
		}
	}

	// Header active chat button opens details & media dialog
	ctrl.activeChatBtn.SetSensitive(false)
	ctrl.activeChatBtn.ConnectClicked(func() {
		if ctrl.currentChat != nil {
			ctrl.showDetailsAndMedia(*ctrl.currentChat)
		}
	})

	// Show empty placeholder by default
	contentStack.SetVisibleChildName("empty")

	// Default sidebar view is Chats
	sidebarStack.SetVisibleChildName("chats")
	chatsNavBtn.AddCSSClass("active")
	if callsNavBtn != nil {
		callsNavBtn.RemoveCSSClass("active")
	}
	archivedNavBtn.RemoveCSSClass("active")
	settingsNavBtn.RemoveCSSClass("active")

	// Navigation rail buttons
	chatsNavBtn.ConnectClicked(func() {
		sidebarStack.SetVisibleChildName("chats")
		chatsNavBtn.AddCSSClass("active")
		if callsNavBtn != nil {
			callsNavBtn.RemoveCSSClass("active")
		}
		archivedNavBtn.RemoveCSSClass("active")
		settingsNavBtn.RemoveCSSClass("active")
		if ctrl.currentChat != nil {
			contentStack.SetVisibleChildName("chat")
		} else {
			contentStack.SetVisibleChildName("empty")
		}
		ctrl.loadChats()
	})
	if callsNavBtn != nil {
		callsNavBtn.ConnectClicked(func() {
			sidebarStack.SetVisibleChildName("calls")
			callsNavBtn.AddCSSClass("active")
			chatsNavBtn.RemoveCSSClass("active")
			archivedNavBtn.RemoveCSSClass("active")
			settingsNavBtn.RemoveCSSClass("active")
			if ctrl.currentChat != nil {
				contentStack.SetVisibleChildName("chat")
			} else {
				contentStack.SetVisibleChildName("empty")
			}
			ctrl.loadCalls()
		})
	}
	archivedNavBtn.ConnectClicked(func() {
		sidebarStack.SetVisibleChildName("archived")
		archivedNavBtn.AddCSSClass("active")
		chatsNavBtn.RemoveCSSClass("active")
		if callsNavBtn != nil {
			callsNavBtn.RemoveCSSClass("active")
		}
		settingsNavBtn.RemoveCSSClass("active")
		if ctrl.currentChat != nil {
			contentStack.SetVisibleChildName("chat")
		} else {
			contentStack.SetVisibleChildName("empty")
		}
		ctrl.loadArchivedChats()
	})
	settingsNavBtn.ConnectClicked(func() {
		sidebarStack.SetVisibleChildName("settings")
		settingsNavBtn.AddCSSClass("active")
		chatsNavBtn.RemoveCSSClass("active")
		if callsNavBtn != nil {
			callsNavBtn.RemoveCSSClass("active")
		}
		archivedNavBtn.RemoveCSSClass("active")
		contentStack.SetVisibleChildName("settings")
		ctrl.refreshSettingsInfo()
	})

	// Settings: Resync App State
	settingsResyncBtn.ConnectClicked(func() {
		settingsResyncBtn.SetSensitive(false)
		settingsResyncBtn.SetLabel("Syncing…")
		go func() {
			if waClient != nil {
				for _, name := range appstate.AllPatchNames {
					_ = waClient.FetchAppState(context.Background(), name, true, false)
				}
			}
			if msgStore != nil {
				msgStore.importStickersFromChatHistory(context.Background(), waClient)
			}
			glib.IdleAdd(func() {
				settingsResyncBtn.SetSensitive(true)
				settingsResyncBtn.SetLabel("Sync Now")
				ctrl.loadChats()
				ctrl.loadArchivedChats()
				ctrl.loadSavedStickersUI()
				ctrl.refreshSettingsInfo()
			})
		}()
	})

	// Settings: Clear Avatar Cache
	settingsClearCacheBtn.ConnectClicked(func() {
		clearAvatarCache()
		settingsClearCacheBtn.SetLabel("Cleared!")
		glib.TimeoutAdd(2000, func() bool {
			settingsClearCacheBtn.SetLabel("Clear")
			return false
		})
	})

	// Settings: Theme Selector (System Default, Light, Dark)
	sm := adw.StyleManagerGetDefault()
	switch sm.ColorScheme() {
	case adw.ColorSchemeForceLight:
		settingsThemeRow.SetSelected(1)
	case adw.ColorSchemeForceDark:
		settingsThemeRow.SetSelected(2)
	default:
		settingsThemeRow.SetSelected(0)
	}
	settingsThemeRow.Connect("notify::selected", func() {
		switch settingsThemeRow.Selected() {
		case 1:
			sm.SetColorScheme(adw.ColorSchemeForceLight)
		case 2:
			sm.SetColorScheme(adw.ColorSchemeForceDark)
		default:
			sm.SetColorScheme(adw.ColorSchemeDefault)
		}
	})

	// Settings: Logout Button
	settingsLogoutBtn.ConnectClicked(func() {
		go func() {
			if waClient != nil {
				_ = waClient.Logout(context.Background())
			}
			glib.IdleAdd(func() {
				activeMainWindow = nil
				syncInfo = newSyncState(false)
				qrTexture = nil
				_ = showPage(pageInitial)
				go func() {
					client, err := newWhatsAppClient()
					if err == nil {
						connectWhatsApp(client)
					}
				}()
			})
		}()
	})

	// Chat selection handlers
	chatsList.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		if row == nil || ctrl.suppressChatSelection {
			return
		}
		idx := row.Index()
		if idx >= 0 && idx < len(ctrl.displayedChatRows) {
			item := ctrl.displayedChatRows[idx]
			if item.isCommunityHeader {
				ctrl.toggleCommunityExpanded(item.communityJID)
				return
			}
			target := item.chat
			if ctrl.currentChat != nil && (ctrl.currentChat.JID == target.JID || (ctrl.isSelfJID(ctrl.currentChat.JID) && ctrl.isSelfJID(target.JID))) {
				return
			}
			ctrl.selectChat(target)
		}
	})
	archivedChatsList.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		if row == nil || ctrl.suppressChatSelection {
			return
		}
		idx := row.Index()
		if idx >= 0 && idx < len(ctrl.displayedArchivedRows) {
			item := ctrl.displayedArchivedRows[idx]
			if item.isCommunityHeader {
				ctrl.toggleCommunityExpanded(item.communityJID)
				return
			}
			target := item.chat
			if ctrl.currentChat != nil && (ctrl.currentChat.JID == target.JID || (ctrl.isSelfJID(ctrl.currentChat.JID) && ctrl.isSelfJID(target.JID))) {
				return
			}
			ctrl.selectChat(target)
		}
	})

	if callsList != nil {
		callsList.ConnectRowSelected(func(row *gtk.ListBoxRow) {
			if row == nil {
				return
			}
			idx := row.Index()
			if idx >= 0 && idx < len(ctrl.calls) {
				ctrl.openCallChat(ctrl.calls[idx])
			}
		})
	}

	// Message sending & reply controls
	sendButton.ConnectClicked(func() {
		ctrl.sendMessage()
	})
	messageEntry.ConnectActivate(func() {
		ctrl.sendMessage()
	})

	if cancelReplyBtn != nil {
		cancelReplyBtn.ConnectClicked(func() {
			ctrl.cancelReply()
		})
	}
	if cancelEditBtn != nil {
		cancelEditBtn.ConnectClicked(func() {
			ctrl.cancelEditing()
		})
	}

	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(val uint, code uint, state gdk.ModifierType) bool {
		if val == gdk.KEY_Escape {
			if ctrl.editingMessage != nil {
				ctrl.cancelEditing()
				return true
			}
			if ctrl.replyingTo != nil {
				ctrl.cancelReply()
				return true
			}
		}
		return false
	})
	messageEntry.AddController(keyController)

	messageEntry.ConnectChanged(func() {
		text := messageEntry.Text()

		// 1. Live preview directly inside message box using Pango attributes
		if attrs, err := FormatWhatsAppMarkdownForEntry(text); err == nil && attrs != nil {
			messageEntry.SetAttributes(attrs)
		} else {
			messageEntry.SetAttributes(pango.NewAttrList())
		}

		// 2. Live preview bar above message box
		if HasWhatsAppMarkdown(text) && strings.TrimSpace(text) != "" {
			previewMarkup, _ := FormatWhatsAppMarkdownForBubble(text, nil, nil)
			if ctrl.markdownPreviewLabel != nil {
				ctrl.markdownPreviewLabel.SetMarkup(previewMarkup)
			}
			if ctrl.markdownPreviewBar != nil {
				ctrl.markdownPreviewBar.SetVisible(true)
			}
		} else {
			if ctrl.markdownPreviewBar != nil {
				ctrl.markdownPreviewBar.SetVisible(false)
			}
		}
	})

	// Populate chat lists & settings info
	ctrl.setupVoiceRecording()
	ctrl.loadChats()
	ctrl.loadArchivedChats()
	ctrl.refreshSettingsInfo()

	// Sync stickers used from phone in background
	go func() {
		if msgStore != nil {
			msgStore.importStickersFromChatHistory(context.Background(), waClient)
		}
	}()

	return window, nil
}

// refreshSettingsInfo loads current account, storage, and stats info into the settings page.
func (c *whatsDownMainController) refreshSettingsInfo() {
	if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
		pushName := waClient.Store.PushName
		if pushName == "" {
			pushName = "WhatsApp User"
		}
		phone := "+" + waClient.Store.ID.User
		c.settingsProfileRow.SetTitle(pushName)
		c.settingsProfileRow.SetSubtitle(phone)
		ownJID := waClient.Store.ID.ToNonAD().String()
		loadAvatar(ownJID, pushName, c.settingsProfileAvatar)
	}

	dbPath, err := databasePath()
	if err == nil {
		c.settingsDbPathRow.SetSubtitle(dbPath)
	}

	if msgStore != nil {
		go func() {
			chats, msgs, _ := msgStore.getStats(context.Background())
			glib.IdleAdd(func() {
				c.settingsStatsRow.SetSubtitle(fmt.Sprintf("%d chats • %d messages", chats, msgs))
			})
		}()
	}
}

// loadChats loads unarchived chats from the message store and populates the sidebar.
func (c *whatsDownMainController) loadChats() {
	if msgStore == nil {
		return
	}
	c.suppressChatSelection = true
	defer func() {
		c.suppressChatSelection = false
	}()

	chats, err := msgStore.listChats(context.Background())
	if err != nil {
		log.Printf("loading chats: %v", err)
		return
	}
	c.chats = chats
	c.renderChatList(chats, false)
}

// loadArchivedChats loads archived chats and populates the archived list.
func (c *whatsDownMainController) loadArchivedChats() {
	if msgStore == nil {
		return
	}
	c.suppressChatSelection = true
	defer func() {
		c.suppressChatSelection = false
	}()

	chats, err := msgStore.listArchivedChats(context.Background())
	if err != nil {
		log.Printf("loading archived chats: %v", err)
		return
	}
	c.archivedChats = chats
	c.renderChatList(chats, true)
}

func (c *whatsDownMainController) toggleCommunityExpanded(communityJID string) {
	c.expandedCommunities[communityJID] = !c.expandedCommunities[communityJID]
	c.loadChats()
	c.loadArchivedChats()
}

// renderChatList renders chats with community folders and child indentation.
func (c *whatsDownMainController) renderChatList(chats []Chat, isArchived bool) {
	listBox := c.chatsList
	searchFilter := c.chatsSearchFilter
	if isArchived {
		listBox = c.archivedChatsList
		searchFilter = c.archivedSearchFilter
	}
	if listBox == nil {
		return
	}
	listBox.RemoveAll()

	if isArchived && len(chats) == 0 {
		c.displayedArchivedRows = nil
		emptyRow := gtk.NewListBoxRow()
		emptyRow.SetSelectable(false)
		emptyRow.SetActivatable(false)
		box := gtk.NewBox(gtk.OrientationVertical, 8)
		box.SetMarginTop(32)
		box.SetMarginBottom(32)
		box.SetHAlign(gtk.AlignCenter)

		icon := gtk.NewImageFromIconName("folder-download-symbolic")
		icon.SetPixelSize(36)
		icon.AddCSSClass("dim-label")
		box.Append(icon)

		lbl := gtk.NewLabel("No archived chats")
		lbl.AddCSSClass("dim-label")
		box.Append(lbl)

		emptyRow.SetChild(box)
		listBox.Append(emptyRow)
		return
	}

	searchLower := strings.ToLower(strings.TrimSpace(searchFilter))

	// Group communities and children
	communityMap := make(map[string]*Chat)
	childrenMap := make(map[string][]Chat)

	for _, chat := range chats {
		if chat.IsParent {
			cp := chat
			communityMap[chat.JID] = &cp
		}
		if chat.ParentJID != "" {
			childrenMap[chat.ParentJID] = append(childrenMap[chat.ParentJID], chat)
			if _, exists := communityMap[chat.ParentJID]; !exists {
				communityMap[chat.ParentJID] = &Chat{
					JID:      chat.ParentJID,
					Name:     "Community",
					IsParent: true,
				}
			}
		}
	}

	// Aggregate latest message time and unread counts for communities
	for pJID, comm := range communityMap {
		children := childrenMap[pJID]
		for _, ch := range children {
			comm.UnreadCount += ch.UnreadCount
			if ch.LastMessageAt.After(comm.LastMessageAt) {
				comm.LastMessageAt = ch.LastMessageAt
				comm.LastMessageText = ch.Name + ": " + ch.LastMessageText
			}
		}
	}

	// Auto-expand community if search matches any child
	if searchLower != "" {
		for pJID, children := range childrenMap {
			for _, ch := range children {
				if strings.Contains(strings.ToLower(ch.Name), searchLower) || strings.Contains(strings.ToLower(ch.LastMessageText), searchLower) {
					c.expandedCommunities[pJID] = true
					break
				}
			}
		}
	}

	// Build top-level chats list: exclude any chat whose ParentJID != ""
	topLevel := make([]*Chat, 0, len(chats))
	addedCommunities := make(map[string]bool)

	for i := range chats {
		chat := &chats[i]
		if chat.ParentJID != "" {
			if !addedCommunities[chat.ParentJID] {
				if comm := communityMap[chat.ParentJID]; comm != nil {
					topLevel = append(topLevel, comm)
					addedCommunities[chat.ParentJID] = true
				}
			}
			continue
		}
		if chat.IsParent {
			if !addedCommunities[chat.JID] {
				if comm := communityMap[chat.JID]; comm != nil {
					topLevel = append(topLevel, comm)
					addedCommunities[chat.JID] = true
				}
			}
			continue
		}
		topLevel = append(topLevel, chat)
	}

	// Sort top-level chats: pinned first, then newest activity
	sort.SliceStable(topLevel, func(i, j int) bool {
		if topLevel[i].Pinned != topLevel[j].Pinned {
			return topLevel[i].Pinned
		}
		return topLevel[i].LastMessageAt.After(topLevel[j].LastMessageAt)
	})

	var displayedRows []chatRowItem

	for _, item := range topLevel {
		if item.IsParent {
			children := childrenMap[item.JID]
			// Sort children: default sub group (Announcements) first, then recent
			sort.SliceStable(children, func(i, j int) bool {
				if children[i].IsDefaultSubGroup != children[j].IsDefaultSubGroup {
					return children[i].IsDefaultSubGroup
				}
				return children[i].LastMessageAt.After(children[j].LastMessageAt)
			})

			commMatches := searchLower == "" || strings.Contains(strings.ToLower(item.Name), searchLower)
			hasMatchingChild := false
			if searchLower != "" {
				for _, ch := range children {
					if strings.Contains(strings.ToLower(ch.Name), searchLower) || strings.Contains(strings.ToLower(ch.LastMessageText), searchLower) {
						hasMatchingChild = true
						break
					}
				}
			}
			if searchLower != "" && !commMatches && !hasMatchingChild {
				continue
			}

			isExpanded := c.expandedCommunities[item.JID]
			commRow := c.createCommunityHeaderRow(*item, len(children), isExpanded)
			listBox.Append(commRow)
			displayedRows = append(displayedRows, chatRowItem{
				isCommunityHeader: true,
				communityJID:      item.JID,
				chat:              *item,
			})

			if isExpanded {
				for _, child := range children {
					if searchLower != "" && !commMatches && !strings.Contains(strings.ToLower(child.Name), searchLower) && !strings.Contains(strings.ToLower(child.LastMessageText), searchLower) {
						continue
					}
					childRow := c.createCommunityChildRow(child, isArchived)
					listBox.Append(childRow)
					displayedRows = append(displayedRows, chatRowItem{
						isCommunityHeader: false,
						communityJID:      item.JID,
						chat:              child,
					})
					if c.currentChat != nil && (c.currentChat.JID == child.JID || (c.isSelfJID(c.currentChat.JID) && c.isSelfJID(child.JID))) {
						listBox.SelectRow(childRow)
					}
				}
			}
		} else {
			if searchLower != "" && !strings.Contains(strings.ToLower(item.Name), searchLower) && !strings.Contains(strings.ToLower(item.LastMessageText), searchLower) {
				continue
			}
			row := c.createChatRow(*item, isArchived)
			listBox.Append(row)
			displayedRows = append(displayedRows, chatRowItem{
				isCommunityHeader: false,
				chat:              *item,
			})
			if c.currentChat != nil && (c.currentChat.JID == item.JID || (c.isSelfJID(c.currentChat.JID) && c.isSelfJID(item.JID))) {
				listBox.SelectRow(row)
			}
		}
	}

	if isArchived {
		c.displayedArchivedRows = displayedRows
	} else {
		c.displayedChatRows = displayedRows
	}
}

func (c *whatsDownMainController) createCommunityHeaderRow(comm Chat, childCount int, isExpanded bool) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-row")
	row.AddCSSClass("community-folder-row")

	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	mainBox.SetHExpand(true)

	displayName := comm.Name
	if displayName == "" {
		displayName = "Community"
	}

	avatar := adw.NewAvatar(40, displayName, true)
	avatar.SetVAlign(gtk.AlignCenter)
	loadAvatar(comm.JID, displayName, avatar)
	mainBox.Append(avatar)

	textBox := gtk.NewBox(gtk.OrientationVertical, 4)
	textBox.SetHExpand(true)
	textBox.SetVAlign(gtk.AlignCenter)

	titleLabel := gtk.NewLabel(displayName)
	titleLabel.SetXAlign(0)
	titleLabel.AddCSSClass("chat-row-title")
	titleLabel.AddCSSClass("community-title")
	titleLabel.SetEllipsize(pango.EllipsizeEnd)
	textBox.Append(titleLabel)

	subText := fmt.Sprintf("%d chats", childCount)
	if childCount == 1 {
		subText = "1 chat"
	}
	subLabel := gtk.NewLabel(subText)
	subLabel.SetXAlign(0)
	subLabel.AddCSSClass("chat-row-subtitle")
	subLabel.SetEllipsize(pango.EllipsizeEnd)
	textBox.Append(subLabel)
	mainBox.Append(textBox)

	metaBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	metaBox.SetVAlign(gtk.AlignCenter)

	if comm.UnreadCount > 0 {
		badge := gtk.NewLabel(fmt.Sprintf("%d", comm.UnreadCount))
		badge.AddCSSClass("unread-badge")
		badge.SetXAlign(0.5)
		metaBox.Append(badge)
	}

	chevronIconName := "pan-end-symbolic"
	if isExpanded {
		chevronIconName = "pan-down-symbolic"
	}
	chevron := gtk.NewImageFromIconName(chevronIconName)
	chevron.AddCSSClass("dim-label")
	chevron.AddCSSClass("community-chevron")
	metaBox.Append(chevron)

	mainBox.Append(metaBox)
	row.SetChild(mainBox)

	return row
}

func (c *whatsDownMainController) createCommunityChildRow(chat Chat, isArchivedView bool) *gtk.ListBoxRow {
	row := c.createChatRow(chat, isArchivedView)
	row.AddCSSClass("community-child-row")
	return row
}

// loadCalls loads call history from the message store and populates the calls list.
func (c *whatsDownMainController) loadCalls() {
	if msgStore == nil || c.callsList == nil {
		return
	}
	query := ""
	if c.callsSearchEntry != nil {
		query = c.callsSearchEntry.Text()
	}
	calls, err := msgStore.listCalls(context.Background(), query)
	if err != nil {
		log.Printf("loading calls: %v", err)
		return
	}
	c.calls = calls
	c.callsList.RemoveAll()

	if len(calls) == 0 {
		emptyRow := gtk.NewListBoxRow()
		emptyRow.SetSelectable(false)
		emptyRow.SetActivatable(false)
		box := gtk.NewBox(gtk.OrientationVertical, 8)
		box.SetMarginTop(32)
		box.SetMarginBottom(32)
		box.SetHAlign(gtk.AlignCenter)

		icon := gtk.NewImageFromIconName("call-start-symbolic")
		icon.SetPixelSize(36)
		icon.AddCSSClass("dim-label")
		box.Append(icon)

		lbl := gtk.NewLabel("No calls yet")
		lbl.AddCSSClass("dim-label")
		box.Append(lbl)

		emptyRow.SetChild(box)
		c.callsList.Append(emptyRow)
		return
	}

	for _, call := range calls {
		row := c.createCallRow(call)
		c.callsList.Append(row)
	}
}

func (c *whatsDownMainController) createCallRow(call CallRecord) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-row")
	row.AddCSSClass("call-row")

	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	mainBox.SetHExpand(true)

	displayName := call.CallerName
	if displayName == "" {
		displayName = cleanPhoneNumber(call.CallerJID)
	}

	avatar := adw.NewAvatar(40, displayName, true)
	avatar.SetVAlign(gtk.AlignCenter)
	loadAvatar(call.CallerJID, displayName, avatar)
	mainBox.Append(avatar)

	textBox := gtk.NewBox(gtk.OrientationVertical, 4)
	textBox.SetHExpand(true)
	textBox.SetVAlign(gtk.AlignCenter)

	titleLabel := gtk.NewLabel(displayName)
	titleLabel.SetXAlign(0)
	titleLabel.AddCSSClass("chat-row-title")
	titleLabel.SetEllipsize(pango.EllipsizeEnd)
	if call.Outcome == CallOutcomeMissed && !call.FromMe {
		titleLabel.AddCSSClass("call-title-missed")
	}
	textBox.Append(titleLabel)

	subBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	subBox.SetVAlign(gtk.AlignCenter)

	var iconName string
	var iconClass string
	if call.FromMe {
		iconName = "call-outgoing-symbolic"
		iconClass = "call-icon-outgoing"
	} else if call.Outcome == CallOutcomeMissed {
		iconName = "call-missed-symbolic"
		iconClass = "call-icon-missed"
	} else {
		iconName = "call-incoming-symbolic"
		iconClass = "call-icon-incoming"
	}

	callIcon := gtk.NewImageFromIconName(iconName)
	callIcon.AddCSSClass(iconClass)
	subBox.Append(callIcon)

	if call.IsVideo {
		vidIcon := gtk.NewImageFromIconName("camera-video-symbolic")
		vidIcon.AddCSSClass("dim-label")
		subBox.Append(vidIcon)
	}

	detailText := formatChatTime(call.Timestamp)
	if call.DurationSecs > 0 {
		detailText = fmt.Sprintf("%s (%s)", detailText, formatCallDuration(call.DurationSecs))
	} else if call.Outcome == CallOutcomeMissed {
		detailText = fmt.Sprintf("%s · Missed", detailText)
	}
	subLabel := gtk.NewLabel(detailText)
	subLabel.SetXAlign(0)
	subLabel.AddCSSClass("chat-row-subtitle")
	if call.Outcome == CallOutcomeMissed && !call.FromMe {
		subLabel.AddCSSClass("call-subtitle-missed")
	}
	subBox.Append(subLabel)
	textBox.Append(subBox)

	mainBox.Append(textBox)

	actionBtn := gtk.NewButton()
	actionBtn.AddCSSClass("flat")
	actionBtn.AddCSSClass("circular")
	actionBtn.SetVAlign(gtk.AlignCenter)
	actionBtnIcon := gtk.NewImageFromIconName("chat-message-new-symbolic")
	actionBtn.SetChild(actionBtnIcon)
	actionBtn.SetTooltipText("Open chat")
	actionBtn.ConnectClicked(func() {
		c.openCallChat(call)
	})
	mainBox.Append(actionBtn)

	row.SetChild(mainBox)
	c.attachCallContextMenu(row, call)

	return row
}

func (c *whatsDownMainController) openCallChat(call CallRecord) {
	chatJID := call.ChatJID
	if chatJID == "" {
		chatJID = call.CallerJID
	}
	targetChat, err := msgStore.getChatOrContact(context.Background(), chatJID)
	if err != nil || targetChat.JID == "" {
		targetChat = Chat{
			JID:  chatJID,
			Name: call.CallerName,
		}
	}
	c.selectChat(targetChat)

	c.sidebarStack.SetVisibleChildName("chats")
	c.chatsNavBtn.AddCSSClass("active")
	if c.callsNavBtn != nil {
		c.callsNavBtn.RemoveCSSClass("active")
	}
	if c.archivedNavBtn != nil {
		c.archivedNavBtn.RemoveCSSClass("active")
	}
	if c.settingsNavBtn != nil {
		c.settingsNavBtn.RemoveCSSClass("active")
	}
	c.contentStack.SetVisibleChildName("chat")
}

func (c *whatsDownMainController) attachCallContextMenu(row *gtk.ListBoxRow, call CallRecord) {
	popover := gtk.NewPopover()
	popover.SetParent(row)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	openBtn := gtk.NewButton()
	openBtn.AddCSSClass("flat")
	openBtn.AddCSSClass("popover-menu-btn")
	openBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	openIcon := gtk.NewImageFromIconName("chat-message-new-symbolic")
	openLabel := gtk.NewLabel("Open Chat")
	openBtnBox.Append(openIcon)
	openBtnBox.Append(openLabel)
	openBtn.SetChild(openBtnBox)
	openBtn.ConnectClicked(func() {
		popover.Popdown()
		c.openCallChat(call)
	})
	menuBox.Append(openBtn)

	delBtn := gtk.NewButton()
	delBtn.AddCSSClass("flat")
	delBtn.AddCSSClass("popover-menu-btn")
	delBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	delIcon := gtk.NewImageFromIconName("user-trash-symbolic")
	delLabel := gtk.NewLabel("Delete from History")
	delBtnBox.Append(delIcon)
	delBtnBox.Append(delLabel)
	delBtn.SetChild(delBtnBox)
	delBtn.ConnectClicked(func() {
		popover.Popdown()
		if msgStore != nil {
			go func() {
				_ = msgStore.deleteCall(context.Background(), call.ID)
				glib.IdleAdd(func() {
					c.loadCalls()
				})
			}()
		}
	})
	menuBox.Append(delBtn)

	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	gesture.ConnectPressed(func(n int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	row.AddController(gesture)
}

// createChatRow creates a styled list row representing a single chat with context menu.
func (c *whatsDownMainController) createChatRow(chat Chat, isArchivedView bool) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-row")

	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	mainBox.SetHExpand(true)

	displayName := formatDisplayName(chat)
	subText := formatChatSubtitle(chat)

	// Avatar with fallback initials and async profile picture
	avatar := adw.NewAvatar(40, displayName, true)
	avatar.SetVAlign(gtk.AlignCenter)
	loadAvatar(chat.JID, displayName, avatar)
	mainBox.Append(avatar)

	// Left: chat name and subtitle
	textBox := gtk.NewBox(gtk.OrientationVertical, 4)
	textBox.SetHExpand(true)
	textBox.SetVAlign(gtk.AlignCenter)

	titleLabel := gtk.NewLabel(displayName)
	titleLabel.SetXAlign(0)
	titleLabel.AddCSSClass("chat-row-title")
	titleLabel.SetEllipsize(pango.EllipsizeEnd)
	textBox.Append(titleLabel)

	if subText != "" {
		subLabel := gtk.NewLabel(subText)
		subLabel.SetXAlign(0)
		subLabel.AddCSSClass("chat-row-subtitle")
		subLabel.SetEllipsize(pango.EllipsizeEnd)
		subLabel.SetSingleLineMode(true)
		textBox.Append(subLabel)
	}
	mainBox.Append(textBox)

	// Right: timestamp, pinned indicator, and unread badge
	metaBox := gtk.NewBox(gtk.OrientationVertical, 4)
	metaBox.SetVAlign(gtk.AlignStart)

	if !chat.LastMessageAt.IsZero() {
		timeStr := formatChatTime(chat.LastMessageAt)
		timeLabel := gtk.NewLabel(timeStr)
		timeLabel.SetXAlign(1)
		timeLabel.AddCSSClass("chat-row-time")
		metaBox.Append(timeLabel)
	}

	subMetaBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	subMetaBox.SetHAlign(gtk.AlignEnd)

	if chat.Muted {
		muteIcon := gtk.NewImageFromIconName("audio-volume-muted-symbolic")
		muteIcon.AddCSSClass("pin-indicator")
		subMetaBox.Append(muteIcon)
	}

	if chat.Pinned {
		pinIcon := gtk.NewImageFromIconName("view-pin-symbolic")
		pinIcon.AddCSSClass("pin-indicator")
		subMetaBox.Append(pinIcon)
	}

	if chat.UnreadCount > 0 {
		badge := gtk.NewLabel(fmt.Sprintf("%d", chat.UnreadCount))
		badge.AddCSSClass("unread-badge")
		badge.SetXAlign(0.5)
		subMetaBox.Append(badge)
	}

	metaBox.Append(subMetaBox)
	mainBox.Append(metaBox)
	row.SetChild(mainBox)

	// Attach right-click context menu
	c.attachContextMenu(row, chat, isArchivedView)

	return row
}

// attachContextMenu attaches a right-click popover context menu to a chat row.
func (c *whatsDownMainController) attachContextMenu(row *gtk.ListBoxRow, chat Chat, isArchivedView bool) {
	popover := gtk.NewPopover()
	popover.SetParent(row)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	// 1. Pin / Unpin
	pinText := "Pin chat"
	if chat.Pinned {
		pinText = "Unpin chat"
	}
	pinBtn := gtk.NewButton()
	pinBtn.AddCSSClass("flat")
	pinBtn.AddCSSClass("popover-menu-btn")
	pinBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	pinIcon := gtk.NewImageFromIconName("view-pin-symbolic")
	pinLabel := gtk.NewLabel(pinText)
	pinBtnBox.Append(pinIcon)
	pinBtnBox.Append(pinLabel)
	pinBtn.SetChild(pinBtnBox)
	pinBtn.ConnectClicked(func() {
		popover.Popdown()
		c.togglePin(chat)
	})
	menuBox.Append(pinBtn)

	// 2. Mute / Unmute
	muteText := "Mute chat"
	muteIconName := "audio-volume-muted-symbolic"
	if chat.Muted {
		muteText = "Unmute chat"
		muteIconName = "audio-volume-high-symbolic"
	}
	muteBtn := gtk.NewButton()
	muteBtn.AddCSSClass("flat")
	muteBtn.AddCSSClass("popover-menu-btn")
	muteBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	muteIcon := gtk.NewImageFromIconName(muteIconName)
	muteLabel := gtk.NewLabel(muteText)
	muteBtnBox.Append(muteIcon)
	muteBtnBox.Append(muteLabel)
	muteBtn.SetChild(muteBtnBox)
	muteBtn.ConnectClicked(func() {
		popover.Popdown()
		c.toggleMute(chat)
	})
	menuBox.Append(muteBtn)

	// 3. See Details and All Media with user
	detailsBtn := gtk.NewButton()
	detailsBtn.AddCSSClass("flat")
	detailsBtn.AddCSSClass("popover-menu-btn")
	detailsBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	detailsIcon := gtk.NewImageFromIconName("help-about-symbolic")
	detailsLabel := gtk.NewLabel("Details & Media")
	detailsBtnBox.Append(detailsIcon)
	detailsBtnBox.Append(detailsLabel)
	detailsBtn.SetChild(detailsBtnBox)
	detailsBtn.ConnectClicked(func() {
		popover.Popdown()
		c.showDetailsAndMedia(chat)
	})
	menuBox.Append(detailsBtn)

	// Separator
	sep := gtk.NewSeparator(gtk.OrientationHorizontal)
	sep.SetMarginTop(2)
	sep.SetMarginBottom(2)
	menuBox.Append(sep)

	// 4. Archive / Unarchive
	archiveText := "Archive chat"
	archiveIconName := "folder-download-symbolic"
	if chat.Archived {
		archiveText = "Unarchive chat"
		archiveIconName = "folder-symbolic"
	}
	archiveBtn := gtk.NewButton()
	archiveBtn.AddCSSClass("flat")
	archiveBtn.AddCSSClass("popover-menu-btn")
	archiveBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	archiveIcon := gtk.NewImageFromIconName(archiveIconName)
	archiveLabel := gtk.NewLabel(archiveText)
	archiveBtnBox.Append(archiveIcon)
	archiveBtnBox.Append(archiveLabel)
	archiveBtn.SetChild(archiveBtnBox)
	archiveBtn.ConnectClicked(func() {
		popover.Popdown()
		c.toggleArchive(chat)
	})
	menuBox.Append(archiveBtn)

	popover.SetChild(menuBox)

	// Right-click gesture
	gesture := gtk.NewGestureClick()
	gesture.SetButton(gdk.BUTTON_SECONDARY)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	row.AddController(gesture)
}

// togglePin flips the pinned state of a chat and refreshes the lists.
func (c *whatsDownMainController) togglePin(chat Chat) {
	if msgStore == nil {
		return
	}
	newPinned := !chat.Pinned
	go func() {
		if err := msgStore.setChatPinned(context.Background(), chat.JID, newPinned); err != nil {
			log.Printf("toggling pin for %s: %v", chat.JID, err)
			return
		}
		if waClient != nil {
			if targetJID, err := types.ParseJID(chat.JID); err == nil {
				if err := waClient.SendAppState(context.Background(), appstate.BuildPin(targetJID, newPinned)); err != nil {
					log.Printf("syncing pin to WhatsApp: %v", err)
				}
			}
		}
		glib.IdleAdd(func() {
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}

// toggleMute flips the muted state of a chat and refreshes the lists.
func (c *whatsDownMainController) toggleMute(chat Chat) {
	if msgStore == nil {
		return
	}
	newMuted := !chat.Muted
	go func() {
		if err := msgStore.setChatMuted(context.Background(), chat.JID, newMuted); err != nil {
			log.Printf("toggling mute for %s: %v", chat.JID, err)
			return
		}
		if waClient != nil {
			if targetJID, err := types.ParseJID(chat.JID); err == nil {
				if err := waClient.SendAppState(context.Background(), appstate.BuildMute(targetJID, newMuted, 0)); err != nil {
					log.Printf("syncing mute to WhatsApp: %v", err)
				}
			}
		}
		glib.IdleAdd(func() {
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}

// toggleArchive flips the archived state of a chat and refreshes the lists.
func (c *whatsDownMainController) toggleArchive(chat Chat) {
	if msgStore == nil {
		return
	}
	newArchived := !chat.Archived
	go func() {
		if err := msgStore.setChatArchived(context.Background(), chat.JID, newArchived); err != nil {
			log.Printf("toggling archive for %s: %v", chat.JID, err)
			return
		}
		if waClient != nil {
			if targetJID, err := types.ParseJID(chat.JID); err == nil {
				if err := waClient.SendAppState(context.Background(), appstate.BuildArchive(targetJID, newArchived, time.Time{}, nil)); err != nil {
					log.Printf("syncing archive to WhatsApp: %v", err)
				}
			}
		}
		glib.IdleAdd(func() {
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}

// showDetailsAndMedia opens an adaptive dialog with contact details and all shared media.
func (c *whatsDownMainController) showDetailsAndMedia(chat Chat) {
	dialog := adw.NewDialog()
	displayName := formatDisplayName(chat)
	dialog.SetTitle(displayName)
	dialog.SetContentWidth(460)
	dialog.SetContentHeight(600)

	toolbarView := adw.NewToolbarView()
	headerBar := adw.NewHeaderBar()
	toolbarView.AddTopBar(headerBar)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 16)
	contentBox.SetMarginTop(18)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(18)
	contentBox.SetMarginEnd(18)

	// Header section: Avatar, Name, Phone
	headerBox := gtk.NewBox(gtk.OrientationVertical, 8)
	headerBox.SetHAlign(gtk.AlignCenter)

	avatar := adw.NewAvatar(80, displayName, true)
	avatar.SetHAlign(gtk.AlignCenter)
	loadAvatar(chat.JID, displayName, avatar)
	headerBox.Append(avatar)

	nameLabel := gtk.NewLabel(displayName)
	nameLabel.AddCSSClass("title-1")
	nameLabel.SetHAlign(gtk.AlignCenter)
	nameLabel.SetWrap(true)
	nameLabel.SetWrapMode(pango.WrapWordChar)
	headerBox.Append(nameLabel)

	phoneSubtitle := formatChatSubtitle(chat)
	if phoneSubtitle == "" {
		if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
			phoneSubtitle = cleanPhoneNumber(chat.JID)
		}
	}
	if phoneSubtitle != "" && phoneSubtitle != displayName {
		subLabel := gtk.NewLabel(phoneSubtitle)
		subLabel.AddCSSClass("dim-label")
		subLabel.SetHAlign(gtk.AlignCenter)
		headerBox.Append(subLabel)
	}

	// Status chips / badges
	chipBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	chipBox.SetHAlign(gtk.AlignCenter)
	chipBox.SetMarginTop(4)

	chatType := "Direct Message"
	if strings.HasSuffix(chat.JID, "@g.us") {
		chatType = "Group Chat"
	} else if strings.HasSuffix(chat.JID, "@newsletter") {
		chatType = "Channel"
	} else if strings.HasSuffix(chat.JID, "@broadcast") {
		chatType = "Broadcast"
	}
	typeBadge := gtk.NewLabel(chatType)
	typeBadge.AddCSSClass("caption")
	typeBadge.AddCSSClass("card")
	typeBadge.AddCSSClass("status-badge")
	chipBox.Append(typeBadge)

	if chat.Muted {
		muteBadge := gtk.NewLabel("🔇 Muted")
		muteBadge.AddCSSClass("caption")
		muteBadge.AddCSSClass("card")
		muteBadge.AddCSSClass("status-badge")
		chipBox.Append(muteBadge)
	}

	if chat.Pinned {
		pinBadge := gtk.NewLabel("📌 Pinned")
		pinBadge.AddCSSClass("caption")
		pinBadge.AddCSSClass("card")
		pinBadge.AddCSSClass("status-badge")
		chipBox.Append(pinBadge)
	}

	if chat.Archived {
		archBadge := gtk.NewLabel("📁 Archived")
		archBadge.AddCSSClass("caption")
		archBadge.AddCSSClass("card")
		archBadge.AddCSSClass("status-badge")
		chipBox.Append(archBadge)
	}
	headerBox.Append(chipBox)
	contentBox.Append(headerBox)

	// Conversation Info Group
	infoGroup := adw.NewPreferencesGroup()
	infoGroup.SetTitle("Conversation Details")

	jidRow := adw.NewActionRow()
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		jidRow.SetTitle("Phone Number")
		jidRow.SetSubtitle(cleanPhoneNumber(chat.JID))
		jidRow.AddPrefix(gtk.NewImageFromIconName("call-start-symbolic"))
	} else if strings.HasSuffix(chat.JID, "@lid") {
		jidRow.SetTitle("Account")
		if strings.HasPrefix(chat.Name, "+") {
			jidRow.SetSubtitle(chat.Name)
		} else {
			jidRow.SetSubtitle("Linked WhatsApp Account")
		}
		jidRow.AddPrefix(gtk.NewImageFromIconName("user-available-symbolic"))
	} else if strings.HasSuffix(chat.JID, "@g.us") {
		jidRow.SetTitle("Chat Type")
		jidRow.SetSubtitle("WhatsApp Group")
		jidRow.AddPrefix(gtk.NewImageFromIconName("system-users-symbolic"))
	} else {
		jidRow.SetTitle("Conversation")
		jidRow.SetSubtitle(displayName)
		jidRow.AddPrefix(gtk.NewImageFromIconName("chat-message-new-symbolic"))
	}
	jidRow.SetSubtitleSelectable(true)
	infoGroup.Add(jidRow)

	if !chat.LastMessageAt.IsZero() {
		lastActiveRow := adw.NewActionRow()
		lastActiveRow.SetTitle("Last Active")
		lastActiveRow.SetSubtitle(chat.LastMessageAt.Format("02 Jan 2006, 15:04"))
		lastActiveRow.AddPrefix(gtk.NewImageFromIconName("document-open-recent-symbolic"))
		infoGroup.Add(lastActiveRow)
	}
	contentBox.Append(infoGroup)

	// Tabbed shared content: Media, Docs, Audio, Starred
	if msgStore != nil {
		mediaMsgs, _ := msgStore.listMediaMessages(context.Background(), chat.JID)
		docMsgs, _ := msgStore.listDocMessages(context.Background(), chat.JID)
		audioMsgs, _ := msgStore.listAudioMessages(context.Background(), chat.JID)
		starredMsgs, _ := msgStore.listStarredMessages(context.Background(), chat.JID)

		stack := gtk.NewStack()
		stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
		stack.SetVhomogeneous(false)
		stack.SetHhomogeneous(true)
		stack.SetInterpolateSize(true)

		// 1. Media Tab (Photos & Videos)
		mediaBox := gtk.NewBox(gtk.OrientationVertical, 8)
		mediaBox.SetMarginTop(8)
		if len(mediaMsgs) == 0 {
			emptyGroup := adw.NewPreferencesGroup()
			emptyRow := adw.NewActionRow()
			emptyRow.SetTitle("No photos or videos")
			emptyRow.SetSubtitle("Shared photos and videos will appear here")
			emptyRow.AddPrefix(gtk.NewImageFromIconName("folder-pictures-symbolic"))
			emptyGroup.Add(emptyRow)
			mediaBox.Append(emptyGroup)
		} else {
			flowBox := gtk.NewFlowBox()
			flowBox.SetSelectionMode(gtk.SelectionNone)
			flowBox.SetHomogeneous(true)
			flowBox.SetMinChildrenPerLine(3)
			flowBox.SetMaxChildrenPerLine(3)
			flowBox.SetColumnSpacing(8)
			flowBox.SetRowSpacing(8)
			flowBox.AddCSSClass("media-gallery-flowbox")

			for _, m := range mediaMsgs {
				overlay := gtk.NewOverlay()
				overlay.AddCSSClass("media-tile")

				pic := gtk.NewPicture()
				pic.SetKeepAspectRatio(true)
				pic.SetCanShrink(true)
				pic.AddCSSClass("media-tile-picture")
				pic.SetSizeRequest(128, 128)
				loadMediaPreview(m, pic, nil)
				overlay.SetChild(pic)

				if m.Kind == messageKindVideo {
					badge := gtk.NewLabel("▶ Video")
					if e2e := parseE2EMessage(m.Raw); e2e != nil && e2e.GetVideoMessage() != nil {
						if s := e2e.GetVideoMessage().GetSeconds(); s > 0 {
							badge.SetText(fmt.Sprintf("▶ %d:%02d", s/60, s%60))
						}
					}
					badge.AddCSSClass("media-video-badge")
					badge.SetHAlign(gtk.AlignStart)
					badge.SetVAlign(gtk.AlignEnd)
					badge.SetMarginStart(6)
					badge.SetMarginBottom(6)
					overlay.AddOverlay(badge)
				}

				mCopy := m
				leftClick := gtk.NewGestureClick()
				leftClick.ConnectPressed(func(nPress int, x, y float64) {
					openMedia(mCopy, nil)
				})
				overlay.AddController(leftClick)

				c.attachMediaContextMenu(overlay, mCopy, dialog)

				flowBox.Append(overlay)
			}
			mediaBox.Append(flowBox)
		}
		stack.AddTitled(mediaBox, "media", fmt.Sprintf("Media (%d)", len(mediaMsgs)))

		// 2. Docs Tab
		docsBox := gtk.NewBox(gtk.OrientationVertical, 8)
		docsBox.SetMarginTop(8)
		if len(docMsgs) == 0 {
			emptyGroup := adw.NewPreferencesGroup()
			emptyRow := adw.NewActionRow()
			emptyRow.SetTitle("No documents")
			emptyRow.SetSubtitle("Shared documents and files will appear here")
			emptyRow.AddPrefix(gtk.NewImageFromIconName("folder-documents-symbolic"))
			emptyGroup.Add(emptyRow)
			docsBox.Append(emptyGroup)
		} else {
			docGroup := adw.NewPreferencesGroup()
			for _, m := range docMsgs {
				row := adw.NewActionRow()
				iconName := "folder-documents-symbolic"
				titleText := "Document"
				e2e := parseE2EMessage(m.Raw)
				if e2e != nil && e2e.GetDocumentMessage() != nil {
					doc := e2e.GetDocumentMessage()
					if doc.GetFileName() != "" {
						titleText = doc.GetFileName()
					}
					if sz := formatFileSize(doc.GetFileLength()); sz != "" {
						titleText += " • " + sz
					}
				}
				row.AddPrefix(gtk.NewImageFromIconName(iconName))
				row.SetTitle(titleText)
				sub := m.Timestamp.Format("02 Jan 15:04")
				if m.FromMe {
					sub = "You • " + sub
				} else if m.SenderName != "" && !strings.Contains(m.SenderName, "@") {
					sub = m.SenderName + " • " + sub
				}
				row.SetSubtitle(sub)

				openBtn := gtk.NewButtonFromIconName("document-open-symbolic")
				openBtn.AddCSSClass("flat")
				openBtn.SetTooltipText("Open")
				openBtn.SetVAlign(gtk.AlignCenter)
				mCopy := m
				openBtn.ConnectClicked(func() {
					openMedia(mCopy, nil)
				})
				row.AddSuffix(openBtn)
				row.SetActivatable(true)
				row.ConnectActivated(func() {
					openMedia(mCopy, nil)
				})

				c.attachMediaContextMenu(row, mCopy, dialog)
				docGroup.Add(row)
			}
			docsBox.Append(docGroup)
		}
		stack.AddTitled(docsBox, "docs", fmt.Sprintf("Docs (%d)", len(docMsgs)))

		// 3. Audio Tab
		audioBox := gtk.NewBox(gtk.OrientationVertical, 8)
		audioBox.SetMarginTop(8)
		if len(audioMsgs) == 0 {
			emptyGroup := adw.NewPreferencesGroup()
			emptyRow := adw.NewActionRow()
			emptyRow.SetTitle("No audio files")
			emptyRow.SetSubtitle("Voice notes and audio files will appear here")
			emptyRow.AddPrefix(gtk.NewImageFromIconName("folder-music-symbolic"))
			emptyGroup.Add(emptyRow)
			audioBox.Append(emptyGroup)
		} else {
			audioGroup := adw.NewPreferencesGroup()
			for _, m := range audioMsgs {
				row := adw.NewActionRow()
				iconName := "folder-music-symbolic"
				titleText := "Audio Message"
				if m.Kind == messageKindVoice {
					titleText = "Voice Note"
				}
				e2e := parseE2EMessage(m.Raw)
				if e2e != nil && e2e.GetAudioMessage() != nil {
					if s := e2e.GetAudioMessage().GetSeconds(); s > 0 {
						titleText += fmt.Sprintf(" (%ds)", s)
					}
				}
				row.AddPrefix(gtk.NewImageFromIconName(iconName))
				row.SetTitle(titleText)
				sub := m.Timestamp.Format("02 Jan 15:04")
				if m.FromMe {
					sub = "You • " + sub
				} else if m.SenderName != "" && !strings.Contains(m.SenderName, "@") {
					sub = m.SenderName + " • " + sub
				}
				row.SetSubtitle(sub)

				openBtn := gtk.NewButtonFromIconName("media-playback-start-symbolic")
				openBtn.AddCSSClass("flat")
				openBtn.SetTooltipText("Play")
				openBtn.SetVAlign(gtk.AlignCenter)
				mCopy := m
				openBtn.ConnectClicked(func() {
					openMedia(mCopy, nil)
				})
				row.AddSuffix(openBtn)
				row.SetActivatable(true)
				row.ConnectActivated(func() {
					openMedia(mCopy, nil)
				})

				c.attachMediaContextMenu(row, mCopy, dialog)
				audioGroup.Add(row)
			}
			audioBox.Append(audioGroup)
		}
		stack.AddTitled(audioBox, "audio", fmt.Sprintf("Audio (%d)", len(audioMsgs)))

		// 4. Starred Tab
		starredBox := gtk.NewBox(gtk.OrientationVertical, 8)
		starredBox.SetMarginTop(8)
		if len(starredMsgs) == 0 {
			emptyGroup := adw.NewPreferencesGroup()
			emptyRow := adw.NewActionRow()
			emptyRow.SetTitle("No starred messages")
			emptyRow.SetSubtitle("Right-click any message to star and bookmark it")
			emptyRow.AddPrefix(gtk.NewImageFromIconName("starred-symbolic"))
			emptyGroup.Add(emptyRow)
			starredBox.Append(emptyGroup)
		} else {
			starredGroup := adw.NewPreferencesGroup()
			for _, m := range starredMsgs {
				row := adw.NewActionRow()
				iconName := "starred-symbolic"
				titleText := m.Text
				if titleText == "" {
					switch m.Kind {
					case messageKindImage:
						titleText = "📷 Photo"
					case messageKindVideo:
						titleText = "📹 Video"
					case messageKindAudio, messageKindVoice:
						titleText = "🎵 Voice Note"
					case messageKindDocument:
						titleText = "📄 Document"
					default:
						titleText = "💬 Message"
					}
				}
				row.AddPrefix(gtk.NewImageFromIconName(iconName))
				row.SetTitle(titleText)
				sub := m.Timestamp.Format("02 Jan 15:04")
				if m.FromMe {
					sub = "You • " + sub
				} else if m.SenderName != "" && !strings.Contains(m.SenderName, "@") {
					sub = m.SenderName + " • " + sub
				}
				row.SetSubtitle(sub)

				jumpBtn := gtk.NewButtonFromIconName("go-jump-symbolic")
				jumpBtn.AddCSSClass("flat")
				jumpBtn.SetTooltipText("Go to Message")
				jumpBtn.SetVAlign(gtk.AlignCenter)
				mCopy := m
				jumpBtn.ConnectClicked(func() {
					dialog.Close()
					c.jumpToMessage(mCopy.ChatJID, mCopy.ID)
				})
				row.AddSuffix(jumpBtn)
				row.SetActivatable(true)
				row.ConnectActivated(func() {
					dialog.Close()
					c.jumpToMessage(mCopy.ChatJID, mCopy.ID)
				})

				c.attachMediaContextMenu(row, mCopy, dialog)
				starredGroup.Add(row)
			}
			starredBox.Append(starredGroup)
		}
		stack.AddTitled(starredBox, "starred", fmt.Sprintf("Starred (%d)", len(starredMsgs)))

		// Switcher
		switcher := gtk.NewStackSwitcher()
		switcher.SetStack(stack)
		switcher.SetHAlign(gtk.AlignCenter)
		switcher.AddCSSClass("details-tabs-switcher")
		contentBox.Append(switcher)
		contentBox.Append(stack)
	}

	scrolled.SetChild(contentBox)
	toolbarView.SetContent(scrolled)
	dialog.SetChild(toolbarView)
	dialog.Present(c.window)
}

// attachMediaContextMenu adds a right-click popover menu to a media tile, document, audio, or starred row.
func (c *whatsDownMainController) attachMediaContextMenu(widget gtk.Widgetter, msg StoredMessage, dialog *adw.Dialog) {
	w := gtk.BaseWidget(widget)
	popover := gtk.NewPopover()
	popover.SetParent(w)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	mCopy := msg

	// 1. Copy Media / Copy Text
	copyBtn := gtk.NewButton()
	copyBtn.AddCSSClass("flat")
	copyBtn.AddCSSClass("popover-menu-btn")
	cBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	cBox.Append(gtk.NewImageFromIconName("edit-copy-symbolic"))
	cLabel := "Copy Media"
	if mCopy.Kind == messageKindText {
		cLabel = "Copy Text"
	}
	cBox.Append(gtk.NewLabel(cLabel))
	copyBtn.SetChild(cBox)
	copyBtn.ConnectClicked(func() {
		popover.Popdown()
		if mCopy.Kind == messageKindText {
			clipboard := gdk.DisplayGetDefault().Clipboard()
			clipboard.SetText(mCopy.Text)
		} else {
			copyMediaToClipboard(mCopy)
		}
	})
	menuBox.Append(copyBtn)

	// 2. Show in File Manager (for media/docs/audio)
	if mCopy.Kind != messageKindText && mCopy.Kind != messageKindDeleted {
		showBtn := gtk.NewButton()
		showBtn.AddCSSClass("flat")
		showBtn.AddCSSClass("popover-menu-btn")
		sBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		sBox.Append(gtk.NewImageFromIconName("folder-open-symbolic"))
		sBox.Append(gtk.NewLabel("Show in File Manager"))
		showBtn.SetChild(sBox)
		showBtn.ConnectClicked(func() {
			popover.Popdown()
			showMediaInFileManager(mCopy)
		})
		menuBox.Append(showBtn)
	}

	// 3. Go to Message
	goBtn := gtk.NewButton()
	goBtn.AddCSSClass("flat")
	goBtn.AddCSSClass("popover-menu-btn")
	gBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	gBox.Append(gtk.NewImageFromIconName("go-jump-symbolic"))
	gBox.Append(gtk.NewLabel("Go to Message"))
	goBtn.SetChild(gBox)
	goBtn.ConnectClicked(func() {
		popover.Popdown()
		if dialog != nil {
			dialog.Close()
		}
		c.jumpToMessage(mCopy.ChatJID, mCopy.ID)
	})
	menuBox.Append(goBtn)

	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	w.AddController(gesture)
}

// jumpToMessage selects the conversation and scrolls/focuses the target message bubble.
func (c *whatsDownMainController) jumpToMessage(chatJID string, msgID string) {
	if c.currentChat == nil || c.currentChat.JID != chatJID {
		for _, ch := range c.chats {
			if ch.JID == chatJID {
				c.selectChat(ch)
				break
			}
		}
		for _, ch := range c.archivedChats {
			if ch.JID == chatJID {
				c.selectChat(ch)
				break
			}
		}
	}

	foundIdx := -1
	for i, m := range c.currentMessages {
		if m.ID == msgID {
			foundIdx = i
			break
		}
	}

	if foundIdx >= 0 {
		row := c.messagesList.RowAtIndex(foundIdx)
		if row != nil {
			row.GrabFocus()
			row.AddCSSClass("message-highlighted")
			glib.TimeoutAdd(2000, func() bool {
				row.RemoveCSSClass("message-highlighted")
				return false
			})
		}
	}
}

// selectChat sets the active conversation, displays previous messages, and
// clears unread counts.
func (c *whatsDownMainController) selectChat(chat Chat) {
	c.currentChat = &chat

	displayName := formatDisplayName(chat)
	subtitle := formatTopbarSubtitle(chat)

	c.activeChatBtn.SetSensitive(true)
	c.activeChatTitle.SetTitle(displayName)
	c.activeChatTitle.SetSubtitle(subtitle)
	loadAvatar(chat.JID, displayName, c.activeChatAvatar)
	c.contentStack.SetVisibleChildName("chat")

	// Reset conversation search
	c.chatSearchBtn.SetSensitive(true)
	c.chatSearchBtn.SetActive(false)
	c.chatSearchBar.SetSearchMode(false)
	c.chatSearchEntry.SetText("")
	c.messagesSearchFilter = ""
	c.searchMatchIndices = nil
	c.currentSearchMatchIndex = 0
	c.updateSearchUI()

	// On narrow/mobile screens, show the conversation content
	c.splitView.SetShowContent(true)

	// Clear active reply and voice recording when switching chats
	c.cancelReply()
	c.deleteVoiceRecording()

	// Clear unread count locally, dispatch read receipts to WhatsApp, and sync AppState across devices
	c.markChatRead(chat.JID)
	glib.IdleAdd(func() {
		c.loadChats()
		c.loadArchivedChats()
	})

	c.loadMessages(chat.JID)
}

// markChatRead marks all messages in the specified chat as read locally,
// sends read receipts to WhatsApp for incoming messages, and syncs app state to paired devices.
func (c *whatsDownMainController) markChatRead(chatJIDStr string) {
	if chatJIDStr == "" {
		return
	}
	targetJID, err := types.ParseJID(chatJIDStr)
	if err != nil {
		return
	}

	if msgStore != nil {
		_ = msgStore.markChatRead(context.Background(), targetJID)
	}

	for i := range c.chats {
		if c.chats[i].JID == chatJIDStr {
			c.chats[i].UnreadCount = 0
			break
		}
	}
	for i := range c.archivedChats {
		if c.archivedChats[i].JID == chatJIDStr {
			c.archivedChats[i].UnreadCount = 0
			break
		}
	}

	if waClient != nil {
		go func() {
			ctx := context.Background()
			_ = waClient.SendAppState(ctx, appstate.BuildMarkChatAsRead(targetJID, true, time.Now(), nil))

			if msgStore != nil {
				msgs, err := msgStore.listMessages(ctx, chatJIDStr, time.Time{}, 50)
				if err == nil {
					senderMap := make(map[types.JID][]types.MessageID)
					now := time.Now()
					for _, m := range msgs {
						if m.FromMe {
							continue
						}
						sJID, err := types.ParseJID(m.SenderJID)
						if err != nil {
							sJID = targetJID
						}
						senderMap[sJID] = append(senderMap[sJID], types.MessageID(m.ID))
					}
					for sJID, ids := range senderMap {
						if len(ids) > 0 {
							_ = waClient.MarkRead(ctx, ids, now, targetJID, sJID)
						}
					}
				}
			}
		}()
	}
}

// loadMessages fetches recent messages for chatJID and appends bubbles to the
// message list.
func (c *whatsDownMainController) loadMessages(chatJID string) {
	c.messagesList.RemoveAll()
	c.messageRowMap = make(map[string]*gtk.ListBoxRow)
	c.messageStatusLabels = make(map[string]*gtk.Label)

	if msgStore == nil {
		return
	}

	msgs, err := msgStore.listMessages(context.Background(), chatJID, time.Time{}, 100)
	if err != nil {
		log.Printf("loading messages for %s: %v", chatJID, err)
		return
	}

	c.currentMessages = msgs
	c.messageRowMap = make(map[string]*gtk.ListBoxRow)
	for _, msg := range msgs {
		bubble := c.createMessageBubble(msg)
		c.messageRowMap[msg.ID] = bubble
		c.messagesList.Append(bubble)
	}
	c.messagesList.InvalidateFilter()

	// Start at the bottom of the conversation as newest messages are at the bottom
	c.scrollToBottomAnimated()
}

// scrollToBottom smoothly positions the viewport at the newest messages.
func (c *whatsDownMainController) scrollToBottom() {
	if c.messagesScrolledWindow == nil {
		return
	}
	adj := c.messagesScrolledWindow.VAdjustment()
	if adj != nil {
		maxVal := adj.Upper() - adj.PageSize()
		if maxVal > 0 {
			adj.SetValue(maxVal)
		} else {
			adj.SetValue(adj.Upper())
		}
	}
	if ch := c.messagesScrolledWindow.Child(); ch != nil {
		if vp, ok := ch.(*gtk.Viewport); ok && c.messagesList != nil {
			if last := c.messagesList.LastChild(); last != nil {
				vp.ScrollTo(last, nil)
			}
		}
	}
	if c.scrollToBottomBtn != nil {
		c.scrollToBottomBtn.SetVisible(false)
	}
}

// scrollToBottomAnimated ensures the viewport reaches the bottom reliably across async layout passes.
func (c *whatsDownMainController) scrollToBottomAnimated() {
	c.autoScrollBottom = true
	c.scrollToBottom()
	glib.IdleAdd(func() {
		c.scrollToBottom()
	})
	glib.TimeoutAdd(40, func() bool {
		c.scrollToBottom()
		return false
	})
	glib.TimeoutAdd(120, func() bool {
		c.scrollToBottom()
		return false
	})
	glib.TimeoutAdd(300, func() bool {
		c.scrollToBottom()
		return false
	})
}

// performMessageSearch searches through messages in current chat and navigates to matching bubbles.
func (c *whatsDownMainController) performMessageSearch(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	c.messagesSearchFilter = query
	c.searchMatchIndices = nil
	c.currentSearchMatchIndex = 0

	if query == "" {
		c.updateSearchUI()
		return
	}

	for i, m := range c.currentMessages {
		if strings.Contains(strings.ToLower(m.Text), query) || strings.Contains(strings.ToLower(m.SenderName), query) {
			c.searchMatchIndices = append(c.searchMatchIndices, i)
		}
	}

	if len(c.searchMatchIndices) > 0 {
		// Jump to latest match (near the bottom) by default
		c.currentSearchMatchIndex = len(c.searchMatchIndices) - 1
		c.jumpToSearchMatch(c.currentSearchMatchIndex)
	}

	c.updateSearchUI()
}

func (c *whatsDownMainController) navigateSearchMatch(delta int) {
	if len(c.searchMatchIndices) == 0 {
		return
	}
	n := len(c.searchMatchIndices)
	c.currentSearchMatchIndex = (c.currentSearchMatchIndex + delta + n) % n
	c.jumpToSearchMatch(c.currentSearchMatchIndex)
	c.updateSearchUI()
}

func (c *whatsDownMainController) updateSearchUI() {
	if c.chatSearchCountLabel == nil {
		return
	}
	if c.messagesSearchFilter == "" {
		c.chatSearchCountLabel.SetText("")
		if c.chatSearchPrevBtn != nil {
			c.chatSearchPrevBtn.SetSensitive(false)
		}
		if c.chatSearchNextBtn != nil {
			c.chatSearchNextBtn.SetSensitive(false)
		}
		return
	}

	if len(c.searchMatchIndices) == 0 {
		c.chatSearchCountLabel.SetText("0 of 0")
		if c.chatSearchPrevBtn != nil {
			c.chatSearchPrevBtn.SetSensitive(false)
		}
		if c.chatSearchNextBtn != nil {
			c.chatSearchNextBtn.SetSensitive(false)
		}
		return
	}

	c.chatSearchCountLabel.SetText(fmt.Sprintf("%d of %d", c.currentSearchMatchIndex+1, len(c.searchMatchIndices)))
	if c.chatSearchPrevBtn != nil {
		c.chatSearchPrevBtn.SetSensitive(true)
	}
	if c.chatSearchNextBtn != nil {
		c.chatSearchNextBtn.SetSensitive(true)
	}
}

func (c *whatsDownMainController) jumpToSearchMatch(index int) {
	if index < 0 || index >= len(c.searchMatchIndices) {
		return
	}
	msgIdx := c.searchMatchIndices[index]
	row := c.messagesList.RowAtIndex(msgIdx)
	if row != nil {
		row.AddCSSClass("message-highlighted")
		glib.TimeoutAdd(2000, func() bool {
			row.RemoveCSSClass("message-highlighted")
			return false
		})

		// Scroll to row without stealing focus from the search entry
		if c.messagesScrolledWindow != nil {
			if _, destY, ok := row.TranslateCoordinates(c.messagesList, 0, 0); ok {
				if adj := c.messagesScrolledWindow.VAdjustment(); adj != nil {
					target := destY - adj.PageSize()/3
					if target < 0 {
						target = 0
					}
					maxVal := adj.Upper() - adj.PageSize()
					if maxVal > 0 && target > maxVal {
						target = maxVal
					}
					adj.SetValue(target)
				}
			}
		}
	}
}

// senderColorPalette contains distinct, high-contrast colors suitable for light mode.
var senderColorPalette = []string{
	"#E542A3", // Magenta / Pink
	"#E55A5A", // Coral Red
	"#F0821E", // Orange
	"#D97706", // Amber
	"#16A34A", // Forest Green
	"#0D9488", // Teal
	"#0284C7", // Sky Blue
	"#4F46E5", // Indigo
	"#7C3AED", // Violet
	"#C026D3", // Fuchsia
	"#059669", // Emerald
	"#2563EB", // Royal Blue
	"#9333EA", // Purple
	"#EA580C", // Deep Orange
	"#0891B2", // Cyan
	"#475569", // Slate
}

// senderColorPaletteDark has each corresponding tone calibrated brighter for dark mode.
var senderColorPaletteDark = []string{
	"#F472B6", // Bright Pink
	"#F87171", // Bright Coral Red
	"#FB923C", // Bright Orange
	"#FBBF24", // Bright Amber
	"#4ADE80", // Bright Forest Green
	"#2DD4BF", // Bright Teal
	"#38BDF8", // Bright Sky Blue
	"#818CF8", // Bright Indigo
	"#A78BFA", // Bright Violet
	"#E879F9", // Bright Fuchsia
	"#34D399", // Bright Emerald
	"#60A5FA", // Bright Royal Blue
	"#C084FC", // Bright Purple
	"#FB923C", // Bright Deep Orange
	"#22D3EE", // Bright Cyan
	"#94A3B8", // Bright Slate
}

func isDarkMode() bool {
	if !gtk.IsInitialized() {
		return false
	}
	sm := adw.StyleManagerGetDefault()
	if sm != nil {
		return sm.Dark()
	}
	return false
}

// getSenderColor returns a deterministic color from senderColorPalette using seed.
// In dark mode, it returns a tone brighter color; in light mode, it keeps the tone as is.
func getSenderColor(seed string) string {
	if seed == "" {
		seed = "member"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	palette := senderColorPalette
	if isDarkMode() {
		palette = senderColorPaletteDark
	}
	idx := h.Sum32() % uint32(len(palette))
	return palette[idx]
}

var (
	mentionRegex     = regexp.MustCompile(`@([a-zA-Z0-9_.-]+)`)
	contactNameMu    sync.RWMutex
	contactNameCache = make(map[string]string)
)

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (c *whatsDownMainController) resolveContactName(targetJID string) string {
	if targetJID == "" {
		return ""
	}
	contactNameMu.RLock()
	if name, ok := contactNameCache[targetJID]; ok {
		contactNameMu.RUnlock()
		return name
	}
	contactNameMu.RUnlock()

	for _, ch := range c.chats {
		if ch.JID == targetJID {
			disp := formatDisplayName(ch)
			contactNameMu.Lock()
			contactNameCache[targetJID] = disp
			contactNameMu.Unlock()
			return disp
		}
	}
	for _, ch := range c.archivedChats {
		if ch.JID == targetJID {
			disp := formatDisplayName(ch)
			contactNameMu.Lock()
			contactNameCache[targetJID] = disp
			contactNameMu.Unlock()
			return disp
		}
	}

	if msgStore != nil {
		chat, err := msgStore.getChatOrContact(context.Background(), targetJID)
		if err == nil {
			disp := formatDisplayName(chat)
			contactNameMu.Lock()
			contactNameCache[targetJID] = disp
			contactNameMu.Unlock()
			return disp
		}
	}

	return cleanPhoneNumber(targetJID)
}

func (c *whatsDownMainController) openUserDetails(targetJID string) {
	if targetJID == "" {
		return
	}
	if !strings.Contains(targetJID, "@") && isAllDigits(targetJID) && len(targetJID) >= 5 {
		targetJID = targetJID + "@s.whatsapp.net"
	}

	for _, ch := range c.chats {
		if ch.JID == targetJID {
			c.showDetailsAndMedia(ch)
			return
		}
	}
	for _, ch := range c.archivedChats {
		if ch.JID == targetJID {
			c.showDetailsAndMedia(ch)
			return
		}
	}

	if msgStore != nil {
		chat, err := msgStore.getChatOrContact(context.Background(), targetJID)
		if err == nil {
			c.showDetailsAndMedia(chat)
			return
		}
	}

	c.showDetailsAndMedia(Chat{
		JID:  targetJID,
		Name: cleanPhoneNumber(targetJID),
	})
}

// formatMessageTextWithMentions parses @mentions and WhatsApp markdown in message text
// and formats them into Pango markup.
func formatMessageTextWithMentions(text string, mentionedJIDs []string, resolveName func(jid string) string) (string, bool) {
	return FormatWhatsAppMarkdownForBubble(text, mentionedJIDs, resolveName)
}

// extractPollDetails extracts question, options, and selectable count from a raw poll protobuf.
func extractPollDetails(raw []byte) (question string, options []string, selectableCount uint32) {
	return extractPollDetailsFromMsg(parseE2EMessage(raw))
}

// extractPollDetailsFromMsg extracts question, options, and selectable count from a waE2E.Message.
func extractPollDetailsFromMsg(e2e *waE2E.Message) (question string, options []string, selectableCount uint32) {
	if e2e == nil {
		return "Poll", nil, 1
	}
	var poll *waE2E.PollCreationMessage
	for _, p := range []*waE2E.PollCreationMessage{
		e2e.GetPollCreationMessage(), e2e.GetPollCreationMessageV2(),
		e2e.GetPollCreationMessageV3(), e2e.GetPollCreationMessageV5(),
		e2e.GetPollCreationMessageV6(),
	} {
		if p != nil {
			poll = p
			break
		}
	}
	if poll == nil {
		return "Poll", nil, 1
	}
	question = poll.GetName()
	if question == "" {
		question = "Poll"
	}
	selectableCount = poll.GetSelectableOptionsCount()
	for _, opt := range poll.GetOptions() {
		options = append(options, opt.GetOptionName())
	}
	return question, options, selectableCount
}


// getMaxImageWidth calculates the maximum width for an image message: half of a normal message width.
func (c *whatsDownMainController) getMaxImageWidth() int {
	listWidth := 0
	if c.messagesList != nil {
		listWidth = c.messagesList.AllocatedWidth()
	}
	if listWidth <= 0 && c.window != nil {
		listWidth = c.window.AllocatedWidth() - 388
	}
	if listWidth <= 0 {
		listWidth = 560
	}
	// Normal message bubble width minus padding and bubble margins
	normalWidth := listWidth - 100
	if normalWidth < 280 {
		normalWidth = 280
	}
	// Image messages get as wide as half of the normal message
	maxImageWidth := normalWidth / 2
	if maxImageWidth < 180 {
		maxImageWidth = 180
	}
	return maxImageWidth
}

// computeImageDisplaySize calculates proportional target dimensions for an image message.
func computeImageDisplaySize(w, h, maxW int) (int, int) {
	if maxW <= 0 {
		maxW = 240
	}
	if w <= 0 || h <= 0 {
		return maxW, int(float64(maxW) * 0.75)
	}
	scale := float64(maxW) / float64(w)
	targetW := maxW
	targetH := int(math.Round(float64(h) * scale))

	// Limit maximum height so very tall images don't stretch indefinitely
	maxH := int(float64(maxW) * 1.6)
	if targetH > maxH {
		targetH = maxH
		targetW = int(math.Round(float64(w) * (float64(maxH) / float64(h))))
	}
	if targetW < 100 {
		targetW = 100
	}
	if targetH < 100 {
		targetH = 100
	}
	return targetW, targetH
}

// createMessageBubble constructs a GTK widget representing a message bubble.
func (c *whatsDownMainController) createMessageBubble(msg StoredMessage) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.SetSelectable(false)
	row.SetActivatable(false)
	row.AddCSSClass("message-row")

	box := gtk.NewBox(gtk.OrientationVertical, 4)
	box.AddCSSClass("chat-bubble")

	isOutgoing := msg.FromMe || c.isSelfJID(msg.ChatJID)
	if isOutgoing {
		box.AddCSSClass("chat-bubble-outgoing")
		box.SetHAlign(gtk.AlignEnd)
	} else {
		box.AddCSSClass("chat-bubble-incoming")
		box.SetHAlign(gtk.AlignStart)

		// Do not show the username on top of the message for DMs.
		// In groups, show the username in a consistent, distinct color.
		isGroup := strings.HasSuffix(msg.ChatJID, "@g.us") || (c.currentChat != nil && strings.HasSuffix(c.currentChat.JID, "@g.us"))
		if isGroup {
			sender := msg.SenderName
			if sender == "" || strings.Contains(sender, "@") || strings.HasPrefix(sender, "+") || sender == "WhatsApp User" {
				if msg.SenderJID != "" {
					resolved := c.resolveContactName(msg.SenderJID)
					if resolved != "" && !strings.Contains(resolved, "@") && !strings.HasPrefix(resolved, "+") && resolved != "WhatsApp User" {
						sender = resolved
					} else if sender == "" || strings.Contains(sender, "@") {
						if resolved != "" && !strings.Contains(resolved, "@") {
							sender = resolved
						} else {
							cleanPhone := cleanPhoneNumber(msg.SenderJID)
							if cleanPhone != "" && !strings.Contains(cleanPhone, "@") {
								sender = cleanPhone
							}
						}
					}
				}
			}
			if sender == "" {
				sender = "Member"
			}

			// For unsaved numbers/users, put a ~ before their username/number on a message in a group.
			if !msg.SenderSaved && sender != "" && sender != "Member" && !strings.HasPrefix(sender, "~") {
				sender = "~" + sender
			}

			colorSeed := msg.SenderJID
			if colorSeed == "" {
				colorSeed = sender
			}
			color := getSenderColor(colorSeed)

			senderLabel := gtk.NewLabel("")
			senderLabel.SetMarkup(fmt.Sprintf(`<span foreground="%s" weight="bold">%s</span>`, color, glib.MarkupEscapeText(sender)))
			senderLabel.SetXAlign(0)
			senderLabel.AddCSSClass("chat-bubble-sender")

			if msg.SenderJID != "" {
				sJID := msg.SenderJID
				gesture := gtk.NewGestureClick()
				gesture.ConnectPressed(func(nPress int, x, y float64) {
					c.openUserDetails(sJID)
				})
				senderLabel.AddController(gesture)
			}

			box.Append(senderLabel)
		}
	}

	mCopy := msg

	// Render quote block if message replies to another message
	if quoteWidget := c.createQuoteWidget(msg); quoteWidget != nil {
		box.Append(quoteWidget)
	}

	// Render media thumbnail and interactive open buttons
	switch msg.Kind {
	case messageKindImage, messageKindVideo:
		box.AddCSSClass("chat-bubble-media")

		maxImageWidth := c.getMaxImageWidth()
		imgW, imgH := 0, 0
		if waMsg := parseE2EMessage(msg.Raw); waMsg != nil {
			if img := waMsg.GetImageMessage(); img != nil {
				imgW = int(img.GetWidth())
				imgH = int(img.GetHeight())
			} else if vid := waMsg.GetVideoMessage(); vid != nil {
				imgW = int(vid.GetWidth())
				imgH = int(vid.GetHeight())
			}
		}
		targetW, targetH := computeImageDisplaySize(imgW, imgH, maxImageWidth)

		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.AddCSSClass("chat-bubble-picture")
		pic.SetSizeRequest(targetW, targetH)

		loadMediaPreview(msg, pic, func(tex *gdk.Texture) {
			if tex != nil {
				glib.IdleAdd(func() {
					tw, th := computeImageDisplaySize(tex.Width(), tex.Height(), c.getMaxImageWidth())
					pic.SetSizeRequest(tw, th)
					pic.QueueResize()
				})
			}
		})

		gesture := gtk.NewGestureClick()

		overlay := gtk.NewOverlay()
		overlay.SetChild(pic)

		if msg.Kind == messageKindVideo {
			badge := gtk.NewLabel("▶ Video")
			if e2e := parseE2EMessage(msg.Raw); e2e != nil && e2e.GetVideoMessage() != nil {
				if s := e2e.GetVideoMessage().GetSeconds(); s > 0 {
					badge.SetText(fmt.Sprintf("▶ %d:%02d", s/60, s%60))
				}
			}
			badge.AddCSSClass("media-video-badge")
			badge.SetHAlign(gtk.AlignStart)
			badge.SetVAlign(gtk.AlignEnd)
			badge.SetMarginStart(6)
			badge.SetMarginBottom(6)
			overlay.AddOverlay(badge)
		}

		openBtn := gtk.NewButton()
		openBtn.AddCSSClass("osd")
		openBtn.AddCSSClass("pill")
		openBtn.AddCSSClass("media-overlay-btn")
		openBtn.SetHAlign(gtk.AlignCenter)
		openBtn.SetVAlign(gtk.AlignCenter)

		iconName := "insert-image-symbolic"
		btnText := "View Photo"
		if msg.Kind == messageKindVideo {
			iconName = "media-playback-start-symbolic"
			btnText = "Play Video"
		}
		btnInner := gtk.NewBox(gtk.OrientationHorizontal, 6)
		btnInner.Append(gtk.NewImageFromIconName(iconName))
		lbl := gtk.NewLabel(btnText)
		lbl.AddCSSClass("caption")
		btnInner.Append(lbl)
		openBtn.SetChild(btnInner)

		openBtn.ConnectClicked(func() {
			openMedia(mCopy, func(s string) {
				glib.IdleAdd(func() {
					lbl.SetText(s)
				})
			})
		})
		overlay.AddOverlay(openBtn)
		// The overlay gesture would otherwise also fire when the "View Photo" /
		// "Play Video" button is clicked, opening the media twice. Ignore presses
		// that land on the button (it handles those itself) and repeat presses
		// from a double-click.
		gesture.ConnectPressed(func(nPress int, x, y float64) {
			if nPress != 1 {
				return
			}
			if picked := overlay.Pick(x, y, gtk.PickDefault); picked != nil {
				pw := gtk.BaseWidget(picked)
				if pw.Native() == gtk.BaseWidget(openBtn).Native() || pw.IsAncestor(openBtn) {
					return
				}
			}
			openMedia(mCopy, nil)
		})
		overlay.AddController(gesture)
		box.Append(overlay)

	case messageKindAudio, messageKindVoice:
		actionBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
		actionBox.AddCSSClass("chat-bubble-audio-box")

		playBtn := gtk.NewButtonFromIconName("media-playback-start-symbolic")
		playBtn.AddCSSClass("circular")
		playBtn.AddCSSClass("suggested-action")
		playBtn.SetTooltipText("Play Preview")

		infoBox := gtk.NewBox(gtk.OrientationVertical, 2)
		infoBox.SetVAlign(gtk.AlignCenter)

		durTxt := "Voice Message"
		if e2e := parseE2EMessage(msg.Raw); e2e != nil && e2e.GetAudioMessage() != nil {
			if s := e2e.GetAudioMessage().GetSeconds(); s > 0 {
				durTxt = fmt.Sprintf("Voice Note (%d:%02d)", s/60, s%60)
			}
		} else if msg.Kind == messageKindAudio {
			durTxt = "Audio"
		}
		titleLbl := gtk.NewLabel(durTxt)
		titleLbl.SetXAlign(0)
		titleLbl.AddCSSClass("caption")

		progBar := gtk.NewProgressBar()
		progBar.SetFraction(0.0)
		progBar.SetSizeRequest(130, -1)
		progBar.AddCSSClass("osd")

		infoBox.Append(titleLbl)
		infoBox.Append(progBar)

		isPlaying := false
		playBtn.ConnectClicked(func() {
			if isPlaying {
				stopCurrentAudio()
				return
			}
			go func() {
				path, err := downloadMedia(context.Background(), mCopy)
				if err != nil {
					return
				}
				glib.IdleAdd(func() {
					playAudioPreview(path, func(playing bool) {
						isPlaying = playing
						if playing {
							playBtn.SetIconName("media-playback-pause-symbolic")
							progBar.Pulse()
						} else {
							playBtn.SetIconName("media-playback-start-symbolic")
							progBar.SetFraction(0.0)
						}
					})
				})
			}()
		})

		actionBox.Append(playBtn)
		actionBox.Append(infoBox)
		box.Append(actionBox)

	case messageKindDocument:
		docName := "Document"
		docSize := ""
		isPDF := false
		if e2e := parseE2EMessage(msg.Raw); e2e != nil && e2e.GetDocumentMessage() != nil {
			doc := e2e.GetDocumentMessage()
			if doc.GetFileName() != "" {
				docName = doc.GetFileName()
			}
			docSize = formatFileSize(doc.GetFileLength())
			if strings.HasSuffix(strings.ToLower(docName), ".pdf") || strings.Contains(doc.GetMimetype(), "pdf") {
				isPDF = true
			}
		}

		if isPDF {
			overlay := gtk.NewOverlay()
			pic := gtk.NewPicture()
			pic.SetKeepAspectRatio(true)
			pic.SetCanShrink(true)
			pic.AddCSSClass("chat-bubble-picture")
			pic.SetSizeRequest(240, 160)
			loadMediaPreview(msg, pic, nil)
			overlay.SetChild(pic)

			badge := gtk.NewLabel("PDF")
			badge.AddCSSClass("media-video-badge")
			badge.SetHAlign(gtk.AlignStart)
			badge.SetVAlign(gtk.AlignStart)
			badge.SetMarginStart(6)
			badge.SetMarginTop(6)
			overlay.AddOverlay(badge)

			gesture := gtk.NewGestureClick()
			gesture.ConnectPressed(func(nPress int, x, y float64) {
				c.openPDFPreview(mCopy, docName)
			})
			overlay.AddController(gesture)
			box.Append(overlay)
		}

		actionBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
		openBtn := gtk.NewButton()
		openBtn.AddCSSClass("flat")

		btnInner := gtk.NewBox(gtk.OrientationHorizontal, 6)
		btnInner.Append(gtk.NewImageFromIconName("folder-documents-symbolic"))
		labelTxt := docName
		if docSize != "" {
			labelTxt += " (" + docSize + ")"
		}
		lbl := gtk.NewLabel(labelTxt)
		lbl.AddCSSClass("caption")
		lbl.SetEllipsize(pango.EllipsizeEnd)
		lbl.SetMaxWidthChars(28)
		btnInner.Append(lbl)
		openBtn.SetChild(btnInner)

		openBtn.ConnectClicked(func() {
			if isPDF {
				c.openPDFPreview(mCopy, docName)
			} else {
				openMedia(mCopy, nil)
			}
		})
		actionBox.Append(openBtn)
		box.Append(actionBox)

	case messageKindLocation:
		actionBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
		openBtn := gtk.NewButton()
		openBtn.AddCSSClass("flat")
		btnInner := gtk.NewBox(gtk.OrientationHorizontal, 6)
		btnInner.Append(gtk.NewImageFromIconName("find-location-symbolic"))
		lbl := gtk.NewLabel("Open Location in Maps")
		lbl.AddCSSClass("caption")
		btnInner.Append(lbl)
		openBtn.SetChild(btnInner)

		openBtn.ConnectClicked(func() {
			openMedia(mCopy, nil)
		})
		actionBox.Append(openBtn)
		box.Append(actionBox)

	case messageKindPoll:
		pollBox := gtk.NewBox(gtk.OrientationVertical, 6)
		pollBox.AddCSSClass("chat-bubble-poll-box")

		headerBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
		pollIcon := gtk.NewImageFromIconName("view-list-bullet-symbolic")
		headerBox.Append(pollIcon)

		question, options, selectableCount := extractPollDetails(msg.Raw)
		if question == "" || question == "Poll" {
			if msg.Text != "" {
				question = msg.Text
			} else {
				question = "Poll"
			}
		}

		qLbl := gtk.NewLabel(question)
		qLbl.AddCSSClass("heading")
		qLbl.SetWrap(true)
		qLbl.SetWrapMode(pango.WrapWordChar)
		qLbl.SetXAlign(0)
		headerBox.Append(qLbl)
		pollBox.Append(headerBox)

		allowMultiple := selectableCount > 1 || selectableCount == 0
		subText := "Select one"
		if allowMultiple {
			subText = "Select one or more"
		}
		subLbl := gtk.NewLabel(subText)
		subLbl.AddCSSClass("caption")
		subLbl.AddCSSClass("dim-label")
		subLbl.SetXAlign(0)
		pollBox.Append(subLbl)

		voterJID := "me"
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			voterJID = waClient.Store.ID.ToNonAD().String()
		}

		var initialVotes map[string]bool
		var initialResults *PollResults
		if msgStore != nil {
			initialVotes, _ = msgStore.getPollVotes(context.Background(), msg.ChatJID, msg.ID, voterJID)
			initialResults, _ = msgStore.getPollResults(context.Background(), msg.ChatJID, msg.ID)
		}
		if initialVotes == nil {
			initialVotes = make(map[string]bool)
		}

		totalVoters := 0
		if initialResults != nil {
			totalVoters = initialResults.TotalVoters
		}

		optWidgets := make(map[string]*pollOptWidget)
		summaryBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
		summaryBox.AddCSSClass("poll-footer-box")
		summaryBox.SetMarginTop(4)

		totalVotesLbl := gtk.NewLabel("No votes yet")
		totalVotesLbl.AddCSSClass("caption")
		totalVotesLbl.AddCSSClass("dim-label")
		totalVotesLbl.SetHExpand(true)
		totalVotesLbl.SetXAlign(0)
		if totalVoters == 1 {
			totalVotesLbl.SetText("1 person voted")
		} else if totalVoters > 1 {
			totalVotesLbl.SetText(fmt.Sprintf("%d people voted", totalVoters))
		}

		for _, opt := range options {
			optCopy := opt
			optBtn := gtk.NewButton()
			optBtn.AddCSSClass("poll-option-btn")

			optCol := gtk.NewBox(gtk.OrientationVertical, 4)

			rowBox := gtk.NewBox(gtk.OrientationHorizontal, 8)

			isSelected := initialVotes[optCopy]
			var iconName string
			if allowMultiple {
				if isSelected {
					iconName = "checkbox-checked-symbolic"
				} else {
					iconName = "checkbox-symbolic"
				}
			} else {
				if isSelected {
					iconName = "radio-checked-symbolic"
				} else {
					iconName = "radio-symbolic"
				}
			}
			indicator := gtk.NewImageFromIconName(iconName)
			rowBox.Append(indicator)

			optLbl := gtk.NewLabel(optCopy)
			optLbl.SetXAlign(0)
			optLbl.SetHExpand(true)
			optLbl.SetWrap(true)
			rowBox.Append(optLbl)

			cnt := 0
			if initialResults != nil {
				cnt = initialResults.VoteCounts[optCopy]
			}
			var cText string
			if cnt == 1 {
				cText = "1 vote"
			} else if cnt > 1 {
				cText = fmt.Sprintf("%d votes", cnt)
			} else {
				cText = "0"
			}
			countLbl := gtk.NewLabel(cText)
			countLbl.AddCSSClass("poll-vote-count")
			rowBox.Append(countLbl)

			optCol.Append(rowBox)

			pBar := gtk.NewProgressBar()
			pBar.AddCSSClass("poll-progress-bar")
			var frac float64
			if totalVoters > 0 {
				frac = float64(cnt) / float64(totalVoters)
			}
			pBar.SetFraction(frac)
			optCol.Append(pBar)

			optBtn.SetChild(optCol)
			if isSelected {
				optBtn.AddCSSClass("selected")
			}

			optWidgets[optCopy] = &pollOptWidget{
				btn:         optBtn,
				indicator:   indicator,
				countLbl:    countLbl,
				progressBar: pBar,
			}

			optBtn.ConnectClicked(func() {
				c.handlePollVote(msg, optCopy, allowMultiple, optWidgets, totalVotesLbl)
			})

			pollBox.Append(optBtn)
		}

		viewVotesBtn := gtk.NewButtonWithLabel("View votes")
		viewVotesBtn.AddCSSClass("flat")
		viewVotesBtn.AddCSSClass("poll-view-votes-btn")
		viewVotesBtn.AddCSSClass("pill")
		viewVotesBtn.ConnectClicked(func() {
			c.showPollVotesDialog(msg, question, options)
		})

		summaryBox.Append(totalVotesLbl)
		summaryBox.Append(viewVotesBtn)
		pollBox.Append(summaryBox)

		box.Append(pollBox)

	case messageKindSticker:
		box.AddCSSClass("chat-bubble-sticker")

		stickerW := 140
		stickerH := 140
		if waMsg := parseE2EMessage(msg.Raw); waMsg != nil && waMsg.GetStickerMessage() != nil {
			w := int(waMsg.GetStickerMessage().GetWidth())
			h := int(waMsg.GetStickerMessage().GetHeight())
			if w > 0 && h > 0 {
				if w >= h {
					stickerH = int(float64(h) * 140.0 / float64(w))
					stickerW = 140
				} else {
					stickerW = int(float64(w) * 140.0 / float64(h))
					stickerH = 140
				}
				if stickerW < 60 {
					stickerW = 60
				}
				if stickerH < 60 {
					stickerH = 60
				}
			}
		}

		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.AddCSSClass("chat-bubble-picture")
		pic.AddCSSClass("chat-bubble-sticker-picture")
		pic.SetSizeRequest(stickerW, stickerH)
		pic.SetCursorFromName("pointer")

		loadMediaPreview(msg, pic, func(tex *gdk.Texture) {
			if tex != nil {
				glib.IdleAdd(func() {
					tw, th := tex.Width(), tex.Height()
					if tw > 0 && th > 0 {
						sw, sh := 140, 140
						if tw >= th {
							sh = int(float64(th) * 140.0 / float64(tw))
							sw = 140
						} else {
							sw = int(float64(tw) * 140.0 / float64(th))
							sh = 140
						}
						if sw < 60 {
							sw = 60
						}
						if sh < 60 {
							sh = 60
						}
						pic.SetSizeRequest(sw, sh)
						pic.QueueResize()
					}
					c.scrollToBottom()
				})
			}
		})

		c.attachStickerContextMenu(pic, mCopy)

		box.Append(pic)
	}

	displayText := msg.Text
	if displayText != "" && msg.Kind != messageKindPoll {
		mentionedJIDs := extractMentionedJIDs(msg.Raw)
		markup, hasMentions := formatMessageTextWithMentions(displayText, mentionedJIDs, c.resolveContactName)

		textLabel := gtk.NewLabel("")
		textLabel.SetWrap(true)
		textLabel.SetWrapMode(pango.WrapWordChar)
		textLabel.SetXAlign(0)
		textLabel.SetSelectable(true)
		textLabel.AddCSSClass("chat-bubble-text")

		if _, _, _, err := pango.ParseMarkup(markup, 0); err == nil {
			textLabel.SetMarkup(markup)
		} else {
			textLabel.SetText(displayText)
		}

		if hasMentions {
			textLabel.ConnectActivateLink(func(uri string) bool {
				if strings.HasPrefix(uri, "user:") {
					targetJID := strings.TrimPrefix(uri, "user:")
					c.openUserDetails(targetJID)
					return true
				}
				return false
			})
		}
		box.Append(textLabel)
	} else if msg.Kind != messageKindImage && msg.Kind != messageKindVideo && msg.Kind != messageKindAudio && msg.Kind != messageKindVoice && msg.Kind != messageKindDocument && msg.Kind != messageKindLocation && msg.Kind != messageKindSticker {
		switch msg.Kind {
		case messageKindPoll:
			displayText = "📊 Poll"
		case messageKindDeleted:
			displayText = "🚫 This message was deleted"
		default:
			displayText = "💬 Message"
		}
		textLabel := gtk.NewLabel(displayText)
		textLabel.SetWrap(true)
		textLabel.SetWrapMode(pango.WrapWordChar)
		textLabel.SetXAlign(0)
		textLabel.SetSelectable(true)
		textLabel.AddCSSClass("chat-bubble-text")
		box.Append(textLabel)
	}

	footerBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	footerBox.SetHAlign(gtk.AlignEnd)
	footerBox.SetVAlign(gtk.AlignCenter)
	footerBox.AddCSSClass("chat-bubble-footer")
	if msg.Kind == messageKindSticker {
		footerBox.AddCSSClass("chat-bubble-sticker-footer")
	}

	if msg.Starred {
		starIcon := gtk.NewImageFromIconName("starred-symbolic")
		starIcon.AddCSSClass("pin-indicator")
		starIcon.SetVAlign(gtk.AlignCenter)
		footerBox.Append(starIcon)
	}

	timeStr := msg.Timestamp.Format("15:04")
	if msg.Edited {
		timeStr += " (edited)"
	}
	timeLabel := gtk.NewLabel(timeStr)
	timeLabel.SetXAlign(1)
	timeLabel.SetVAlign(gtk.AlignCenter)
	timeLabel.AddCSSClass("chat-bubble-time")
	footerBox.Append(timeLabel)

	if isOutgoing {
		status := msg.Status
		if c.isSelfJID(msg.ChatJID) {
			status = MessageStatusRead
		}
		statusLabel := gtk.NewLabel("")
		statusLabel.SetVAlign(gtk.AlignCenter)
		statusLabel.AddCSSClass("chat-bubble-status")
		c.applyStatusToLabel(statusLabel, status)
		footerBox.Append(statusLabel)
		if c.messageStatusLabels != nil {
			c.messageStatusLabels[msg.ID] = statusLabel
		}
	}

	box.Append(footerBox)

	// Right-click context menu on message bubble
	c.attachMessageContextMenu(box, msg)

	rowBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	rowBox.AddCSSClass("message-row-box")

	if msg.Kind != messageKindDeleted {
		replyBtn := gtk.NewButton()
		replyBtn.AddCSSClass("flat")
		replyBtn.AddCSSClass("circular")
		replyBtn.AddCSSClass("message-reply-btn")
		replyBtn.SetIconName("mail-reply-sender-symbolic")
		replyBtn.SetTooltipText("Reply")
		replyBtn.SetVAlign(gtk.AlignCenter)
		replyBtn.SetCanFocus(false)
		replyBtn.SetOpacity(0.0)

		replyBtn.ConnectClicked(func() {
			if replyBtn.Opacity() < 0.2 {
				return
			}
			c.startReply(mCopy)
		})

		motion := gtk.NewEventControllerMotion()
		motion.ConnectEnter(func(x, y float64) {
			replyBtn.SetOpacity(1.0)
		})
		motion.ConnectLeave(func() {
			replyBtn.SetOpacity(0.0)
		})
		row.AddController(motion)

		if isOutgoing {
			rowBox.SetHAlign(gtk.AlignEnd)
			rowBox.Append(replyBtn)
			rowBox.Append(box)
		} else {
			rowBox.SetHAlign(gtk.AlignStart)
			rowBox.Append(box)
			rowBox.Append(replyBtn)
		}
	} else {
		if isOutgoing {
			rowBox.SetHAlign(gtk.AlignEnd)
		} else {
			rowBox.SetHAlign(gtk.AlignStart)
		}
		rowBox.Append(box)
	}

	row.SetChild(rowBox)
	return row
}

// attachMessageContextMenu adds a right-click popover context menu to a message bubble.
func (c *whatsDownMainController) attachMessageContextMenu(widget gtk.Widgetter, msg StoredMessage) {
	w := gtk.BaseWidget(widget)
	popover := gtk.NewPopover()
	popover.SetParent(w)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	mCopy := msg

	// For stickers, add Save Sticker and Open as Image options at the top of the context menu
	if mCopy.Kind == messageKindSticker {
		isSaved := c.isStickerSaved(mCopy)
		saveBtn := gtk.NewButton()
		saveBtn.AddCSSClass("flat")
		saveBtn.AddCSSClass("popover-menu-btn")
		saveRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
		if isSaved {
			saveRow.Append(gtk.NewImageFromIconName("user-trash-symbolic"))
			saveRow.Append(gtk.NewLabel("Remove from Saved Stickers"))
			saveBtn.SetChild(saveRow)
			saveBtn.ConnectClicked(func() {
				popover.Popdown()
				c.removeSavedStickerByMessage(mCopy)
			})
		} else {
			saveRow.Append(gtk.NewImageFromIconName("star-new-symbolic"))
			saveRow.Append(gtk.NewLabel("Save Sticker"))
			saveBtn.SetChild(saveRow)
			saveBtn.ConnectClicked(func() {
				popover.Popdown()
				c.saveStickerFromMessage(mCopy)
			})
		}
		menuBox.Append(saveBtn)

		openBtn := gtk.NewButton()
		openBtn.AddCSSClass("flat")
		openBtn.AddCSSClass("popover-menu-btn")
		openRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
		openRow.Append(gtk.NewImageFromIconName("insert-image-symbolic"))
		openRow.Append(gtk.NewLabel("Open as Image"))
		openBtn.SetChild(openRow)
		openBtn.ConnectClicked(func() {
			popover.Popdown()
			openMedia(mCopy, nil)
		})
		menuBox.Append(openBtn)
	}

	// 0. Reply
	if mCopy.Kind != messageKindDeleted {
		replyBtn := gtk.NewButton()
		replyBtn.AddCSSClass("flat")
		replyBtn.AddCSSClass("popover-menu-btn")
		rBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		rBox.Append(gtk.NewImageFromIconName("mail-reply-sender-symbolic"))
		rBox.Append(gtk.NewLabel("Reply"))
		replyBtn.SetChild(rBox)
		replyBtn.ConnectClicked(func() {
			popover.Popdown()
			c.startReply(mCopy)
		})
		menuBox.Append(replyBtn)
	}

	// Edit Message (only for text messages sent by user)
	if mCopy.FromMe && mCopy.Kind == messageKindText {
		editBtn := gtk.NewButton()
		editBtn.AddCSSClass("flat")
		editBtn.AddCSSClass("popover-menu-btn")
		eBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		eBox.Append(gtk.NewImageFromIconName("document-edit-symbolic"))
		eBox.Append(gtk.NewLabel("Edit Message"))
		editBtn.SetChild(eBox)
		editBtn.ConnectClicked(func() {
			popover.Popdown()
			c.startEditing(mCopy)
		})
		menuBox.Append(editBtn)
	}

	// View Edit History (ONLY IF the message has been edited)
	if mCopy.Edited {
		historyBtn := gtk.NewButton()
		historyBtn.AddCSSClass("flat")
		historyBtn.AddCSSClass("popover-menu-btn")
		hBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		hBox.Append(gtk.NewImageFromIconName("document-revert-symbolic"))
		hBox.Append(gtk.NewLabel("View Edit History"))
		historyBtn.SetChild(hBox)
		historyBtn.ConnectClicked(func() {
			popover.Popdown()
			c.showEditHistoryDialog(mCopy)
		})
		menuBox.Append(historyBtn)
	}

	// 1. Star / Unstar
	starText := "Star Message"
	starIcon := "starred-symbolic"
	if msg.Starred {
		starText = "Unstar Message"
		starIcon = "non-starred-symbolic"
	}
	starBtn := gtk.NewButton()
	starBtn.AddCSSClass("flat")
	starBtn.AddCSSClass("popover-menu-btn")
	sBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	sBox.Append(gtk.NewImageFromIconName(starIcon))
	sBox.Append(gtk.NewLabel(starText))
	starBtn.SetChild(sBox)
	starBtn.ConnectClicked(func() {
		popover.Popdown()
		c.toggleMessageStarred(mCopy)
	})
	menuBox.Append(starBtn)

	// 2. Copy Text / Copy Media
	copyBtn := gtk.NewButton()
	copyBtn.AddCSSClass("flat")
	copyBtn.AddCSSClass("popover-menu-btn")
	cBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	cBox.Append(gtk.NewImageFromIconName("edit-copy-symbolic"))
	cLabel := "Copy Text"
	if mCopy.Kind == messageKindImage || mCopy.Kind == messageKindVideo || mCopy.Kind == messageKindDocument || mCopy.Kind == messageKindAudio || mCopy.Kind == messageKindVoice {
		cLabel = "Copy Media"
	}
	cBox.Append(gtk.NewLabel(cLabel))
	copyBtn.SetChild(cBox)
	copyBtn.ConnectClicked(func() {
		popover.Popdown()
		if mCopy.Kind == messageKindText {
			clipboard := gdk.DisplayGetDefault().Clipboard()
			clipboard.SetText(mCopy.Text)
		} else {
			copyMediaToClipboard(mCopy)
		}
	})
	menuBox.Append(copyBtn)

	// 3. Show in File Manager for media
	if mCopy.Kind == messageKindImage || mCopy.Kind == messageKindVideo || mCopy.Kind == messageKindDocument || mCopy.Kind == messageKindAudio || mCopy.Kind == messageKindVoice {
		fmBtn := gtk.NewButton()
		fmBtn.AddCSSClass("flat")
		fmBtn.AddCSSClass("popover-menu-btn")
		fBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		fBox.Append(gtk.NewImageFromIconName("folder-open-symbolic"))
		fBox.Append(gtk.NewLabel("Show in File Manager"))
		fmBtn.SetChild(fBox)
		fmBtn.ConnectClicked(func() {
			popover.Popdown()
			showMediaInFileManager(mCopy)
		})
		menuBox.Append(fmBtn)
	}

	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	w.AddController(gesture)
}

// attachStickerContextMenu configures left and right click actions on sticker bubbles.
func (c *whatsDownMainController) attachStickerContextMenu(widget gtk.Widgetter, msg StoredMessage) {
	w := gtk.BaseWidget(widget)
	popover := gtk.NewPopover()
	popover.SetParent(w)
	popover.SetHasArrow(true)
	popover.AddCSSClass("popover-menu-box")

	mCopy := msg

	populate := func() {
		menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
		menuBox.SetMarginTop(4)
		menuBox.SetMarginBottom(4)
		menuBox.SetMarginStart(4)
		menuBox.SetMarginEnd(4)

		isSaved := c.isStickerSaved(mCopy)

		// 1. Save Sticker / Remove from Saved Stickers
		saveBtn := gtk.NewButton()
		saveBtn.AddCSSClass("flat")
		saveBtn.AddCSSClass("popover-menu-btn")
		saveRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
		if isSaved {
			saveRow.Append(gtk.NewImageFromIconName("user-trash-symbolic"))
			saveRow.Append(gtk.NewLabel("Remove from Saved Stickers"))
			saveBtn.SetChild(saveRow)
			saveBtn.ConnectClicked(func() {
				popover.Popdown()
				c.removeSavedStickerByMessage(mCopy)
			})
		} else {
			saveRow.Append(gtk.NewImageFromIconName("star-new-symbolic"))
			saveRow.Append(gtk.NewLabel("Save Sticker"))
			saveBtn.SetChild(saveRow)
			saveBtn.ConnectClicked(func() {
				popover.Popdown()
				c.saveStickerFromMessage(mCopy)
			})
		}
		menuBox.Append(saveBtn)

		// 2. Open as Image
		openBtn := gtk.NewButton()
		openBtn.AddCSSClass("flat")
		openBtn.AddCSSClass("popover-menu-btn")
		openRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
		openRow.Append(gtk.NewImageFromIconName("insert-image-symbolic"))
		openRow.Append(gtk.NewLabel("Open as Image"))
		openBtn.SetChild(openRow)
		openBtn.ConnectClicked(func() {
			popover.Popdown()
			openMedia(mCopy, nil)
		})
		menuBox.Append(openBtn)

		// 3. Reply
		if mCopy.Kind != messageKindDeleted {
			replyBtn := gtk.NewButton()
			replyBtn.AddCSSClass("flat")
			replyBtn.AddCSSClass("popover-menu-btn")
			rBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
			rBox.Append(gtk.NewImageFromIconName("mail-reply-sender-symbolic"))
			rBox.Append(gtk.NewLabel("Reply"))
			replyBtn.SetChild(rBox)
			replyBtn.ConnectClicked(func() {
				popover.Popdown()
				c.startReply(mCopy)
			})
			menuBox.Append(replyBtn)
		}

		// 4. Star / Unstar
		starText := "Star Message"
		starIcon := "starred-symbolic"
		if mCopy.Starred {
			starText = "Unstar Message"
			starIcon = "non-starred-symbolic"
		}
		starBtn := gtk.NewButton()
		starBtn.AddCSSClass("flat")
		starBtn.AddCSSClass("popover-menu-btn")
		sBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		sBox.Append(gtk.NewImageFromIconName(starIcon))
		sBox.Append(gtk.NewLabel(starText))
		starBtn.SetChild(sBox)
		starBtn.ConnectClicked(func() {
			popover.Popdown()
			c.toggleMessageStarred(mCopy)
		})
		menuBox.Append(starBtn)

		// 5. Copy Sticker
		copyBtn := gtk.NewButton()
		copyBtn.AddCSSClass("flat")
		copyBtn.AddCSSClass("popover-menu-btn")
		cBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		cBox.Append(gtk.NewImageFromIconName("edit-copy-symbolic"))
		cBox.Append(gtk.NewLabel("Copy Sticker"))
		copyBtn.SetChild(cBox)
		copyBtn.ConnectClicked(func() {
			popover.Popdown()
			copyMediaToClipboard(mCopy)
		})
		menuBox.Append(copyBtn)

		// 6. Show in File Manager
		fmBtn := gtk.NewButton()
		fmBtn.AddCSSClass("flat")
		fmBtn.AddCSSClass("popover-menu-btn")
		fBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		fBox.Append(gtk.NewImageFromIconName("folder-open-symbolic"))
		fBox.Append(gtk.NewLabel("Show in File Manager"))
		fmBtn.SetChild(fBox)
		fmBtn.ConnectClicked(func() {
			popover.Popdown()
			showMediaInFileManager(mCopy)
		})
		menuBox.Append(fmBtn)

		popover.SetChild(menuBox)
	}

	gesture := gtk.NewGestureClick()
	gesture.SetButton(0)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		populate()
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	w.AddController(gesture)
}

// isStickerSaved checks if a sticker message has already been saved to the user's collection.
func (c *whatsDownMainController) isStickerSaved(msg StoredMessage) bool {
	if c.savedStickerHashes != nil {
		if c.savedStickerHashes[msg.ID] {
			return true
		}
		localPath := getLocalMediaPath(msg)
		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
			if data, err := os.ReadFile(localPath); err == nil {
				hash := fmt.Sprintf("%x", sha256.Sum256(data))
				if c.savedStickerHashes[hash] {
					return true
				}
			}
		}
	}
	return false
}

// saveStickerFromMessage saves a sticker from a chat message into the user's personal sticker collection.
func (c *whatsDownMainController) saveStickerFromMessage(msg StoredMessage) {
	go func() {
		localPath, err := downloadMedia(context.Background(), msg)
		if err != nil {
			log.Printf("saving sticker: download failed: %v", err)
			return
		}

		data, err := os.ReadFile(localPath)
		if err != nil {
			log.Printf("saving sticker: reading file failed: %v", err)
			return
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		destDir := savedStickersDir()
		destFile := filepath.Join(destDir, hash+".webp")
		_ = os.WriteFile(destFile, data, 0o600)

		sticker := SavedSticker{
			ID:       hash,
			FilePath: destFile,
			FileHash: hash,
			Width:    512,
			Height:   512,
			SavedAt:  time.Now(),
		}
		if waMsg := parseE2EMessage(msg.Raw); waMsg != nil && waMsg.GetStickerMessage() != nil {
			if w := int(waMsg.GetStickerMessage().GetWidth()); w > 0 {
				sticker.Width = w
			}
			if h := int(waMsg.GetStickerMessage().GetHeight()); h > 0 {
				sticker.Height = h
			}
		}

		if msgStore != nil {
			_ = msgStore.saveSticker(context.Background(), sticker)
		}

		glib.IdleAdd(func() {
			c.loadSavedStickersUI()
		})
	}()
}

// removeSavedStickerByMessage deletes a sticker associated with a message from saved stickers.
func (c *whatsDownMainController) removeSavedStickerByMessage(msg StoredMessage) {
	localPath := getLocalMediaPath(msg)
	var hash string
	if data, err := os.ReadFile(localPath); err == nil {
		hash = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	if hash != "" {
		c.removeSavedSticker(hash)
	}
	c.removeSavedSticker(msg.ID)
}

// removeSavedSticker deletes a saved sticker by ID or hash.
func (c *whatsDownMainController) removeSavedSticker(id string) {
	if msgStore != nil {
		_ = msgStore.deleteSavedSticker(context.Background(), id)
	}
	glib.IdleAdd(func() {
		c.loadSavedStickersUI()
	})
}

// loadSavedStickersUI populates the sticker popover flowbox with the user's saved stickers.
func (c *whatsDownMainController) loadSavedStickersUI() {
	if msgStore == nil || c.stickerFlowbox == nil {
		return
	}

	stickers, err := msgStore.listSavedStickers(context.Background())
	if err != nil {
		log.Printf("listing saved stickers: %v", err)
		return
	}

	c.savedStickers = stickers
	c.savedStickerHashes = make(map[string]bool)
	for _, s := range stickers {
		c.savedStickerHashes[s.ID] = true
		if s.FileHash != "" {
			c.savedStickerHashes[s.FileHash] = true
		}
	}

	for {
		ch := c.stickerFlowbox.FirstChild()
		if ch == nil {
			break
		}
		c.stickerFlowbox.Remove(ch)
	}

	if len(stickers) == 0 {
		if c.stickerEmptyBox != nil {
			c.stickerEmptyBox.SetVisible(true)
		}
		if c.stickerScrolledWindow != nil {
			c.stickerScrolledWindow.SetVisible(false)
		}
		return
	}

	if c.stickerEmptyBox != nil {
		c.stickerEmptyBox.SetVisible(false)
	}
	if c.stickerScrolledWindow != nil {
		c.stickerScrolledWindow.SetVisible(true)
	}

	for _, s := range stickers {
		stk := s
		btn := gtk.NewButton()
		btn.AddCSSClass("flat")
		btn.AddCSSClass("sticker-picker-item")
		btn.SetCursorFromName("pointer")
		btn.SetTooltipText("Click to send")

		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.SetSizeRequest(64, 64)

		if fi, err := os.Stat(stk.FilePath); err == nil && fi.Size() > 0 {
			if tex, err := gdk.NewTextureFromFilename(stk.FilePath); err == nil && tex != nil {
				pic.SetPaintable(tex)
			}
		}

		btn.SetChild(pic)

		// Click to send
		btn.ConnectClicked(func() {
			if c.stickerPopover != nil {
				c.stickerPopover.Popdown()
			}
			c.sendSticker(stk.FilePath)
		})

		// Right click on sticker in picker
		itemPopover := gtk.NewPopover()
		itemPopover.SetParent(btn)
		itemPopover.SetHasArrow(true)
		itemPopover.AddCSSClass("popover-menu-box")

		popBox := gtk.NewBox(gtk.OrientationVertical, 2)
		popBox.SetMarginTop(4)
		popBox.SetMarginBottom(4)
		popBox.SetMarginStart(4)
		popBox.SetMarginEnd(4)

		sendItem := gtk.NewButton()
		sendItem.AddCSSClass("flat")
		sendItem.AddCSSClass("popover-menu-btn")
		sRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
		sRow.Append(gtk.NewImageFromIconName("mail-send-symbolic"))
		sRow.Append(gtk.NewLabel("Send Sticker"))
		sendItem.SetChild(sRow)
		sendItem.ConnectClicked(func() {
			itemPopover.Popdown()
			if c.stickerPopover != nil {
				c.stickerPopover.Popdown()
			}
			c.sendSticker(stk.FilePath)
		})
		popBox.Append(sendItem)

		delItem := gtk.NewButton()
		delItem.AddCSSClass("flat")
		delItem.AddCSSClass("popover-menu-btn")
		dRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
		dRow.Append(gtk.NewImageFromIconName("user-trash-symbolic"))
		dRow.Append(gtk.NewLabel("Remove from Saved"))
		delItem.SetChild(dRow)
		delItem.ConnectClicked(func() {
			itemPopover.Popdown()
			c.removeSavedSticker(stk.ID)
		})
		popBox.Append(delItem)

		itemPopover.SetChild(popBox)

		rGesture := gtk.NewGestureClick()
		rGesture.SetButton(3)
		rGesture.ConnectPressed(func(nPress int, x, y float64) {
			rect := gdk.NewRectangle(int(x), int(y), 1, 1)
			itemPopover.SetPointingTo(&rect)
			itemPopover.Popup()
		})
		btn.AddController(rGesture)

		c.stickerFlowbox.Append(btn)
	}
}

// openFileChooserForSticker allows the user to import an image from disk to save as a sticker.
func (c *whatsDownMainController) openFileChooserForSticker() {
	title := "Choose Image for Sticker"
	dialog := gtk.NewFileChooserNative(title, &c.window.Window, gtk.FileChooserActionOpen, "_Select", "_Cancel")

	filter := gtk.NewFileFilter()
	filter.SetName("Sticker Images (*.webp, *.png, *.jpg, *.jpeg)")
	filter.AddMIMEType("image/webp")
	filter.AddMIMEType("image/png")
	filter.AddMIMEType("image/jpeg")
	filter.AddPattern("*.webp")
	filter.AddPattern("*.WEBP")
	filter.AddPattern("*.png")
	filter.AddPattern("*.PNG")
	filter.AddPattern("*.jpg")
	filter.AddPattern("*.jpeg")
	filter.AddPattern("*.JPG")
	filter.AddPattern("*.JPEG")
	dialog.SetFilter(filter)

	dialog.ConnectResponse(func(responseId int) {
		if responseId == int(gtk.ResponseAccept) {
			if file := dialog.File(); file != nil {
				path := file.Path()
				if path != "" {
					c.addNewStickerFromFile(path)
				}
			}
		}
		dialog.Destroy()
	})
	dialog.Show()
}

// addNewStickerFromFile converts and saves an external image into the user's saved stickers collection.
func (c *whatsDownMainController) addNewStickerFromFile(filePath string) {
	go func() {
		data, err := ensureWebPSticker(filePath)
		if err != nil {
			log.Printf("adding sticker from file %s: %v", filePath, err)
			return
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		destFile := filepath.Join(savedStickersDir(), hash+".webp")
		_ = os.WriteFile(destFile, data, 0o600)

		sticker := SavedSticker{
			ID:       hash,
			FilePath: destFile,
			FileHash: hash,
			Width:    512,
			Height:   512,
			SavedAt:  time.Now(),
		}

		if msgStore != nil {
			_ = msgStore.saveSticker(context.Background(), sticker)
		}

		glib.IdleAdd(func() {
			c.loadSavedStickersUI()
		})
	}()
}

// sendSticker sends a sticker to the currently active conversation.
func (c *whatsDownMainController) sendSticker(filePath string) {
	if c.currentChat == nil {
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("reading sticker file %s: %v", filePath, err)
		return
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	go func() {
		ctx := context.Background()
		var msgID string
		var rawProto []byte

		if waClient != nil {
			targetJID, err := types.ParseJID(chatJIDStr)
			if err != nil {
				log.Printf("parsing JID for sticker send: %v", err)
				return
			}

			uploadResp, err := waClient.Upload(ctx, data, whatsmeow.MediaImage)
			if err != nil {
				log.Printf("uploading sticker to WhatsApp: %v", err)
				return
			}

			fileLength := uint64(len(data))
			stkMsg := &waE2E.StickerMessage{
				URL:           &uploadResp.URL,
				DirectPath:    &uploadResp.DirectPath,
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    &fileLength,
				Mimetype:      proto.String("image/webp"),
				Width:         proto.Uint32(512),
				Height:        proto.Uint32(512),
				ContextInfo:   ctxInfo,
			}
			var waMsg waE2E.Message
			waMsg.StickerMessage = stkMsg

			resp, err := waClient.SendMessage(ctx, targetJID, &waMsg)
			if err != nil {
				log.Printf("sending sticker message: %v", err)
				return
			}
			msgID = resp.ID
			rawProto, _ = proto.Marshal(&waMsg)
		} else {
			msgID = fmt.Sprintf("dev_sticker_%d", time.Now().UnixNano())
			var waMsg waE2E.Message
			waMsg.StickerMessage = &waE2E.StickerMessage{
				Mimetype:    proto.String("image/webp"),
				Width:       proto.Uint32(512),
				Height:      proto.Uint32(512),
				ContextInfo: ctxInfo,
			}
			rawProto, _ = proto.Marshal(&waMsg)
		}

		initStatus := MessageStatusSent
		if c.isSelfJID(chatJIDStr) {
			initStatus = MessageStatusRead
		}

		sentMsg := StoredMessage{
			ChatJID:     chatJIDStr,
			ID:          msgID,
			SenderJID:   "me",
			FromMe:      true,
			Timestamp:   time.Now(),
			Kind:        messageKindSticker,
			Text:        "",
			Raw:         rawProto,
			Status:      initStatus,
			SenderSaved: true,
		}
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			sentMsg.SenderJID = waClient.Store.ID.ToNonAD().String()
		}

		destPath := getLocalMediaPath(sentMsg)
		_ = os.WriteFile(destPath, data, 0o600)

		if msgStore != nil {
			_, _ = msgStore.db.ExecContext(ctx, `
				INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (chat_jid, id) DO UPDATE SET status = MAX(whatsdown_messages.status, excluded.status)`,
				sentMsg.ChatJID, sentMsg.ID, sentMsg.SenderJID, "", 1, sentMsg.Timestamp.Unix(), sentMsg.Kind, sentMsg.Text, sentMsg.Raw, sentMsg.Status)

			_, _ = msgStore.db.ExecContext(ctx, `
				UPDATE whatsdown_chats SET last_message_at = ? WHERE jid = ?`,
				sentMsg.Timestamp.Unix(), sentMsg.ChatJID)

			glib.IdleAdd(func() {
				if c.currentChat != nil && (c.currentChat.JID == chatJIDStr || (c.isSelfJID(c.currentChat.JID) && c.isSelfJID(chatJIDStr))) {
					c.currentMessages = append(c.currentMessages, sentMsg)
					bubble := c.createMessageBubble(sentMsg)
					if c.messageRowMap != nil {
						c.messageRowMap[sentMsg.ID] = bubble
					}
					c.messagesList.Append(bubble)
					c.scrollToBottomAnimated()
				}
				c.loadChats()
				c.loadArchivedChats()
			})
		}
	}()
}

// toggleMessageStarred flips the starred status of a message and updates the database & WhatsApp.
func (c *whatsDownMainController) toggleMessageStarred(msg StoredMessage) {
	if msgStore == nil {
		return
	}
	newStarred := !msg.Starred
	go func() {
		_ = msgStore.toggleMessageStarred(context.Background(), msg.ChatJID, msg.ID, newStarred)
		if waClient != nil {
			chatJID, err1 := types.ParseJID(msg.ChatJID)
			senderJID, err2 := types.ParseJID(msg.SenderJID)
			if err1 == nil && err2 == nil {
				_ = waClient.SendAppState(context.Background(), appstate.BuildStar(chatJID, senderJID, types.MessageID(msg.ID), msg.FromMe, newStarred))
			}
		}
		glib.IdleAdd(func() {
			if c.currentChat != nil && c.currentChat.JID == msg.ChatJID {
				c.loadMessages(msg.ChatJID)
			}
		})
	}()
}

// startReply initiates replying to a specific message, showing the preview bar above the message entry.
func (c *whatsDownMainController) startReply(msg StoredMessage) {
	c.replyingTo = &msg

	sender := "You"
	if !msg.FromMe && !c.isSelfJID(msg.ChatJID) {
		sender = msg.SenderName
		if sender == "" || strings.Contains(sender, "@") || strings.HasPrefix(sender, "+") {
			if msg.SenderJID != "" {
				if resolved := c.resolveContactName(msg.SenderJID); resolved != "" {
					sender = resolved
				}
			}
		}
		if sender == "" {
			sender = cleanPhoneNumber(msg.SenderJID)
		}
		if sender == "" {
			sender = "Sender"
		}
	}
	if c.replyPreviewSender != nil {
		c.replyPreviewSender.SetText(fmt.Sprintf("Replying to %s", sender))
	}

	previewText := msg.Text
	if previewText == "" {
		switch msg.Kind {
		case messageKindImage:
			previewText = "📷 Photo"
		case messageKindVideo:
			previewText = "🎥 Video"
		case messageKindAudio, messageKindVoice:
			previewText = "🎵 Audio"
		case messageKindDocument:
			previewText = "📄 Document"
		case messageKindLocation:
			previewText = "📍 Location"
		case messageKindPoll:
			previewText = "📊 Poll"
		case messageKindSticker:
			previewText = "🏷️ Sticker"
		default:
			previewText = "Message"
		}
	}
	if c.replyPreviewText != nil {
		c.replyPreviewText.SetText(previewText)
	}
	if c.replyPreviewBar != nil {
		c.replyPreviewBar.SetVisible(true)
	}
	if c.messageEntry != nil {
		c.messageEntry.GrabFocus()
	}
}

// cancelReply clears the active reply state and hides the preview bar.
func (c *whatsDownMainController) cancelReply() {
	c.replyingTo = nil
	if c.replyPreviewBar != nil {
		c.replyPreviewBar.SetVisible(false)
	}
}

// startEditing populates the messageEntry with the message text and reveals the edit banner.
func (c *whatsDownMainController) startEditing(msg StoredMessage) {
	c.cancelReply()
	mCopy := msg
	c.editingMessage = &mCopy
	if c.editBannerText != nil {
		c.editBannerText.SetText(msg.Text)
	}
	if c.editBannerBar != nil {
		c.editBannerBar.SetVisible(true)
	}
	if c.messageEntry != nil {
		c.messageEntry.SetText(msg.Text)
		c.messageEntry.SetPosition(len([]rune(msg.Text)))
		c.messageEntry.GrabFocus()
	}
}

// cancelEditing clears the active message editing state and hides the edit banner.
func (c *whatsDownMainController) cancelEditing() {
	c.editingMessage = nil
	if c.editBannerBar != nil {
		c.editBannerBar.SetVisible(false)
	}
	if c.messageEntry != nil {
		c.messageEntry.SetText("")
	}
}

// submitEdit updates the message in local storage and sends the edit protocol message via WhatsApp.
func (c *whatsDownMainController) submitEdit(msg StoredMessage, newText string) {
	c.cancelEditing()
	if strings.TrimSpace(newText) == "" || newText == msg.Text {
		return
	}

	// 1. Save edit to local DB (archiving previous version)
	if msgStore != nil {
		if err := msgStore.saveMessageEdit(context.Background(), msg.ChatJID, msg.ID, newText); err != nil {
			log.Printf("saving message edit: %v", err)
		}
	}

	// 2. Send edit via WhatsApp if connected
	if waClient != nil {
		targetJID, err := types.ParseJID(msg.ChatJID)
		if err == nil {
			editMsg := waClient.BuildEdit(targetJID, msg.ID, &waE2E.Message{
				Conversation: proto.String(newText),
			})
			go func() {
				_, err := waClient.SendMessage(context.Background(), targetJID, editMsg)
				if err != nil {
					log.Printf("sending edit message: %v", err)
				}
			}()
		}
	}

	// 3. Reload active chat messages
	if c.currentChat != nil && c.currentChat.JID == msg.ChatJID {
		c.loadMessages(msg.ChatJID)
	}
	c.loadChats()
}

// showEditHistoryDialog displays a modal dialog showing previous versions and the latest edit of a message.
func (c *whatsDownMainController) showEditHistoryDialog(msg StoredMessage) {
	var edits []MessageEdit
	if msgStore != nil {
		var err error
		edits, err = msgStore.getMessageEdits(context.Background(), msg.ChatJID, msg.ID)
		if err != nil {
			log.Printf("fetching message edits: %v", err)
		}
	}

	win := adw.NewWindow()
	win.SetTransientFor(&c.window.Window)
	win.SetModal(true)
	win.SetTitle("Edit History")
	win.SetDefaultSize(440, 480)

	tbView := adw.NewToolbarView()
	hb := adw.NewHeaderBar()
	hb.SetTitleWidget(adw.NewWindowTitle("Edit History", cleanPhoneNumber(msg.ChatJID)))
	tbView.AddTopBar(hb)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 12)
	mainBox.SetMarginTop(16)
	mainBox.SetMarginBottom(16)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	// List of historical edits
	for i, e := range edits {
		versionLabel := fmt.Sprintf("Edit %d", i+1)
		if i == 0 {
			versionLabel = "Original Message"
		}

		card := gtk.NewBox(gtk.OrientationVertical, 8)
		card.AddCSSClass("card")
		card.AddCSSClass("edit-history-card")
		card.SetMarginBottom(8)

		topRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
		tag := gtk.NewLabel(versionLabel)
		tag.AddCSSClass("caption")
		tag.AddCSSClass("dim-label")
		tag.SetHExpand(true)
		tag.SetXAlign(0)

		timeLbl := gtk.NewLabel(e.EditedAt.Format("02 Jan 15:04"))
		timeLbl.AddCSSClass("caption")
		timeLbl.AddCSSClass("dim-label")
		timeLbl.SetXAlign(1)

		topRow.Append(tag)
		topRow.Append(timeLbl)
		card.Append(topRow)

		markup, _ := FormatWhatsAppMarkdownForBubble(e.Text, nil, nil)
		textLbl := gtk.NewLabel("")
		textLbl.SetWrap(true)
		textLbl.SetWrapMode(pango.WrapWordChar)
		textLbl.SetXAlign(0)
		textLbl.SetSelectable(true)
		if _, _, _, err := pango.ParseMarkup(markup, 0); err == nil {
			textLbl.SetMarkup(markup)
		} else {
			textLbl.SetText(e.Text)
		}
		card.Append(textLbl)
		mainBox.Append(card)
	}

	// Current / Latest version
	currentCard := gtk.NewBox(gtk.OrientationVertical, 8)
	currentCard.AddCSSClass("card")
	currentCard.AddCSSClass("edit-history-card")
	currentCard.AddCSSClass("edit-history-card-latest")

	topRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	tag := gtk.NewLabel("Latest Version (Current)")
	tag.AddCSSClass("caption")
	tag.AddCSSClass("accent")
	tag.SetHExpand(true)
	tag.SetXAlign(0)

	timeLbl := gtk.NewLabel(msg.Timestamp.Format("02 Jan 15:04"))
	timeLbl.AddCSSClass("caption")
	timeLbl.AddCSSClass("dim-label")
	timeLbl.SetXAlign(1)

	topRow.Append(tag)
	topRow.Append(timeLbl)
	currentCard.Append(topRow)

	markup, _ := FormatWhatsAppMarkdownForBubble(msg.Text, nil, nil)
	textLbl := gtk.NewLabel("")
	textLbl.SetWrap(true)
	textLbl.SetWrapMode(pango.WrapWordChar)
	textLbl.SetXAlign(0)
	textLbl.SetSelectable(true)
	if _, _, _, err := pango.ParseMarkup(markup, 0); err == nil {
		textLbl.SetMarkup(markup)
	} else {
		textLbl.SetText(msg.Text)
	}
	currentCard.Append(textLbl)
	mainBox.Append(currentCard)

	scrolled.SetChild(mainBox)
	tbView.SetContent(scrolled)
	win.SetContent(tbView)
	win.Present()
}

// buildReplyContextInfo constructs a waE2E.ContextInfo quoting the specified message.
func (c *whatsDownMainController) buildReplyContextInfo(quoted StoredMessage) *waE2E.ContextInfo {
	var quotedRaw *waE2E.Message
	if len(quoted.Raw) > 0 {
		quotedRaw = parseE2EMessage(quoted.Raw)
	}
	if quotedRaw == nil {
		quotedRaw = &waE2E.Message{
			Conversation: proto.String(quoted.Text),
		}
	} else {
		// Clone and strip any nested ContextInfo from the quoted snapshot
		quotedRaw = proto.Clone(quotedRaw).(*waE2E.Message)
		if quotedRaw.ExtendedTextMessage != nil {
			quotedRaw.ExtendedTextMessage.ContextInfo = nil
		}
		if quotedRaw.ImageMessage != nil {
			quotedRaw.ImageMessage.ContextInfo = nil
		}
		if quotedRaw.VideoMessage != nil {
			quotedRaw.VideoMessage.ContextInfo = nil
		}
		if quotedRaw.AudioMessage != nil {
			quotedRaw.AudioMessage.ContextInfo = nil
		}
		if quotedRaw.DocumentMessage != nil {
			quotedRaw.DocumentMessage.ContextInfo = nil
		}
		if quotedRaw.StickerMessage != nil {
			quotedRaw.StickerMessage.ContextInfo = nil
		}
	}

	var participant string
	if quoted.FromMe || c.isSelfJID(quoted.ChatJID) {
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			participant = waClient.Store.ID.ToNonAD().String()
		}
	} else {
		participant = quoted.SenderJID
		if participant == "" {
			participant = quoted.ChatJID
		}
	}
	if pj, err := types.ParseJID(participant); err == nil {
		participant = pj.ToNonAD().String()
	}

	ctxInfo := &waE2E.ContextInfo{
		StanzaID:      proto.String(quoted.ID),
		QuotedMessage: quotedRaw,
	}
	if participant != "" {
		ctxInfo.Participant = proto.String(participant)
	}
	return ctxInfo
}

// createQuoteWidget builds a quote preview block inside a chat bubble if msg replies to another message.
func (c *whatsDownMainController) createQuoteWidget(msg StoredMessage) *gtk.Box {
	ctxInfo := extractContextInfo(msg.Raw)
	if ctxInfo == nil {
		return nil
	}
	stanzaID := ctxInfo.GetStanzaID()
	participant := ctxInfo.GetParticipant()
	quotedRaw := ctxInfo.GetQuotedMessage()
	if stanzaID == "" && quotedRaw == nil {
		return nil
	}

	var origMsg *StoredMessage
	if msgStore != nil && stanzaID != "" {
		origMsg, _ = msgStore.getMessageByID(context.Background(), stanzaID)
	}

	quotedSenderName := ""
	if origMsg != nil {
		if origMsg.FromMe || c.isSelfJID(origMsg.SenderJID) {
			quotedSenderName = "You"
		} else {
			quotedSenderName = origMsg.SenderName
			if quotedSenderName == "" || strings.Contains(quotedSenderName, "@") || strings.HasPrefix(quotedSenderName, "+") {
				if origMsg.SenderJID != "" {
					if resolved := c.resolveContactName(origMsg.SenderJID); resolved != "" {
						quotedSenderName = resolved
					}
				}
			}
			if quotedSenderName == "" {
				quotedSenderName = cleanPhoneNumber(origMsg.SenderJID)
			}
		}
	}
	if quotedSenderName == "" {
		if c.isSelfJID(participant) {
			quotedSenderName = "You"
		} else if participant != "" {
			if resolved := c.resolveContactName(participant); resolved != "" {
				quotedSenderName = resolved
			} else {
				quotedSenderName = cleanPhoneNumber(participant)
			}
		}
	}
	if quotedSenderName == "" {
		if msg.FromMe {
			quotedSenderName = "You"
		} else {
			quotedSenderName = "Message"
		}
	}

	var quotedSnippet string
	if origMsg != nil && origMsg.Text != "" {
		quotedSnippet = origMsg.Text
	} else if quotedRaw != nil {
		quotedSnippet, _ = summarizeProtoMessage(quotedRaw)
	} else if origMsg != nil {
		switch origMsg.Kind {
		case messageKindImage:
			quotedSnippet = "📷 Photo"
		case messageKindVideo:
			quotedSnippet = "🎥 Video"
		case messageKindAudio, messageKindVoice:
			quotedSnippet = "🎵 Audio"
		case messageKindDocument:
			quotedSnippet = "📄 Document"
		case messageKindLocation:
			quotedSnippet = "📍 Location"
		case messageKindPoll:
			quotedSnippet = "📊 Poll"
		case messageKindSticker:
			quotedSnippet = "🏷️ Sticker"
		default:
			quotedSnippet = "Message"
		}
	} else {
		quotedSnippet = "Original message"
	}

	quoteBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	quoteBox.AddCSSClass("chat-bubble-quote")
	quoteBox.SetCursorFromName("pointer")

	contentBox := gtk.NewBox(gtk.OrientationVertical, 2)
	contentBox.SetHExpand(true)

	colorSeed := participant
	if colorSeed == "" {
		colorSeed = quotedSenderName
	}
	authorColor := getSenderColor(colorSeed)
	if quotedSenderName == "You" {
		authorColor = "#3584e4"
	}

	authorLabel := gtk.NewLabel("")
	authorLabel.SetMarkup(fmt.Sprintf(`<span foreground="%s" weight="bold">%s</span>`, authorColor, glib.MarkupEscapeText(quotedSenderName)))
	authorLabel.SetXAlign(0)
	authorLabel.AddCSSClass("chat-bubble-quote-sender")
	contentBox.Append(authorLabel)

	textLabel := gtk.NewLabel(quotedSnippet)
	textLabel.SetXAlign(0)
	textLabel.SetEllipsize(pango.EllipsizeEnd)
	textLabel.SetMaxWidthChars(36)
	textLabel.AddCSSClass("chat-bubble-quote-text")
	contentBox.Append(textLabel)

	quoteBox.Append(contentBox)

	// Show small thumbnail if available
	if origMsg != nil && (origMsg.Kind == messageKindImage || origMsg.Kind == messageKindVideo) {
		if tex := getMediaThumbnailTexture(*origMsg); tex != nil {
			thumbPic := gtk.NewPicture()
			thumbPic.SetKeepAspectRatio(true)
			thumbPic.SetCanShrink(true)
			thumbPic.SetSizeRequest(36, 36)
			thumbPic.SetPaintable(tex)
			thumbPic.AddCSSClass("chat-bubble-quote-thumb")
			quoteBox.Append(thumbPic)
		}
	} else if quotedRaw != nil {
		var jpegBytes []byte
		if img := quotedRaw.GetImageMessage(); img != nil && len(img.GetJPEGThumbnail()) > 0 {
			jpegBytes = img.GetJPEGThumbnail()
		} else if vid := quotedRaw.GetVideoMessage(); vid != nil && len(vid.GetJPEGThumbnail()) > 0 {
			jpegBytes = vid.GetJPEGThumbnail()
		}
		if len(jpegBytes) > 0 {
			if tex, err := gdk.NewTextureFromBytes(glib.NewBytes(jpegBytes)); err == nil && tex != nil {
				thumbPic := gtk.NewPicture()
				thumbPic.SetKeepAspectRatio(true)
				thumbPic.SetCanShrink(true)
				thumbPic.SetSizeRequest(36, 36)
				thumbPic.SetPaintable(tex)
				thumbPic.AddCSSClass("chat-bubble-quote-thumb")
				quoteBox.Append(thumbPic)
			}
		}
	}

	if stanzaID != "" {
		targetID := stanzaID
		gesture := gtk.NewGestureClick()
		gesture.ConnectPressed(func(nPress int, x, y float64) {
			c.scrollToMessageID(targetID)
		})
		quoteBox.AddController(gesture)
	}

	return quoteBox
}

// scrollToMessageID scrolls the chat viewport to the message with the given ID and highlights it.
func (c *whatsDownMainController) scrollToMessageID(id string) {
	if c.messageRowMap == nil {
		return
	}
	row, ok := c.messageRowMap[id]
	if !ok || row == nil {
		return
	}

	if c.messagesScrolledWindow != nil {
		if vp, ok := c.messagesScrolledWindow.Child().(*gtk.Viewport); ok {
			vp.ScrollTo(row, nil)
		} else {
			row.GrabFocus()
		}
	} else {
		row.GrabFocus()
	}

	row.AddCSSClass("message-highlight")
	glib.TimeoutAdd(1200, func() bool {
		row.RemoveCSSClass("message-highlight")
		return false
	})
}

// sendMessage sends the text from messageEntry to the currently selected chat.
func (c *whatsDownMainController) sendMessage() {
	if c.currentChat == nil || waClient == nil {
		return
	}

	text := c.messageEntry.Text()
	if text == "" {
		return
	}
	c.messageEntry.SetText("")
	c.messageEntry.SetAttributes(pango.NewAttrList())
	if c.markdownPreviewBar != nil {
		c.markdownPreviewBar.SetVisible(false)
	}

	if c.editingMessage != nil {
		targetMsg := *c.editingMessage
		c.submitEdit(targetMsg, text)
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	targetJID, err := types.ParseJID(c.currentChat.JID)
	if err != nil {
		log.Printf("parsing target JID: %v", err)
		return
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}
	go func() {
		ctx := context.Background()
		var waMsg waE2E.Message
		if ctxInfo != nil {
			waMsg = waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text:        proto.String(text),
					ContextInfo: ctxInfo,
				},
			}
		} else {
			waMsg = waE2E.Message{
				Conversation: proto.String(text),
			}
		}

		resp, err := waClient.SendMessage(ctx, targetJID, &waMsg)
		if err != nil {
			log.Printf("sending message to %s: %v", targetJID, err)
			return
		}

		rawProto, _ := proto.Marshal(&waMsg)
		initStatus := MessageStatusSent
		if c.isSelfJID(chatJIDStr) {
			initStatus = MessageStatusRead
		}
		if msgStore != nil {
			sentMsg := StoredMessage{
				ChatJID:   chatJIDStr,
				ID:        resp.ID,
				SenderJID: waClient.Store.ID.ToNonAD().String(),
				FromMe:    true,
				Timestamp: time.Now(),
				Kind:      messageKindText,
				Text:      text,
				Raw:       rawProto,
				Status:    initStatus,
			}
			_, _ = msgStore.db.ExecContext(ctx, `
				INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (chat_jid, id) DO UPDATE SET status = MAX(whatsdown_messages.status, excluded.status)`,
				sentMsg.ChatJID, sentMsg.ID, sentMsg.SenderJID, "", 1, sentMsg.Timestamp.Unix(), sentMsg.Kind, sentMsg.Text, sentMsg.Raw, sentMsg.Status)

			_, _ = msgStore.db.ExecContext(ctx, `
				UPDATE whatsdown_chats SET last_message_at = ? WHERE jid = ?`,
				sentMsg.Timestamp.Unix(), sentMsg.ChatJID)

			glib.IdleAdd(func() {
				if c.currentChat != nil && (c.currentChat.JID == chatJIDStr || (c.isSelfJID(c.currentChat.JID) && c.isSelfJID(chatJIDStr))) {
					c.currentMessages = append(c.currentMessages, sentMsg)
					bubble := c.createMessageBubble(sentMsg)
					if c.messageRowMap != nil {
						c.messageRowMap[sentMsg.ID] = bubble
					}
					c.messagesList.Append(bubble)
					c.scrollToBottomAnimated()
				}
				c.loadChats()
				c.loadArchivedChats()
			})
		}
	}()
}

// openFileChooserForMedia displays a native file chooser for attaching media or documents.
func (c *whatsDownMainController) openFileChooserForMedia(imagesAndVideosOnly bool) {
	title := "Choose Document to Send"
	if imagesAndVideosOnly {
		title = "Choose Photo or Video to Send"
	}

	dialog := gtk.NewFileChooserNative(title, &c.window.Window, gtk.FileChooserActionOpen, "_Select", "_Cancel")
	filter := gtk.NewFileFilter()
	if imagesAndVideosOnly {
		filter.SetName("Photos & Videos")
		filter.AddMIMEType("image/*")
		filter.AddMIMEType("video/*")
		filter.AddPattern("*.jpg")
		filter.AddPattern("*.jpeg")
		filter.AddPattern("*.png")
		filter.AddPattern("*.webp")
		filter.AddPattern("*.gif")
		filter.AddPattern("*.mp4")
		filter.AddPattern("*.mov")
		filter.AddPattern("*.mkv")
		filter.AddPattern("*.webm")
	} else {
		filter.SetName("All Documents & Files")
		filter.AddMIMEType("application/pdf")
		filter.AddMIMEType("application/*")
		filter.AddMIMEType("text/*")
		filter.AddPattern("*")
	}
	dialog.SetFilter(filter)

	dialog.ConnectResponse(func(responseId int) {
		if responseId == int(gtk.ResponseAccept) {
			if file := dialog.File(); file != nil {
				path := file.Path()
				if path != "" {
					glib.IdleAdd(func() {
						c.showSendAttachmentDialog(path)
					})
				}
			}
		}
	})
	dialog.Show()
}

// showSendAttachmentDialog presents an in-app preview dialog with preview of audio, video, photo, or PDF
// before confirming and sending the attachment.
func (c *whatsDownMainController) showSendAttachmentDialog(filePath string) {
	if c.currentChat == nil {
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	filename := filepath.Base(filePath)

	fi, err := os.Stat(filePath)
	if err != nil {
		return
	}
	sizeStr := formatFileSize(uint64(fi.Size()))

	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(480, 520)

	titleStr := "Send File"
	isImage := false
	isVideo := false
	isAudio := false
	isPDF := false

	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp":
		titleStr = "Send Photo"
		isImage = true
	case ".mp4", ".mov", ".mkv", ".webm", ".avi":
		titleStr = "Send Video"
		isVideo = true
	case ".mp3", ".ogg", ".opus", ".m4a", ".wav", ".aac", ".flac":
		titleStr = "Send Audio"
		isAudio = true
	case ".pdf":
		titleStr = "Send PDF"
		isPDF = true
	default:
		titleStr = "Send Document"
	}
	dialog.SetTitle(titleStr)

	dialog.ConnectCloseRequest(func() bool {
		stopCurrentAudio()
		return false
	})

	headerBar := adw.NewHeaderBar()
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("flat")
	cancelBtn.ConnectClicked(func() {
		stopCurrentAudio()
		dialog.Close()
	})
	headerBar.PackStart(cancelBtn)

	sendBtn := gtk.NewButtonWithLabel("Send")
	sendBtn.AddCSSClass("suggested-action")
	headerBar.PackEnd(sendBtn)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 12)
	mainBox.SetMarginTop(12)
	mainBox.SetMarginBottom(16)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	previewContainer := gtk.NewBox(gtk.OrientationVertical, 8)
	previewContainer.SetVExpand(true)
	previewContainer.SetHAlign(gtk.AlignCenter)
	previewContainer.SetVAlign(gtk.AlignCenter)

	if isImage {
		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.SetSizeRequest(360, 260)
		pic.AddCSSClass("chat-bubble-picture")
		if tex, err := gdk.NewTextureFromFilename(filePath); err == nil && tex != nil {
			pic.SetPaintable(tex)
		}
		previewContainer.Append(pic)
	} else if isVideo {
		overlay := gtk.NewOverlay()
		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.SetSizeRequest(360, 260)
		pic.AddCSSClass("chat-bubble-picture")
		if tex := getVideoThumbnailTexture(filePath); tex != nil {
			pic.SetPaintable(tex)
		}
		overlay.SetChild(pic)

		playBtn := gtk.NewButtonFromIconName("media-playback-start-symbolic")
		playBtn.AddCSSClass("circular")
		playBtn.AddCSSClass("suggested-action")
		playBtn.SetHAlign(gtk.AlignCenter)
		playBtn.SetVAlign(gtk.AlignCenter)
		playBtn.SetTooltipText("Preview Video")
		playBtn.ConnectClicked(func() {
			_ = openFile(filePath)
		})
		overlay.AddOverlay(playBtn)
		previewContainer.Append(overlay)
	} else if isPDF {
		overlay := gtk.NewOverlay()
		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.SetSizeRequest(320, 240)
		pic.AddCSSClass("chat-bubble-picture")
		totalPages := getPDFPageCount(filePath)
		currentPage := 1

		if tex := getPDFPageTexture(filePath, 1); tex != nil {
			pic.SetPaintable(tex)
		} else {
			go func() {
				tex := getPDFPageTexture(filePath, 1)
				if tex != nil {
					glib.IdleAdd(func() {
						pic.SetPaintable(tex)
					})
				}
			}()
		}
		overlay.SetChild(pic)

		badge := gtk.NewLabel("PDF Preview")
		badge.AddCSSClass("media-video-badge")
		badge.SetHAlign(gtk.AlignStart)
		badge.SetVAlign(gtk.AlignStart)
		badge.SetMarginStart(6)
		badge.SetMarginTop(6)
		overlay.AddOverlay(badge)

		previewContainer.Append(overlay)

		// Page switcher bar
		switcherBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
		switcherBox.SetHAlign(gtk.AlignCenter)
		switcherBox.SetMarginTop(8)
		switcherBox.AddCSSClass("pdf-page-switcher")

		prevBtn := gtk.NewButtonFromIconName("go-previous-symbolic")
		prevBtn.AddCSSClass("flat")
		prevBtn.AddCSSClass("circular")
		prevBtn.SetSensitive(false)
		prevBtn.SetTooltipText("Previous page")

		pageLbl := gtk.NewLabel(fmt.Sprintf("Page %d of %d", currentPage, totalPages))
		pageLbl.AddCSSClass("caption")
		pageLbl.AddCSSClass("dim-label")

		nextBtn := gtk.NewButtonFromIconName("go-next-symbolic")
		nextBtn.AddCSSClass("flat")
		nextBtn.AddCSSClass("circular")
		nextBtn.SetSensitive(totalPages > 1)
		nextBtn.SetTooltipText("Next page")

		updatePage := func(newPage int) {
			if newPage < 1 || newPage > totalPages {
				return
			}
			currentPage = newPage
			pageLbl.SetText(fmt.Sprintf("Page %d of %d", currentPage, totalPages))
			prevBtn.SetSensitive(currentPage > 1)
			nextBtn.SetSensitive(currentPage < totalPages)
			go func() {
				tex := getPDFPageTexture(filePath, currentPage)
				if tex != nil {
					glib.IdleAdd(func() {
						pic.SetPaintable(tex)
					})
				}
			}()
		}

		prevBtn.ConnectClicked(func() {
			updatePage(currentPage - 1)
		})
		nextBtn.ConnectClicked(func() {
			updatePage(currentPage + 1)
		})

		switcherBox.Append(prevBtn)
		switcherBox.Append(pageLbl)
		switcherBox.Append(nextBtn)
		previewContainer.Append(switcherBox)
	} else if isAudio {
		cardBox := gtk.NewBox(gtk.OrientationVertical, 12)
		cardBox.SetHAlign(gtk.AlignCenter)

		icon := gtk.NewImageFromIconName("folder-music-symbolic")
		icon.SetPixelSize(64)
		cardBox.Append(icon)

		playBtn := gtk.NewButton()
		playBtn.AddCSSClass("pill")
		playBtn.AddCSSClass("suggested-action")
		playBtn.SetHAlign(gtk.AlignCenter)

		btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
		btnIcon := gtk.NewImageFromIconName("media-playback-start-symbolic")
		btnLbl := gtk.NewLabel("Play Audio Preview")
		btnBox.Append(btnIcon)
		btnBox.Append(btnLbl)
		playBtn.SetChild(btnBox)

		isPlaying := false
		playBtn.ConnectClicked(func() {
			if isPlaying {
				stopCurrentAudio()
				return
			}
			playAudioPreview(filePath, func(playing bool) {
				isPlaying = playing
				if playing {
					btnIcon.SetFromIconName("media-playback-pause-symbolic")
					btnLbl.SetText("Pause Preview")
				} else {
					btnIcon.SetFromIconName("media-playback-start-symbolic")
					btnLbl.SetText("Play Audio Preview")
				}
			})
		})
		cardBox.Append(playBtn)
		previewContainer.Append(cardBox)
	} else {
		icon := gtk.NewImageFromIconName("folder-documents-symbolic")
		icon.SetPixelSize(72)
		previewContainer.Append(icon)
	}

	fileInfo := gtk.NewLabel(fmt.Sprintf("%s (%s)", filename, sizeStr))
	fileInfo.AddCSSClass("caption")
	fileInfo.AddCSSClass("dim-label")
	fileInfo.SetHAlign(gtk.AlignCenter)
	previewContainer.Append(fileInfo)

	mainBox.Append(previewContainer)

	captionEntry := gtk.NewEntry()
	captionEntry.SetPlaceholderText("Add a caption…")
	captionEntry.SetHExpand(true)
	mainBox.Append(captionEntry)

	doSend := func() {
		stopCurrentAudio()
		capText := captionEntry.Text()
		dialog.Close()
		c.sendAttachment(filePath, capText)
	}

	sendBtn.ConnectClicked(doSend)
	captionEntry.ConnectActivate(doSend)

	dialog.SetTitlebar(headerBar)
	dialog.SetChild(mainBox)
	dialog.Present()
}

// sendAttachment handles uploading (if connected), storing, and displaying outgoing media files.
func (c *whatsDownMainController) sendAttachment(filePath, caption string) {
	if c.currentChat == nil {
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("reading attachment file: %v", err)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	filename := filepath.Base(filePath)
	var kind string
	var appMediaType whatsmeow.MediaType
	var mimeType string

	switch ext {
	case ".jpg", ".jpeg":
		kind = messageKindImage
		appMediaType = whatsmeow.MediaImage
		mimeType = "image/jpeg"
	case ".png":
		kind = messageKindImage
		appMediaType = whatsmeow.MediaImage
		mimeType = "image/png"
	case ".webp":
		kind = messageKindImage
		appMediaType = whatsmeow.MediaImage
		mimeType = "image/webp"
	case ".mp4", ".mov", ".mkv", ".webm", ".avi":
		kind = messageKindVideo
		appMediaType = whatsmeow.MediaVideo
		mimeType = "video/mp4"
	case ".ogg", ".opus", ".mp3", ".m4a", ".wav", ".aac", ".flac":
		kind = messageKindAudio
		appMediaType = whatsmeow.MediaAudio
		mimeType = "audio/ogg"
		if ext == ".mp3" {
			mimeType = "audio/mpeg"
		} else if ext == ".m4a" {
			mimeType = "audio/mp4"
		}
	default:
		kind = messageKindDocument
		appMediaType = whatsmeow.MediaDocument
		mimeType = "application/octet-stream"
		if ext == ".pdf" {
			mimeType = "application/pdf"
		}
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}
	go func() {
		ctx := context.Background()
		var msgID string
		var rawProto []byte

		if waClient != nil {
			targetJID, err := types.ParseJID(chatJIDStr)
			if err != nil {
				log.Printf("parsing JID for media send: %v", err)
				return
			}

			uploadResp, err := waClient.Upload(ctx, data, appMediaType)
			if err != nil {
				log.Printf("uploading media to WhatsApp: %v", err)
				return
			}

			fileLength := uint64(len(data))
			var waMsg waE2E.Message
			switch kind {
			case messageKindImage:
				imgMsg := &waE2E.ImageMessage{
					Caption:       proto.String(caption),
					Mimetype:      proto.String(mimeType),
					URL:           &uploadResp.URL,
					DirectPath:    &uploadResp.DirectPath,
					MediaKey:      uploadResp.MediaKey,
					FileEncSHA256: uploadResp.FileEncSHA256,
					FileSHA256:    uploadResp.FileSHA256,
					FileLength:    &fileLength,
					ContextInfo:   ctxInfo,
				}
				waMsg.ImageMessage = imgMsg
			case messageKindVideo:
				vidMsg := &waE2E.VideoMessage{
					Caption:       proto.String(caption),
					Mimetype:      proto.String(mimeType),
					URL:           &uploadResp.URL,
					DirectPath:    &uploadResp.DirectPath,
					MediaKey:      uploadResp.MediaKey,
					FileEncSHA256: uploadResp.FileEncSHA256,
					FileSHA256:    uploadResp.FileSHA256,
					FileLength:    &fileLength,
					ContextInfo:   ctxInfo,
				}
				waMsg.VideoMessage = vidMsg
			case messageKindAudio:
				audMsg := &waE2E.AudioMessage{
					Mimetype:      proto.String(mimeType),
					URL:           &uploadResp.URL,
					DirectPath:    &uploadResp.DirectPath,
					MediaKey:      uploadResp.MediaKey,
					FileEncSHA256: uploadResp.FileEncSHA256,
					FileSHA256:    uploadResp.FileSHA256,
					FileLength:    &fileLength,
					ContextInfo:   ctxInfo,
				}
				waMsg.AudioMessage = audMsg
			default:
				docMsg := &waE2E.DocumentMessage{
					Caption:       proto.String(caption),
					FileName:      proto.String(filename),
					Mimetype:      proto.String(mimeType),
					URL:           &uploadResp.URL,
					DirectPath:    &uploadResp.DirectPath,
					MediaKey:      uploadResp.MediaKey,
					FileEncSHA256: uploadResp.FileEncSHA256,
					FileSHA256:    uploadResp.FileSHA256,
					FileLength:    &fileLength,
					ContextInfo:   ctxInfo,
				}
				waMsg.DocumentMessage = docMsg
			}

			resp, err := waClient.SendMessage(ctx, targetJID, &waMsg)
			if err != nil {
				log.Printf("sending media message: %v", err)
				return
			}
			msgID = resp.ID
			rawProto, _ = proto.Marshal(&waMsg)
		} else {
			msgID = fmt.Sprintf("dev_media_%d", time.Now().UnixNano())
			var waMsg waE2E.Message
			switch kind {
			case messageKindImage:
				waMsg.ImageMessage = &waE2E.ImageMessage{Caption: proto.String(caption), Mimetype: proto.String(mimeType), ContextInfo: ctxInfo}
			case messageKindVideo:
				waMsg.VideoMessage = &waE2E.VideoMessage{Caption: proto.String(caption), Mimetype: proto.String(mimeType), ContextInfo: ctxInfo}
			case messageKindAudio:
				waMsg.AudioMessage = &waE2E.AudioMessage{Mimetype: proto.String(mimeType), ContextInfo: ctxInfo}
			default:
				waMsg.DocumentMessage = &waE2E.DocumentMessage{FileName: proto.String(filename), Caption: proto.String(caption), Mimetype: proto.String(mimeType), ContextInfo: ctxInfo}
			}
			rawProto, _ = proto.Marshal(&waMsg)
		}

		initStatus := MessageStatusSent
		if c.isSelfJID(chatJIDStr) {
			initStatus = MessageStatusRead
		}
		sentMsg := StoredMessage{
			ChatJID:     chatJIDStr,
			ID:          msgID,
			SenderJID:   "",
			SenderName:  "You",
			FromMe:      true,
			Timestamp:   time.Now(),
			Kind:        kind,
			Text:        caption,
			Raw:         rawProto,
			Status:      initStatus,
			SenderSaved: true,
		}

		destPath := getLocalMediaPath(sentMsg)
		_ = os.WriteFile(destPath, data, 0o600)

		if msgStore != nil {
			_, _ = msgStore.db.ExecContext(ctx, `
				INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (chat_jid, id) DO UPDATE SET status = MAX(whatsdown_messages.status, excluded.status)`,
				sentMsg.ChatJID, sentMsg.ID, sentMsg.SenderJID, "", 1, sentMsg.Timestamp.Unix(), sentMsg.Kind, sentMsg.Text, sentMsg.Raw, sentMsg.Status)

			_, _ = msgStore.db.ExecContext(ctx, `
				UPDATE whatsdown_chats SET last_message_at = ? WHERE jid = ?`,
				sentMsg.Timestamp.Unix(), sentMsg.ChatJID)

			glib.IdleAdd(func() {
				if c.currentChat != nil && c.currentChat.JID == chatJIDStr {
					c.currentMessages = append(c.currentMessages, sentMsg)
					bubble := c.createMessageBubble(sentMsg)
					if c.messageRowMap != nil {
						c.messageRowMap[sentMsg.ID] = bubble
					}
					c.messagesList.Append(bubble)
					c.scrollToBottomAnimated()
				}
				c.loadChats()
				c.loadArchivedChats()
			})
		}
	}()
}

// setupVoiceRecording wires up the voice message recording and meter controls.
func (c *whatsDownMainController) setupVoiceRecording() {
	if c.voiceRecordBtn != nil {
		c.voiceRecordBtn.ConnectClicked(func() {
			c.startVoiceRecording()
		})
	}
	if c.voiceDeleteBtn != nil {
		c.voiceDeleteBtn.ConnectClicked(func() {
			c.deleteVoiceRecording()
		})
	}
	if c.voicePauseBtn != nil {
		c.voicePauseBtn.ConnectClicked(func() {
			c.togglePauseVoiceRecording()
		})
	}
	if c.voiceStopBtn != nil {
		c.voiceStopBtn.ConnectClicked(func() {
			c.stopVoiceRecording()
		})
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.ConnectClicked(func() {
			c.togglePlayVoicePreview()
		})
	}
	if c.voiceSendBtn != nil {
		c.voiceSendBtn.ConnectClicked(func() {
			c.sendVoiceRecording()
		})
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.SetDrawFunc(func(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
			c.drawVoiceMeter(area, cr, width, height)
		})
	}
}

// startVoiceRecording begins audio capture and presents the recording bar.
func (c *whatsDownMainController) startVoiceRecording() {
	if c.currentChat == nil {
		return
	}
	stopCurrentAudio()
	c.stopVoicePreview()

	rec, err := newVoiceRecorder()
	if err != nil {
		log.Printf("startVoiceRecording error: %v", err)
		return
	}
	c.voiceRecorder = rec

	if c.chatInputStack != nil {
		c.chatInputStack.SetVisibleChildName("recording")
	}
	if c.voiceTimerLabel != nil {
		c.voiceTimerLabel.SetText("0:00")
	}
	if c.voiceStatusIcon != nil {
		c.voiceStatusIcon.SetFromIconName("media-record-symbolic")
		c.voiceStatusIcon.SetVisible(true)
	}
	if c.voicePauseBtn != nil {
		c.voicePauseBtn.SetIconName("media-playback-pause-symbolic")
		c.voicePauseBtn.SetTooltipText("Pause recording")
		c.voicePauseBtn.SetVisible(true)
	}
	if c.voiceStopBtn != nil {
		c.voiceStopBtn.SetVisible(true)
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetVisible(false)
	}
	if c.voiceSendBtn != nil {
		c.voiceSendBtn.SetVisible(false)
	}

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	c.voiceTimerHandle = glib.TimeoutAdd(50, func() bool {
		if c.voiceRecorder == nil {
			c.voiceTimerHandle = 0
			return false
		}
		dur := c.voiceRecorder.Duration()
		s := int(dur.Seconds())
		if c.voiceTimerLabel != nil {
			c.voiceTimerLabel.SetText(fmt.Sprintf("%d:%02d", s/60, s%60))
		}
		if c.voiceMeterArea != nil {
			c.voiceMeterArea.QueueDraw()
		}
		if c.voiceRecorder.stopped {
			c.voiceTimerHandle = 0
			return false
		}
		return true
	})
}

// togglePauseVoiceRecording toggles pause/resume of the active recording.
func (c *whatsDownMainController) togglePauseVoiceRecording() {
	if c.voiceRecorder == nil || c.voiceRecorder.stopped {
		return
	}
	if c.voiceRecorder.Paused() {
		if err := c.voiceRecorder.Resume(); err != nil {
			log.Printf("resuming voice recording: %v", err)
			return
		}
		if c.voicePauseBtn != nil {
			c.voicePauseBtn.SetIconName("media-playback-pause-symbolic")
			c.voicePauseBtn.SetTooltipText("Pause recording")
		}
		if c.voiceStatusIcon != nil {
			c.voiceStatusIcon.SetFromIconName("media-record-symbolic")
		}
	} else {
		c.voiceRecorder.Pause()
		if c.voicePauseBtn != nil {
			c.voicePauseBtn.SetIconName("media-playback-start-symbolic")
			c.voicePauseBtn.SetTooltipText("Resume recording")
		}
		if c.voiceStatusIcon != nil {
			c.voiceStatusIcon.SetFromIconName("media-playback-pause-symbolic")
		}
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.QueueDraw()
	}
}

// stopVoiceRecording ends audio capture and presents delete/send controls.
func (c *whatsDownMainController) stopVoiceRecording() {
	if c.voiceRecorder == nil {
		return
	}
	c.voiceRecorder.Stop()

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	dur := c.voiceRecorder.Duration()
	s := int(dur.Seconds())
	if c.voiceTimerLabel != nil {
		c.voiceTimerLabel.SetText(fmt.Sprintf("%d:%02d", s/60, s%60))
	}
	if c.voicePauseBtn != nil {
		c.voicePauseBtn.SetVisible(false)
	}
	if c.voiceStopBtn != nil {
		c.voiceStopBtn.SetVisible(false)
	}
	if c.voiceStatusIcon != nil {
		c.voiceStatusIcon.SetFromIconName("audio-x-generic-symbolic")
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetIconName("media-playback-start-symbolic")
		c.voicePlayBtn.SetVisible(true)
	}
	if c.voiceSendBtn != nil {
		c.voiceSendBtn.SetVisible(true)
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.QueueDraw()
	}
}

// deleteVoiceRecording discards the active or stopped recording.
func (c *whatsDownMainController) deleteVoiceRecording() {
	c.stopVoicePreview()

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	if c.voiceRecorder != nil {
		c.voiceRecorder.Stop()
		c.voiceRecorder = nil
	}

	if c.chatInputStack != nil {
		c.chatInputStack.SetVisibleChildName("normal")
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.QueueDraw()
	}
}

// togglePlayVoicePreview allows playing back the stopped recording before sending.
func (c *whatsDownMainController) togglePlayVoicePreview() {
	if c.voiceRecorder == nil {
		return
	}
	if c.voiceIsPlaying {
		c.stopVoicePreview()
		return
	}
	if c.voicePreviewPath == "" {
		tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("preview_%d.ogg", time.Now().UnixNano()))
		if err := c.voiceRecorder.EncodeOpus(tmpPath); err != nil {
			log.Printf("encoding voice preview: %v", err)
			return
		}
		c.voicePreviewPath = tmpPath
	}

	c.voiceIsPlaying = true
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetIconName("media-playback-pause-symbolic")
	}

	playAudioPreview(c.voicePreviewPath, func(playing bool) {
		c.voiceIsPlaying = playing
		if !playing {
			if c.voicePlayBtn != nil {
				c.voicePlayBtn.SetIconName("media-playback-start-symbolic")
			}
		}
	})
}

// stopVoicePreview stops playback of the recorded voice note preview.
func (c *whatsDownMainController) stopVoicePreview() {
	if c.voiceIsPlaying {
		stopCurrentAudio()
		c.voiceIsPlaying = false
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetIconName("media-playback-start-symbolic")
	}
	if c.voicePreviewPath != "" {
		_ = os.Remove(c.voicePreviewPath)
		c.voicePreviewPath = ""
	}
}

// sendVoiceRecording stops the recorder, encodes to Opus, and dispatches the voice message.
func (c *whatsDownMainController) sendVoiceRecording() {
	if c.voiceRecorder == nil || c.currentChat == nil {
		c.deleteVoiceRecording()
		return
	}

	c.voiceRecorder.Stop()
	c.stopVoicePreview()

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	dur := c.voiceRecorder.Duration()
	if dur < 500*time.Millisecond {
		c.deleteVoiceRecording()
		return
	}

	waveform := c.voiceRecorder.Waveform()
	rec := c.voiceRecorder
	c.voiceRecorder = nil

	if c.chatInputStack != nil {
		c.chatInputStack.SetVisibleChildName("normal")
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("voice_msg_%d.ogg", time.Now().UnixNano()))
	go func() {
		if err := rec.EncodeOpus(tmpFile); err != nil {
			log.Printf("encoding voice message: %v", err)
			return
		}
		glib.IdleAdd(func() {
			c.sendVoiceMessage(tmpFile, dur, waveform)
		})
	}()
}

// sendVoiceMessage uploads and sends an Opus voice message (PTT).
func (c *whatsDownMainController) sendVoiceMessage(filePath string, duration time.Duration, waveform []byte) {
	if c.currentChat == nil {
		_ = os.Remove(filePath)
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	data, err := os.ReadFile(filePath)
	_ = os.Remove(filePath)
	if err != nil {
		log.Printf("reading voice file: %v", err)
		return
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	seconds := uint32(duration.Seconds())
	if seconds == 0 {
		seconds = 1
	}

	mimeType := "audio/ogg; codecs=opus"

	go func() {
		ctx := context.Background()
		var msgID string
		var rawProto []byte

		if waClient != nil {
			targetJID, err := types.ParseJID(chatJIDStr)
			if err != nil {
				log.Printf("parsing JID for voice send: %v", err)
				return
			}

			uploadResp, err := waClient.Upload(ctx, data, whatsmeow.MediaAudio)
			if err != nil {
				log.Printf("uploading voice message to WhatsApp: %v", err)
				return
			}

			fileLength := uint64(len(data))
			audMsg := &waE2E.AudioMessage{
				Mimetype:      proto.String(mimeType),
				URL:           &uploadResp.URL,
				DirectPath:    &uploadResp.DirectPath,
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    &fileLength,
				Seconds:       &seconds,
				PTT:           proto.Bool(true),
				Waveform:      waveform,
				ContextInfo:   ctxInfo,
			}
			waMsg := waE2E.Message{AudioMessage: audMsg}

			resp, err := waClient.SendMessage(ctx, targetJID, &waMsg)
			if err != nil {
				log.Printf("sending voice message: %v", err)
				return
			}
			msgID = resp.ID
			rawProto, _ = proto.Marshal(&waMsg)
		} else {
			msgID = fmt.Sprintf("dev_voice_%d", time.Now().UnixNano())
			audMsg := &waE2E.AudioMessage{
				Mimetype:    proto.String(mimeType),
				Seconds:     &seconds,
				PTT:         proto.Bool(true),
				Waveform:    waveform,
				ContextInfo: ctxInfo,
			}
			waMsg := waE2E.Message{AudioMessage: audMsg}
			rawProto, _ = proto.Marshal(&waMsg)
		}

		initStatus := MessageStatusSent
		if c.isSelfJID(chatJIDStr) {
			initStatus = MessageStatusRead
		}
		sentMsg := StoredMessage{
			ChatJID:     chatJIDStr,
			ID:          msgID,
			SenderJID:   "",
			SenderName:  "You",
			FromMe:      true,
			Timestamp:   time.Now(),
			Kind:        messageKindVoice,
			Text:        "",
			Raw:         rawProto,
			Status:      initStatus,
			SenderSaved: true,
		}

		destPath := getLocalMediaPath(sentMsg)
		_ = os.WriteFile(destPath, data, 0o600)

		if msgStore != nil {
			_, _ = msgStore.db.ExecContext(ctx, `
				INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (chat_jid, id) DO UPDATE SET status = MAX(whatsdown_messages.status, excluded.status)`,
				sentMsg.ChatJID, sentMsg.ID, sentMsg.SenderJID, "", 1, sentMsg.Timestamp.Unix(), sentMsg.Kind, sentMsg.Text, sentMsg.Raw, sentMsg.Status)

			_, _ = msgStore.db.ExecContext(ctx, `
				UPDATE whatsdown_chats SET last_message_at = ? WHERE jid = ?`,
				sentMsg.Timestamp.Unix(), sentMsg.ChatJID)

			glib.IdleAdd(func() {
				if c.currentChat != nil && c.currentChat.JID == chatJIDStr {
					c.currentMessages = append(c.currentMessages, sentMsg)
					bubble := c.createMessageBubble(sentMsg)
					if c.messageRowMap != nil {
						c.messageRowMap[sentMsg.ID] = bubble
					}
					c.messagesList.Append(bubble)
					c.scrollToBottomAnimated()
				}
				c.loadChats()
				c.loadArchivedChats()
			})
		}
	}()
}

// drawVoiceMeter renders the audio level meter bars into the recording bar drawing area.
func (c *whatsDownMainController) drawVoiceMeter(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}

	barWidth := 3.0
	barGap := 2.0
	step := barWidth + barGap
	totalBars := int(float64(width) / step)
	if totalBars <= 0 {
		return
	}

	var levels []float64
	if c.voiceRecorder != nil {
		levels = c.voiceRecorder.Levels()
	}

	isRecording := c.voiceRecorder != nil && !c.voiceRecorder.stopped
	isPaused := c.voiceRecorder != nil && c.voiceRecorder.Paused()

	r, g, b, a := 0.0, 0.66, 0.52, 0.95
	if isPaused {
		a = 0.5
	} else if !isRecording {
		r, g, b, a = 0.2, 0.6, 0.86, 0.85
	}

	cr.SetSourceRGBA(r, g, b, a)

	if isRecording {
		startIdx := len(levels) - totalBars
		for i := 0; i < totalBars; i++ {
			idx := startIdx + i
			lvl := 0.08
			if idx >= 0 && idx < len(levels) {
				lvl = math.Max(0.08, math.Min(1.0, levels[idx]))
			}
			h := float64(height) * lvl
			x := float64(i) * step
			y := (float64(height) - h) / 2.0
			drawRoundedBar(cr, x, y, barWidth, h)
		}
	} else if len(levels) > 0 {
		waveformLevels := resampleLevels(levels, totalBars)
		peak := 0.0
		for _, v := range waveformLevels {
			peak = math.Max(peak, v)
		}
		for i := 0; i < totalBars; i++ {
			lvl := 0.08
			if peak > 0 && i < len(waveformLevels) {
				lvl = math.Max(0.08, math.Min(1.0, waveformLevels[i]/peak))
			}
			h := float64(height) * lvl
			x := float64(i) * step
			y := (float64(height) - h) / 2.0
			drawRoundedBar(cr, x, y, barWidth, h)
		}
	} else {
		for i := 0; i < totalBars; i++ {
			h := float64(height) * 0.08
			x := float64(i) * step
			y := (float64(height) - h) / 2.0
			drawRoundedBar(cr, x, y, barWidth, h)
		}
	}
}

func drawRoundedBar(cr *cairo.Context, x, y, w, h float64) {
	radius := w / 2.0
	if h < radius*2 {
		cr.Rectangle(x, y, w, h)
		cr.Fill()
		return
	}
	cr.NewPath()
	cr.Arc(x+radius, y+radius, radius, math.Pi, 0)
	cr.Arc(x+radius, y+h-radius, radius, 0, math.Pi)
	cr.ClosePath()
	cr.Fill()
}

// showCreatePollDialog opens a modal dialog allowing the user to create and send a poll.
func (c *whatsDownMainController) showCreatePollDialog() {
	if c.currentChat == nil {
		return
	}

	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(440, 520)
	dialog.SetTitle("Create Poll")

	headerBar := adw.NewHeaderBar()
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("flat")
	cancelBtn.ConnectClicked(func() {
		dialog.Close()
	})
	headerBar.PackStart(cancelBtn)

	createBtn := gtk.NewButtonWithLabel("Create")
	createBtn.AddCSSClass("suggested-action")
	headerBar.PackEnd(createBtn)

	dialog.SetTitlebar(headerBar)

	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 16)
	mainBox.SetMarginTop(16)
	mainBox.SetMarginBottom(20)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	// 1. Question Group
	questionGroup := adw.NewPreferencesGroup()
	questionGroup.SetTitle("Question")

	qRow := adw.NewActionRow()
	qRow.SetTitle("Question")
	questionEntry := gtk.NewEntry()
	questionEntry.SetPlaceholderText("Ask a question…")
	questionEntry.SetHExpand(true)
	qRow.AddSuffix(questionEntry)
	qRow.SetActivatableWidget(questionEntry)
	questionGroup.Add(qRow)
	mainBox.Append(questionGroup)

	// 2. Options Group
	optionsGroup := adw.NewPreferencesGroup()
	optionsGroup.SetTitle("Options")

	var optionEntries []*gtk.Entry

	addOptionRow := func(placeholder string) {
		row := adw.NewActionRow()
		row.SetTitle(placeholder)
		entry := gtk.NewEntry()
		entry.SetPlaceholderText("Add option…")
		entry.SetHExpand(true)
		row.AddSuffix(entry)
		row.SetActivatableWidget(entry)
		optionsGroup.Add(row)
		optionEntries = append(optionEntries, entry)
	}

	addOptionRow("Option 1")
	addOptionRow("Option 2")
	mainBox.Append(optionsGroup)

	addBtn := gtk.NewButtonWithLabel("+ Add Option")
	addBtn.AddCSSClass("flat")
	addBtn.AddCSSClass("pill")
	addBtn.SetHAlign(gtk.AlignStart)
	addBtn.ConnectClicked(func() {
		if len(optionEntries) < 12 {
			idx := len(optionEntries) + 1
			addOptionRow(fmt.Sprintf("Option %d", idx))
			if len(optionEntries) >= 12 {
				addBtn.SetSensitive(false)
			}
		}
	})
	mainBox.Append(addBtn)

	// 3. Settings Group
	settingsGroup := adw.NewPreferencesGroup()
	settingsGroup.SetTitle("Settings")

	multipleRow := adw.NewActionRow()
	multipleRow.SetTitle("Allow multiple answers")
	multipleRow.SetSubtitle("Voters can pick more than one option")

	multipleSwitch := gtk.NewSwitch()
	multipleSwitch.SetVAlign(gtk.AlignCenter)
	multipleRow.AddSuffix(multipleSwitch)
	multipleRow.SetActivatableWidget(multipleSwitch)
	settingsGroup.Add(multipleRow)
	mainBox.Append(settingsGroup)

	// Error label
	errorLabel := gtk.NewLabel("")
	errorLabel.AddCSSClass("caption")
	errorLabel.AddCSSClass("error-label")
	errorLabel.SetHAlign(gtk.AlignCenter)
	errorLabel.SetVisible(false)
	mainBox.Append(errorLabel)

	scrolledWindow.SetChild(mainBox)
	dialog.SetChild(scrolledWindow)

	createBtn.ConnectClicked(func() {
		q := strings.TrimSpace(questionEntry.Text())
		if q == "" {
			errorLabel.SetText("Please enter a question.")
			errorLabel.SetVisible(true)
			return
		}

		var opts []string
		for _, e := range optionEntries {
			txt := strings.TrimSpace(e.Text())
			if txt != "" {
				opts = append(opts, txt)
			}
		}

		if len(opts) < 2 {
			errorLabel.SetText("Please enter at least 2 options.")
			errorLabel.SetVisible(true)
			return
		}

		allowMultiple := multipleSwitch.Active()
		dialog.Close()
		c.sendPoll(q, opts, allowMultiple)
	})

	dialog.Present()
}

// sendPoll builds, sends, and stores a WhatsApp poll.
func (c *whatsDownMainController) sendPoll(question string, options []string, allowMultiple bool) {
	if c.currentChat == nil {
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	selectableCount := 1
	if allowMultiple {
		selectableCount = len(options)
	}

	go func() {
		ctx := context.Background()
		var msgID string
		var rawProto []byte

		if waClient != nil {
			targetJID, err := types.ParseJID(chatJIDStr)
			if err != nil {
				log.Printf("parsing JID for poll send: %v", err)
				return
			}
			waMsg := waClient.BuildPollCreation(question, options, selectableCount)
			if ctxInfo != nil && waMsg != nil && waMsg.PollCreationMessage != nil {
				waMsg.PollCreationMessage.ContextInfo = ctxInfo
			}
			resp, err := waClient.SendMessage(ctx, targetJID, waMsg)
			if err != nil {
				log.Printf("sending poll message: %v", err)
				return
			}
			msgID = resp.ID
			rawProto, _ = proto.Marshal(waMsg)
		} else {
			// Dev mode fallback
			msgID = fmt.Sprintf("dev_poll_%d", time.Now().UnixNano())
			var optProtos []*waE2E.PollCreationMessage_Option
			for _, opt := range options {
				optProtos = append(optProtos, &waE2E.PollCreationMessage_Option{OptionName: proto.String(opt)})
			}
			waMsg := &waE2E.Message{
				PollCreationMessage: &waE2E.PollCreationMessage{
					Name:                   proto.String(question),
					Options:                optProtos,
					SelectableOptionsCount: proto.Uint32(uint32(selectableCount)),
					ContextInfo:            ctxInfo,
				},
			}
			rawProto, _ = proto.Marshal(waMsg)
		}

		var ownJID string
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			ownJID = waClient.Store.ID.ToNonAD().String()
		}

		initStatus := MessageStatusSent
		if c.isSelfJID(chatJIDStr) {
			initStatus = MessageStatusRead
		}
		sentMsg := StoredMessage{
			ChatJID:     chatJIDStr,
			ID:          msgID,
			SenderJID:   ownJID,
			SenderName:  "You",
			FromMe:      true,
			Timestamp:   time.Now(),
			Kind:        messageKindPoll,
			Text:        question,
			Raw:         rawProto,
			Status:      initStatus,
			SenderSaved: true,
		}

		if msgStore != nil {
			_, _ = msgStore.db.ExecContext(ctx, `
				INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (chat_jid, id) DO UPDATE SET status = MAX(whatsdown_messages.status, excluded.status)`,
				sentMsg.ChatJID, sentMsg.ID, sentMsg.SenderJID, "", 1, sentMsg.Timestamp.Unix(), sentMsg.Kind, sentMsg.Text, sentMsg.Raw, sentMsg.Status)

			_, _ = msgStore.db.ExecContext(ctx, `
				UPDATE whatsdown_chats SET last_message_at = ? WHERE jid = ?`,
				sentMsg.Timestamp.Unix(), sentMsg.ChatJID)

			glib.IdleAdd(func() {
				if c.currentChat != nil && c.currentChat.JID == chatJIDStr {
					c.currentMessages = append(c.currentMessages, sentMsg)
					bubble := c.createMessageBubble(sentMsg)
					if c.messageRowMap != nil {
						c.messageRowMap[sentMsg.ID] = bubble
					}
					c.messagesList.Append(bubble)
					c.scrollToBottomAnimated()
				}
				c.loadChats()
				c.loadArchivedChats()
			})
		}
	}()
}

type pollOptWidget struct {
	btn         *gtk.Button
	indicator   *gtk.Image
	countLbl    *gtk.Label
	progressBar *gtk.ProgressBar
}

// reloadPollBubble refreshes the messages list if the currently viewed chat matches.
func (c *whatsDownMainController) reloadPollBubble(pollID string) {
	if c.currentChat == nil {
		return
	}
	c.loadMessages(c.currentChat.JID)
}

// showPollVotesDialog opens a dialog displaying who voted for each option in a poll.
func (c *whatsDownMainController) showPollVotesDialog(msg StoredMessage, question string, options []string) {
	dialog := adw.NewDialog()
	dialog.SetTitle("Poll Details")
	dialog.SetContentWidth(420)
	dialog.SetContentHeight(500)

	toolbarView := adw.NewToolbarView()
	headerBar := adw.NewHeaderBar()
	toolbarView.AddTopBar(headerBar)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 16)
	contentBox.SetMarginTop(18)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(18)
	contentBox.SetMarginEnd(18)

	qLabel := gtk.NewLabel(question)
	qLabel.AddCSSClass("title-2")
	qLabel.SetWrap(true)
	qLabel.SetWrapMode(pango.WrapWordChar)
	qLabel.SetXAlign(0)
	contentBox.Append(qLabel)

	if msgStore != nil {
		results, _ := msgStore.getPollResults(context.Background(), msg.ChatJID, msg.ID)
		ownJID := ""
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			ownJID = waClient.Store.ID.ToNonAD().String()
		}

		for _, opt := range options {
			group := adw.NewPreferencesGroup()
			cnt := 0
			var voters []string
			if results != nil {
				cnt = results.VoteCounts[opt]
				voters = results.VotersByOption[opt]
			}
			suffix := "votes"
			if cnt == 1 {
				suffix = "vote"
			}
			group.SetTitle(fmt.Sprintf("%s (%d %s)", opt, cnt, suffix))

			if len(voters) == 0 {
				row := adw.NewActionRow()
				row.SetTitle("No votes yet")
				row.AddCSSClass("dim-label")
				group.Add(row)
			} else {
				for _, v := range voters {
					row := adw.NewActionRow()
					name := "You"
					if v != "me" && (ownJID == "" || v != ownJID) {
						resolved := c.resolveContactName(v)
						if resolved != "" && resolved != "WhatsApp User" {
							name = resolved
						} else {
							clean := cleanPhoneNumber(v)
							if clean != "" {
								name = clean
							} else {
								name = v
							}
						}
					}
					row.SetTitle(name)
					row.AddPrefix(gtk.NewImageFromIconName("avatar-default-symbolic"))
					group.Add(row)
				}
			}
			contentBox.Append(group)
		}
	}

	scrolled.SetChild(contentBox)
	toolbarView.SetContent(scrolled)
	dialog.SetChild(toolbarView)
	dialog.Present(c.window)
}

// handlePollVote toggles a vote on a poll option, persisting it locally and broadcasting
// via WhatsApp if connected, then updates the option widget states and results.
func (c *whatsDownMainController) handlePollVote(msg StoredMessage, optName string, allowMultiple bool, widgets map[string]*pollOptWidget, totalVotesLbl *gtk.Label) {
	if msgStore == nil {
		return
	}
	go func() {
		voterJID := "me"
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			voterJID = waClient.Store.ID.ToNonAD().String()
		}

		userVotes, err := msgStore.togglePollVote(context.Background(), msg.ChatJID, msg.ID, optName, allowMultiple, voterJID)
		if err != nil {
			log.Printf("toggling poll vote error: %v", err)
			return
		}
		if voterJID != "me" {
			var selectedList []string
			for name, sel := range userVotes {
				if sel {
					selectedList = append(selectedList, name)
				}
			}
			_ = msgStore.setPollVotesForVoter(context.Background(), msg.ChatJID, msg.ID, "me", selectedList)
		}

		if waClient != nil && waClient.IsConnected() {
			ensurePollMessageSecret(context.Background(), msg)
			var selectedList []string
			for name, sel := range userVotes {
				if sel {
					selectedList = append(selectedList, name)
				}
			}
			chatJID, _ := types.ParseJID(msg.ChatJID)
			senderJID, _ := types.ParseJID(msg.SenderJID)
			if (senderJID.IsEmpty() || msg.FromMe || c.isSelfJID(msg.ChatJID)) && waClient.Store != nil && waClient.Store.ID != nil {
				senderJID = waClient.Store.ID.ToNonAD()
			}
			pollInfo := &types.MessageInfo{
				ID:        types.MessageID(msg.ID),
				Chat:      chatJID.ToNonAD(),
				Sender:    senderJID.ToNonAD(),
				IsFromMe:  msg.FromMe || c.isSelfJID(msg.ChatJID),
				IsGroup:   strings.HasSuffix(msg.ChatJID, "@g.us"),
				Timestamp: msg.Timestamp,
			}
			voteMsg, err := waClient.BuildPollVote(context.Background(), pollInfo, selectedList)
			if err != nil {
				log.Printf("building poll vote error: %v", err)
			} else if voteMsg != nil {
				if resp, err := waClient.SendMessage(context.Background(), chatJID, voteMsg); err != nil {
					log.Printf("sending poll vote error: %v", err)
				} else {
					log.Printf("poll vote successfully sent to WhatsApp: %v", resp.ID)
				}
			}
		}

		results, _ := msgStore.getPollResults(context.Background(), msg.ChatJID, msg.ID)

		glib.IdleAdd(func() {
			totalVoters := 0
			if results != nil {
				totalVoters = results.TotalVoters
			}
			for name, w := range widgets {
				isSelected := userVotes[name]
				if isSelected {
					w.btn.AddCSSClass("selected")
					if allowMultiple {
						w.indicator.SetFromIconName("checkbox-checked-symbolic")
					} else {
						w.indicator.SetFromIconName("radio-checked-symbolic")
					}
				} else {
					w.btn.RemoveCSSClass("selected")
					if allowMultiple {
						w.indicator.SetFromIconName("checkbox-symbolic")
					} else {
						w.indicator.SetFromIconName("radio-symbolic")
					}
				}

				cnt := 0
				if results != nil {
					cnt = results.VoteCounts[name]
				}
				var cText string
				if cnt == 1 {
					cText = "1 vote"
				} else if cnt > 1 {
					cText = fmt.Sprintf("%d votes", cnt)
				} else {
					cText = "0"
				}
				w.countLbl.SetText(cText)

				var frac float64
				if totalVoters > 0 {
					frac = float64(cnt) / float64(totalVoters)
				}
				w.progressBar.SetFraction(frac)
			}
			if totalVotesLbl != nil {
				if totalVoters == 0 {
					totalVotesLbl.SetText("No votes yet")
				} else if totalVoters == 1 {
					totalVotesLbl.SetText("1 person voted")
				} else {
					totalVotesLbl.SetText(fmt.Sprintf("%d people voted", totalVoters))
				}
			}
		})
	}()
}

// openPDFPreview downloads the PDF if needed and presents the in-app PDF preview dialog.
func (c *whatsDownMainController) openPDFPreview(msg StoredMessage, docName string) {
	go func() {
		path, err := downloadMedia(context.Background(), msg)
		if err != nil {
			log.Printf("downloading PDF for preview error: %v", err)
			return
		}
		glib.IdleAdd(func() {
			c.showPDFViewerDialog(path, docName)
		})
	}()
}

// showPDFViewerDialog displays an in-app viewer for a PDF document with a page switcher.
func (c *whatsDownMainController) showPDFViewerDialog(filePath, docName string) {
	if filePath == "" {
		return
	}
	totalPages := getPDFPageCount(filePath)
	currentPage := 1

	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(600, 720)

	headerBar := gtk.NewHeaderBar()
	titleBox := gtk.NewBox(gtk.OrientationVertical, 0)
	titleLbl := gtk.NewLabel(docName)
	titleLbl.AddCSSClass("heading")
	titleLbl.SetEllipsize(pango.EllipsizeEnd)
	titleBox.Append(titleLbl)
	headerBar.SetTitleWidget(titleBox)

	openExtBtn := gtk.NewButton()
	openExtBtn.SetTooltipText("Open in Default App")
	openExtBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	openExtBox.Append(gtk.NewImageFromIconName("document-open-symbolic"))
	openExtBox.Append(gtk.NewLabel("Open"))
	openExtBtn.SetChild(openExtBox)
	openExtBtn.ConnectClicked(func() {
		_ = openFile(filePath)
	})
	headerBar.PackEnd(openExtBtn)
	dialog.SetTitlebar(headerBar)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 8)
	mainBox.SetMarginTop(12)
	mainBox.SetMarginBottom(12)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	scroll.SetVExpand(true)
	scroll.SetHExpand(true)

	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.SetHExpand(true)
	pic.SetVExpand(true)
	pic.AddCSSClass("pdf-preview-picture")
	scroll.SetChild(pic)
	mainBox.Append(scroll)

	switcherBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	switcherBox.SetHAlign(gtk.AlignCenter)
	switcherBox.SetMarginTop(8)
	switcherBox.AddCSSClass("pdf-page-switcher")

	prevBtn := gtk.NewButtonFromIconName("go-previous-symbolic")
	prevBtn.AddCSSClass("flat")
	prevBtn.AddCSSClass("circular")
	prevBtn.SetSensitive(false)
	prevBtn.SetTooltipText("Previous page")

	pageLbl := gtk.NewLabel(fmt.Sprintf("Page %d of %d", currentPage, totalPages))
	pageLbl.AddCSSClass("caption")

	nextBtn := gtk.NewButtonFromIconName("go-next-symbolic")
	nextBtn.AddCSSClass("flat")
	nextBtn.AddCSSClass("circular")
	nextBtn.SetSensitive(totalPages > 1)
	nextBtn.SetTooltipText("Next page")

	updatePage := func(newPage int) {
		if newPage < 1 || newPage > totalPages {
			return
		}
		currentPage = newPage
		pageLbl.SetText(fmt.Sprintf("Page %d of %d", currentPage, totalPages))
		prevBtn.SetSensitive(currentPage > 1)
		nextBtn.SetSensitive(currentPage < totalPages)
		go func() {
			tex := getPDFPageTexture(filePath, currentPage)
			if tex != nil {
				glib.IdleAdd(func() {
					pic.SetPaintable(tex)
				})
			}
		}()
	}

	prevBtn.ConnectClicked(func() {
		updatePage(currentPage - 1)
	})
	nextBtn.ConnectClicked(func() {
		updatePage(currentPage + 1)
	})

	switcherBox.Append(prevBtn)
	switcherBox.Append(pageLbl)
	switcherBox.Append(nextBtn)
	mainBox.Append(switcherBox)

	dialog.SetChild(mainBox)

	// Load initial page
	if tex := getPDFPageTexture(filePath, 1); tex != nil {
		pic.SetPaintable(tex)
	} else {
		go func() {
			tex := getPDFPageTexture(filePath, 1)
			if tex != nil {
				glib.IdleAdd(func() {
					pic.SetPaintable(tex)
				})
			}
		}()
	}

	dialog.Present()
}

// dispatchDesktopNotification dispatches a system desktop notification.
func dispatchDesktopNotification(title, body string) {
	glib.IdleAdd(func() {
		if application != nil {
			notif := gio.NewNotification(title)
			notif.SetBody(body)
			notif.SetIcon(gio.NewThemedIcon("chat-message-new-symbolic"))
			application.SendNotification("", notif)
			return
		}

		go func() {
			if _, err := exec.LookPath("notify-send"); err == nil {
				_ = exec.Command("notify-send",
					"-a", "WhatsDown",
					"-i", "chat-message-new-symbolic",
					title, body,
				).Run()
			}
		}()
	})
}

// formatNotificationContent extracts a user-friendly sender title and message preview for notifications.
func (c *whatsDownMainController) formatNotificationContent(evt *events.Message) (string, string) {
	chatStr := evt.Info.Chat.ToNonAD().String()
	senderStr := evt.Info.Sender.ToNonAD().String()

	senderName := c.resolveContactName(senderStr)
	if senderName == "" || senderName == senderStr {
		if evt.Info.PushName != "" {
			senderName = evt.Info.PushName
		} else {
			senderName = cleanPhoneNumber(senderStr)
		}
	}
	if senderName == "" {
		senderName = "WhatsApp User"
	}

	var title string
	if evt.Info.IsGroup {
		chatName := c.resolveContactName(chatStr)
		if chatName == "" || chatName == chatStr {
			chatName = "Group"
		}
		title = fmt.Sprintf("%s (%s)", senderName, chatName)
	} else {
		title = senderName
	}

	body := "New message"
	if evt.Message != nil {
		kind, text, ok := describeMessage(evt.Message)
		if ok && text != "" {
			switch kind {
			case messageKindPoll:
				body = "📊 Poll: " + text
			case messageKindImage:
				body = "📷 Photo"
				if text != "" && text != "photo" {
					body += ": " + text
				}
			case messageKindVideo:
				body = "🎥 Video"
				if text != "" && text != "video" {
					body += ": " + text
				}
			case messageKindAudio, messageKindVoice:
				body = "🎵 Audio message"
			case messageKindDocument:
				body = "📄 " + text
			default:
				body = text
			}
		}
	}

	return title, body
}

// handleIncomingLiveMessage updates the UI when a live message arrives
// and dispatches a desktop notification.
func (c *whatsDownMainController) handleIncomingLiveMessage(evt *events.Message) {
	if evt.Message != nil && evt.Message.GetPollUpdateMessage() != nil {
		return
	}

	chatStr := evt.Info.Chat.ToNonAD().String()
	isSelfChat := c.isSelfJID(chatStr)
	if isSelfChat {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatStr = canonical
		}
	}
	isCurrentChat := c.currentChat != nil && (c.currentChat.JID == chatStr || (c.isSelfJID(c.currentChat.JID) && isSelfChat))

	if isCurrentChat {
		var newMsg *StoredMessage
		if msgStore != nil {
			newMsg, _ = msgStore.getMessage(context.Background(), chatStr, evt.Info.ID)
			if newMsg == nil {
				newMsg, _ = msgStore.getMessageByID(context.Background(), evt.Info.ID)
			}
		}

		if newMsg != nil {
			if _, exists := c.messageRowMap[newMsg.ID]; !exists {
				c.currentMessages = append(c.currentMessages, *newMsg)
				bubble := c.createMessageBubble(*newMsg)
				if c.messageRowMap != nil {
					c.messageRowMap[newMsg.ID] = bubble
				}
				c.messagesList.Append(bubble)
				c.scrollToBottomAnimated()
			}
		} else {
			c.loadMessages(c.currentChat.JID)
		}
		c.markChatRead(chatStr)
	}

	c.loadChats()
	c.loadArchivedChats()

	if !evt.Info.IsFromMe && !isSelfChat {
		isMuted := false
		if msgStore != nil {
			if ch, err := msgStore.getChatOrContact(context.Background(), chatStr); err == nil {
				isMuted = ch.Muted
			}
		}
		if !isMuted {
			title, body := c.formatNotificationContent(evt)
			dispatchDesktopNotification(title, body)
		}
	}
}

// applyStatusToLabel sets the checkmark text and CSS classes based on message delivery and read status.
func (c *whatsDownMainController) applyStatusToLabel(lbl *gtk.Label, status int) {
	if lbl == nil {
		return
	}
	lbl.RemoveCSSClass("chat-bubble-status-sent")
	lbl.RemoveCSSClass("chat-bubble-status-delivered")
	lbl.RemoveCSSClass("chat-bubble-status-read")

	switch status {
	case MessageStatusRead:
		lbl.SetText("✓✓")
		lbl.AddCSSClass("chat-bubble-status-read")
	case MessageStatusDelivered:
		lbl.SetText("✓✓")
		lbl.AddCSSClass("chat-bubble-status-delivered")
	case MessageStatusSent:
		lbl.SetText("✓")
		lbl.AddCSSClass("chat-bubble-status-sent")
	default:
		lbl.SetText("✓✓")
		lbl.AddCSSClass("chat-bubble-status-read")
	}
}

// handleReceiptUpdate updates the read/delivery ticks on outgoing messages.
func (c *whatsDownMainController) handleReceiptUpdate(ids []string, status int) {
	for _, id := range ids {
		if lbl, ok := c.messageStatusLabels[id]; ok && lbl != nil {
			c.applyStatusToLabel(lbl, status)
		}
		for i := range c.currentMessages {
			if c.currentMessages[i].ID == id {
				if status > c.currentMessages[i].Status {
					c.currentMessages[i].Status = status
				}
				break
			}
		}
	}
}

// insertEmoji inserts the selected emoji into the message entry at the current cursor position.
func (c *whatsDownMainController) insertEmoji(text string) {
	if c.messageEntry == nil {
		return
	}
	runes := []rune(c.messageEntry.Text())
	pos := c.messageEntry.Position()
	if pos < 0 || pos > len(runes) {
		pos = len(runes)
	}
	newRunes := append(runes[:pos], append([]rune(text), runes[pos:]...)...)
	c.messageEntry.SetText(string(newRunes))
	c.messageEntry.SetPosition(pos + len([]rune(text)))
	c.messageEntry.GrabFocus()
}

// formatStatusMarkup returns Pango markup representing delivery and read ticks.
// Uses vertically centered Japanese periods (・) for crisp vertical alignment.
func formatStatusMarkup(status int) string {
	switch status {
	case MessageStatusRead:
		return `<span foreground="#53bdeb" font_weight="bold">・・</span>`
	case MessageStatusDelivered:
		return `<span alpha="85%">・・</span>`
	case MessageStatusSent:
		return `<span alpha="70%">・</span>`
	default:
		return `<span foreground="#53bdeb" font_weight="bold">・・</span>`
	}
}

// formatChatTime formats a timestamp for the chat list row.
func formatChatTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Mon")
	}
	return t.Format("02/01/06")
}

// cleanPhoneNumber extracts and formats a phone number cleanly with a leading '+',
// stripping any device indices (:device) and domain suffixes (@s.whatsapp.net, etc.).
// For WhatsApp LID identifiers (@lid), it does not prepend '+' since LIDs are internal
// identifiers, returning "WhatsApp User" if not resolved to a real phone number.
func cleanPhoneNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "@lid") {
		return "WhatsApp User"
	}
	if idx := strings.Index(raw, "@"); idx != -1 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, ":"); idx != -1 {
		raw = raw[:idx]
	}
	if raw == "" {
		return ""
	}
	// Check if the identifier consists purely of digits
	allDigits := true
	for _, r := range raw {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits && len(raw) >= 5 {
		if len(raw) == 15 && strings.HasPrefix(raw, "1") {
			return "WhatsApp User"
		}
		return "+" + raw
	}
	return raw
}

// formatDisplayName returns the user-facing title for a chat, prioritizing
// contact names as saved on the phone, clean international phone numbers for unsaved
// contacts, and friendly fallback labels without exposing internal WhatsApp JID strings.
func formatDisplayName(chat Chat) string {
	if activeMainWindow != nil && activeMainWindow.isSelfJID(chat.JID) {
		name := strings.TrimSpace(chat.Name)
		if name != "" && !strings.Contains(name, "@") && !strings.Contains(name, "∙∙∙") {
			return name + " (You)"
		}
		return "Message yourself"
	}

	name := strings.TrimSpace(chat.Name)

	// If name is saved and not a raw JID or redacted placeholder
	if name != "" && !strings.Contains(name, "@") && !strings.Contains(name, "∙∙∙") {
		return name
	}

	// For standard direct messages (@s.whatsapp.net)
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		return cleanPhoneNumber(chat.JID)
	}

	// For LID direct messages (@lid)
	if strings.HasSuffix(chat.JID, "@lid") {
		if strings.HasPrefix(name, "+") {
			return name
		}
		return "WhatsApp User"
	}

	// For Group chats (@g.us)
	if strings.HasSuffix(chat.JID, "@g.us") {
		if name != "" && !strings.Contains(name, "@") {
			return name
		}
		return "Group"
	}

	// For Status / Broadcasts
	if strings.HasSuffix(chat.JID, "@broadcast") {
		return "Status Updates"
	}

	// For Channels (@newsletter)
	if strings.HasSuffix(chat.JID, "@newsletter") {
		if name != "" && !strings.Contains(name, "@") {
			return name
		}
		return "Channel"
	}

	if name != "" && !strings.Contains(name, "@") {
		return name
	}
	return cleanPhoneNumber(chat.JID)
}

// formatMessagePreview formats a message into a concise single-line preview snippet.
func formatMessagePreview(kind, text string) string {
	cleanText := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	switch kind {
	case messageKindImage:
		if cleanText != "" {
			return "📷 " + cleanText
		}
		return "📷 Photo"
	case messageKindVideo:
		if cleanText != "" {
			return "📹 " + cleanText
		}
		return "📹 Video"
	case messageKindAudio:
		return "🎵 Audio"
	case messageKindVoice:
		return "🎤 Voice message"
	case messageKindDocument:
		if cleanText != "" {
			return "📄 " + cleanText
		}
		return "📄 Document"
	case messageKindSticker:
		return "Sticker"
	case messageKindLocation:
		if cleanText != "" {
			return "📍 " + cleanText
		}
		return "📍 Location"
	case messageKindContact:
		if cleanText != "" {
			return "👤 " + cleanText
		}
		return "👤 Contact"
	case messageKindPoll:
		if cleanText != "" {
			return "📊 " + cleanText
		}
		return "📊 Poll"
	case messageKindDeleted:
		return "🚫 This message was deleted"
	default:
		if cleanText != "" {
			return cleanText
		}
		return "Message"
	}
}

// formatChatSubtitle returns the secondary line for a chat row or dialog.
// It displays the last message sent in that chat below the user/group name.
func formatChatSubtitle(chat Chat) string {
	if chat.LastMessageKind != "" || chat.LastMessageText != "" {
		preview := formatMessagePreview(chat.LastMessageKind, chat.LastMessageText)
		isGroup := strings.HasSuffix(chat.JID, "@g.us")

		if isGroup {
			if chat.LastMessageFromMe {
				return "You: " + preview
			}
			sender := chat.LastMessageSender
			if sender != "" && !strings.Contains(sender, "@") {
				return sender + ": " + preview
			}
			return preview
		}

		// 1-on-1 DM:
		if chat.LastMessageFromMe {
			return "You: " + preview
		}
		return preview
	}

	displayName := formatDisplayName(chat)

	if strings.HasSuffix(chat.JID, "@g.us") {
		return "Group"
	}
	if strings.HasSuffix(chat.JID, "@broadcast") {
		return "Status"
	}
	if strings.HasSuffix(chat.JID, "@newsletter") {
		return "Channel"
	}

	// For phone DMs: show phone number if different from display name
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		phone := cleanPhoneNumber(chat.JID)
		if phone != "" && phone != displayName {
			return phone
		}
		return ""
	}

	// For LID DMs: show phone number if resolved from LID mapping
	if strings.HasSuffix(chat.JID, "@lid") {
		if strings.HasPrefix(chat.Name, "+") && chat.Name != displayName {
			return chat.Name
		}
		return ""
	}

	return ""
}

// formatTopbarSubtitle returns the secondary line for the active conversation header bar.
func formatTopbarSubtitle(chat Chat) string {
	displayName := formatDisplayName(chat)

	if strings.HasSuffix(chat.JID, "@g.us") {
		return "Group"
	}
	if strings.HasSuffix(chat.JID, "@broadcast") {
		return "Status"
	}
	if strings.HasSuffix(chat.JID, "@newsletter") {
		return "Channel"
	}

	// For phone DMs: show phone number if different from display name
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		phone := cleanPhoneNumber(chat.JID)
		if phone != "" && phone != displayName {
			return phone
		}
		return ""
	}

	// For LID DMs: show phone number if resolved from LID mapping
	if strings.HasSuffix(chat.JID, "@lid") {
		if strings.HasPrefix(chat.Name, "+") && chat.Name != displayName {
			return chat.Name
		}
		return ""
	}

	return ""
}

// isSelfJID checks whether the given JID represents the logged-in user.
func (c *whatsDownMainController) isSelfJID(jid string) bool {
	if jid == "" {
		return false
	}
	if msgStore != nil {
		return msgStore.isSelfJID(context.Background(), jid)
	}
	if waClient != nil && waClient.Store != nil {
		parsed, err := types.ParseJID(jid)
		if err == nil {
			nonAD := parsed.ToNonAD().String()
			user := parsed.User
			if waClient.Store.ID != nil {
				ownPN := waClient.Store.ID.ToNonAD().String()
				if nonAD == ownPN || user == waClient.Store.ID.User {
					return true
				}
			}
			if !waClient.Store.LID.IsEmpty() {
				ownLID := waClient.Store.LID.ToNonAD().String()
				if nonAD == ownLID || user == waClient.Store.LID.User {
					return true
				}
			}
		}
	}
	return false
}

// getCanonicalSelfJID returns the primary JID for the logged-in user.
func (c *whatsDownMainController) getCanonicalSelfJID() string {
	if msgStore != nil {
		if canonical := msgStore.getCanonicalSelfJID(context.Background()); canonical != "" {
			return canonical
		}
	}
	if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
		return waClient.Store.ID.ToNonAD().String()
	}
	return ""
}
