# WhatsDown

WhatsDown is a native Linux WhatsApp client built with **Go**, **GTK 4**, and **Libadwaita** using [whatsmeow](https://github.com/tulir/whatsmeow) and [gotk4](https://github.com/diamondburned/gotk4).

## Features

- **Native GNOME Experience:** Modern interface built with Libadwaita and GTK 4, following GNOME design guidelines.
- **QR Code Pairing:** Clean QR login interface with instant visual feedback.
- **Syncing Progress:** Dedicated syncing screen tracking initial history sync chunks (conversations & messages) as well as offline catches upon reconnect.
- **Main Chat Interface (`WhatsDownMain`):** Full Libadwaita adaptive split-view interface:
  - **Left-hand Navigation Menu:** Modern navigation rail allowing one-click switching between active **Chats** (default), **Calls**, **Archived** conversations, and **Settings**.
  - **WhatsApp Community Grouping:**
    - Groups belonging to a WhatsApp Community are automatically organized together under an expandable community folder.
    - Community folders appear among other chats sorted by their latest activity, displaying the community avatar, title, chat count, and aggregated unread count badge.
    - Clicking a community expands or collapses it inline (like a Discord server group), showing its member chats indented with tree branch styling.
    - Community member chats do not appear in the top-level chat list by themselves.
    - Searching chats automatically auto-expands any community whose child chats match the query.
  - **Calls Page & WhatsApp Call History:**
    - Dedicated **Calls** page in the navigation rail displaying your full WhatsApp call history.
    - Tracks incoming, outgoing, and missed voice and video calls with durations, caller avatars, resolved names/numbers, and relative timestamps.
    - Live sync of call events (`CallOffer`, `CallTerminate`, and `CallLogMessage` protobuf payloads).
    - One-click navigation to open the chat conversation with any caller, plus a context menu to remove entries from history.
  - **Voice Messages (PTT Recording & Playback):**
    - Built-in voice recorder with live audio capture and Opus/OGG encoding.
    - Interactive recording bar that replaces the message entry during recording, featuring live waveform/level meter, timer, pause/resume, stop, review playback, cancel/delete, and send actions.
  - **Clipboard Media Paste & Drag-and-Drop:**
    - Paste screenshots and clipboard images directly into the message box with `Ctrl+V` (or `Shift+Insert`) without needing to save them as files first.
    - Copy files from your file manager (Nautilus, Dolphin, etc.) and paste them directly into the conversation.
    - Drag and drop files from desktop or file managers directly onto the chat window.
    - Any text already typed in the message box is automatically preserved and pre-filled into the attachment's caption.
  - **Interactive Polls:**
    - Create single-choice or multi-choice polls with multiple options directly in conversations.
    - Live voting with real-time percentage progress bars, voter counts, and voter breakdown popovers.
  - **Stickers & Sticker Management:**
    - Native sticker picker popover with saved stickers gallery.
    - Right-click any sticker in chat to save it to your personal sticker collection.
    - Automatic background import of sent and received stickers from chat history.
  - **Message Editing & Deletion:**
    - Right-click sent messages to edit within the 15-minute WhatsApp window.
    - Inline edit mode with cancel action, displaying an `Edited` indicator and edit timestamp history.
    - Revoke/delete sent messages for everyone or for yourself.
  - **Cross-Device Pinned & Archived Sync:** Pinned and archived statuses are synced across all devices using WhatsApp App State synchronization (`events.Pin` and `events.Archive`). Any pin or archive changes made on your phone or in WhatsDown are updated seamlessly across devices.
  - **Right-Click Context Menu:** Right-click on any chat to:
    - **Pin / Unpin chat:** Keeps important conversations pinned to the top of your list with a visual pin indicator and syncs state to WhatsApp.
    - **Details & Media:** Opens a modal dialog displaying contact details, phone/account status, and an interactive gallery of shared media with a dedicated **Photos & Videos Gallery Grid** and categorized **Documents & Audio** list.
    - **Archive / Unarchive conversation:** Move conversations between the active chat list and the dedicated Archived page.
  - **Sidebar User & Chat Search Bar:** Top-level search bar on the sidebar allowing instant real-time filtering of contacts, users, and conversations by name, username, or phone number.
  - **In-Conversation Message Search:** Dedicated search button on the right side of the conversation header that reveals an integrated search bar to filter and locate messages in real time.
  - **Clickable Chat Header:** Active conversation title and avatar are placed in a prominent clickable header button with a right chevron indicator that opens the Contact Details & Media dialog directly.
  - **Photos & Videos Gallery & Shared Media Rendering:**
    - High-performance visual gallery grid (`GtkFlowBox`) for photos and videos with video duration badges and click-to-open actions.
    - Automatic asynchronous background loading and caching of media thumbnails directly into preview widgets.
    - One-click native launching (`xdg-open` / default desktop apps) for photos, videos, documents, and voice messages.
    - Shared locations open directly in OpenStreetMap / maps in your default browser.
  - **Clean Contact Name & Number Formatting:**
    - Prioritizes saved phone address book contacts and LID mappings over raw numbers.
    - Unsaved users display clean international phone numbers (e.g. `+1234567890`) rather than internal JID strings (`@s.whatsapp.net` or `@lid`).
    - Unnamed group chats display cleanly as "Group".
  - **Group Sender Color Coding & DM Bubble Streamlining:**
    - In direct messages (1-on-1 chats), redundant usernames on top of incoming message bubbles are hidden for a clean look.
    - In group chats, usernames are displayed using deterministic, high-contrast colors derived from a hash seed (JID/username) so colors stay consistent across restarts and re-renders.
  - **Clickable @ Mentions:**
    - `@` user tags within messages are formatted as clickable links that open the contact details and shared media modal for the tagged participant.
  - **Archived Chats Page:** Dedicated view displaying all archived chats with instant unarchiving and full messaging support.
  - **Settings Page:** Comprehensive configuration view:
    - **Account Profile:** Connected push name, phone number, and avatar.
    - **Sync & Devices:** Manual "Sync Now" button to trigger immediate app state re-synchronization with WhatsApp servers.
    - **Storage & Data:** Database path location, total stored chats and messages counters, and one-click avatar cache cleaner.
    - **Appearance:** Native dark mode switcher using Libadwaita's style manager.
    - **Session Management:** One-click Log Out action to unlink and return to QR login.
  - **Conversation View:** Contact avatar, title, incoming/outgoing chat bubbles, sender tags, timestamps, and message input bar for sending messages directly.
- **Profile Picture & Avatar Support:**
  - Uses `Adw.Avatar` for clean circular avatars with automatic initials and deterministic fallback colors.
  - Fetches contact and group profile pictures via `whatsmeow`'s profile picture APIs.
  - Two-level caching (in-memory textures and disk cache in `~/.local/share/whatsdown/avatars/`) to eliminate unnecessary network traffic.
  - Automatically updates on profile photo change events (`events.Picture`).
- **Smart Session Persistence:** Follows the XDG Base Directory specification, storing sessions and chat data at `~/.local/share/whatsdown/whatsdown.db`. When an active session is detected, the QR step is bypassed automatically.
- **Local Chat & Message Storage:** SQLite-backed storage for chats and messages:
  - Preserves initial history sync from your phone
  - Stores live messages received while connected
  - Tracks message edits and message revokes (deletions)
  - Syncs unread counters, read receipts, pin states, and archive states
- **Developer Live Reload:** Real-time hot-reloading for Blueprint templates (`.blp`), compiled UI definitions (`.ui`), CSS styles, and automated restarts on Go source changes.

---

## Prerequisites

Ensure you have the following system dependencies installed:

### Debian / Ubuntu
```bash
sudo apt install -y \
  golang \
  libgtk-4-dev \
  libadwaita-1-dev \
  blueprint-compiler \
  libsqlite3-dev \
  gcc \
  pkg-config
```

### Fedora
```bash
sudo dnf install -y \
  golang \
  gtk4-devel \
  libadwaita-devel \
  blueprint-compiler \
  sqlite-devel \
  gcc \
  pkgconf-pkg-config
```

### Arch Linux
```bash
sudo pacman -S --needed \
  go \
  gtk4 \
  libadwaita \
  blueprint-compiler \
  sqlite \
  gcc \
  pkgconf
```

---

## Building and Running

### Development Mode (with Live Reload)

The repository includes a development script `run.sh` that watches for file changes:
- Changes to `ui/*.blp` are automatically recompiled into `.ui` via `blueprint-compiler`.
- Changes to UI templates and CSS files are reloaded directly into the running application.
- Changes to Go source code trigger an automatic recompile and restart.

To start in development mode:
```bash
./run.sh
```

To preview individual screens directly without pairing:
```bash
# Preview the syncing progress screen:
WHATSDOWN_DEV_PAGE=syncing ./run.sh

# Preview the main chat interface:
WHATSDOWN_DEV_PAGE=main ./run.sh
```

### Standard Run (Embedded Assets)

To run the application using compiled, embedded assets without file watching:
```bash
blueprint-compiler batch-compile ui ui ui/*.blp
WHATSDOWN_DEV=0 ./run.sh
```
Or build the binary directly:
```bash
go build -o whatsdown .
./whatsdown
```

### Running Tests

```bash
go test -v ./...
```

---

## Database Architecture

WhatsDown stores application data in SQLite at `~/.local/share/whatsdown/whatsdown.db`:

- **Session store (`whatsmeow_*` tables):** Managed by `whatsmeow` for encryption keys, paired device credentials, and connection state.
- **Chat store (`whatsdown_chats`):** Tracks chat JIDs, contact/group names, unread message counts, timestamps, pinned/archived/muted states, and community hierarchy (`parent_jid`, `is_parent`, `is_default_sub_group`).
- **Message store (`whatsdown_messages`):** Tracks message IDs, sender JID/name, direction (`from_me`), timestamp, message kind (text, media, location, poll, call, voice, etc.), display text, edit state, read status, and raw protobuf payload.
- **Call store (`whatsdown_calls`):** Tracks call IDs, caller and chat JIDs, caller name, timestamps, duration, call type (audio/video, direct/group), direction, and outcome (connected, missed, rejected, declined).
- **Poll votes (`whatsdown_poll_votes`):** Tracks vote selections per message and voter JID.
- **Saved stickers (`whatsdown_saved_stickers`):** Tracks personal saved sticker assets and metadata.
- **Message edits (`whatsdown_message_edits`):** Stores previous versions of edited messages.
- **Migration table (`whatsdown_version`):** Tracks schema migrations applied to the local database.

---

## License

This project is licensed under the [MIT License](LICENSE).