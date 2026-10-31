# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.0.0] - 2026-10-31

### Added
- Large 64pt single-emoji display in chat bubbles for messages containing only a single emoji.
- Desktop notification audio alerts using the system sound theme (`message-new-instant`).
- Window focusing and unminimizing on notification activation, bringing the app above other windows.
- Message scrolling and visual highlight when navigating to a chat via notification click.
- Automatic withdrawal of desktop notifications when reading a chat or using Mark All as Read.
- Sender avatars in direct-message notifications and group avatars in group notifications.
- Coloured chat folders with avatar clusters, multi-chat selection, drag-and-drop organization, and an "Add to Folder" context action.
- Message selection mode, forwarding to multiple recipients, deletion, and per-recipient Message Info.
- Last message sender display in group chat previews.
- In-window photo, video, and PDF viewers with zoom, rotation, page navigation, and reply/forward/locate actions.
- Voice message seeking, pause/resume, and pitch-preserving playback speed control (0.5x, 1x, 1.5x, 2x).
- Keyboard shortcuts for chat navigation, text formatting, editing the last message, sending files, and shortcut help dialog.
- "Mark all as read" button in Chats and About dialog in Settings.
- Clickable message URLs and day separators labelled Today, Yesterday, or the date.

### Fixed
- Fixed "Mark all as read" failure caused by WhatsApp app-state snapshot LTHash verification mismatch on `regular_low` patches.
- Fixed app-state synchronization conflict handling (HTTP 409) with automatic patch catch-up retries.
- Fixed read receipts sent from primary phone being ignored, enabling real-time and offline catch-up read synchronization.
- Fixed desktop notifications not opening or focusing the app when clicked.
- Fixed emoji autocomplete leaving shortcode text behind when messages contained emojis or accented characters.
- Fixed read and delivery status synchronization across linked devices and group participants.
- Fixed excessive memory usage when opening chats, viewing media, and syncing with WhatsApp.
- Fixed typing stalls when updating the message composer and loading mention suggestions.
- Fixed conversation jumping to the bottom when selecting a message to reply to.
- Fixed voice playback controls triggering replies and improved visibility of read ticks and outgoing poll results.
- Fixed Markdown rendering crash when typing a number followed by a space and a letter in the message box.

## [0.2.0] - 2026-10-22

### Added
- Custom chat backgrounds: set any chat background independently to a solid colour or image wallpaper.
- Window tinting: optional customization that tints the titlebar and message input with the chat background colour.
- Colour Handles: coloured sidebar indicator stripes matching each chat's custom colour.
- Group creator dialog for creating group chats directly from WhatsDown.
- Group settings modal for moderating group permissions, changing settings, and managing participants.
- @mention autocomplete in the composer with fuzzy search across participants.
- Clickable and interactive @mention rendering in message bubbles.
- Demo Mode for offline evaluation and UI testing.

### Fixed
- Fixed chat scroll locking for up to 1500ms on long conversations.
- Fixed duplicate chats appearing in the sidebar chat list.
- Fixed community avatar rendering and alignment in the chat list.
- Fixed unnecessary re-rendering of avatars on view updates.
- Fixed titlebar avatar failing to update when switching between chats.

## [0.1.0] - 2026-10-21

### Added
- Initial release of WhatsDown for Ubuntu / Linux desktops.
