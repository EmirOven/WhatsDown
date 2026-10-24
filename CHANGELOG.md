# Changelog

## Release 3

### Added

- Add local coloured chat folders with avatar clusters, Ctrl-click selection, drag-and-drop organization, and an Add to Folder context-menu action.
- Add message selection, forwarding, deletion, and per-recipient Message Info with direct chat actions.
- Show the last message sender in group chat previews.
- Add in-window photo, video, and PDF viewers with Reply, Forward, Go to Message, Open in Default App, and Show in Files actions.
- Add photo and PDF zoom, scrollwheel zoom, drag panning, rotation, and PDF page navigation. Escape closes the viewer.
- Add voice-message seeking, pause/resume, and pitch-preserving playback at 0.5x, 1x, 1.5x, and 2x.
- Add a compact Read All button to Chats and a native libadwaita About dialog in Settings.
- Update Release 3 packaging and build dependencies for native media playback and PDF rendering.

### Fixed

- Preserve phone-synced read state across delayed messages and history replay, and import WhatsApp receipt metadata including pending recipients.
- Merge recipient identity aliases, exclude the sender from group recipient counts, and retain WhatsApp's aggregate delivery/read status when the full recipient list is unavailable.
- Recover invalid WhatsApp app-state hashes with a verified collection snapshot and a single retry; serialize app-state writes to reduce conflicts.
- Reduce typing stalls by coalescing composer layout and preview work, loading mention candidates asynchronously, and batching receipt updates without rebuilding the sidebars.
- Expand and collapse folders using existing rows, preserve sidebar scroll position, and persist collapse state asynchronously.
- Keep voice controls compact and prevent playback controls from triggering double-click replies. Selecting a reply no longer scrolls the conversation to the bottom.
- Improve tick size and spacing, remove folder avatar outlines, and make outgoing poll progress bars visible.

- Reduce memory usage when opening chats, viewing media, and syncing with WhatsApp.
- Bound image caches, scale previews, and release callbacks for removed widgets.
- Fetch incremental app-state updates and coalesce sync refreshes.
- Add clickable message URLs and day separators labelled Today, Yesterday, or the date.
- Fixed an issue with Markdown rendering, where typing a number followed by a space and a letter crashed the app (do not ask how)

## v0.2.0 (Colourful)

### Added

- The new Colourful UI is here! Bored of that Gnome-ish gray on the background of your most precious chats? Now you can set the background of any chat independently to be a solid colour, or your favourite image!
- Tinting is a new opt-in customization option that takes the background colour of your chat, and tints your text box and title bar with that colour! It is opt-in currently because it has some non-breaking visual bugs, and will become opt-out in future versions.
- The Colourful update comes with a brand new way of visually memorizing your chats. Colour Handles are thin lines on the left of your chats, inhereting the background colour of your chat. If checking a chat's avatar or name takes too much time, just look at its colour!
- A group creator: You can now create group chats directly from WhatsDown.
- Group settings: You do not need to use your phone to moderate your groups now, you can change group's settings, or remove/add users from your computer.
- Autocomplete for tags: When you are tagging your friend in a group with @username (or annoying everyone by tagging @all), you now get autocompletion after you type @. The autocomplete does a fuzzy search of all participants, and you can select by clicking or with arrow keys and enter.
- Added rendering for @ tags on message bubbles. They previously showed the internal ID of the user instead of their display name, and they were not interactable.

### Fixed

- Fixed a bug where the app would try to scroll down on long chats for up to 1500ms, so users couldn't scroll up for a few seconds after switching to a chat.
- Duplicate chats should (hopefully) be gone now. Please create an issue report if you encounter it.
- Fixed the rendering of communities in the chats list, where their avatars were not a 1:1 circle and they were not properly aligned.
- Fixed the rendering of avatars as well, now they shouldn't re-render every time the view updates.
- Fixed a bug where if you switched from a chat with an avatar to a chat without one, the avatar on the title bar wouldn't update and show the avatar of the previous chat instead.
