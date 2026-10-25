# Changelog

## Release 3 (Functional)

### Added

- Added sender avatars to direct-message notifications and group avatars to group notifications.
- Added coloured chat folders with avatar clusters, multiple-chat selection, drag-and-drop organization, and an Add to Folder context-menu action.
- Added message selection, forwarding to multiple recipients, deletion, and per-recipient Message Info.
- Added the last message sender to group chat previews.
- Added in-window photo, video, and PDF viewers with zoom, rotation, page navigation, and actions to reply, forward, locate, or open media in another application.
- Added voice-message seeking, pause/resume, and pitch-preserving playback at 0.5x, 1x, 1.5x, and 2x.
- Added keyboard shortcuts for chat navigation, selected-text formatting, editing the last sent message, sending files, and viewing shortcut help from the keyboard or Settings.
- Added a Read All button to Chats and a About dialog in Settings.
- Added clickable message links and day separators labelled Today, Yesterday, or the date.

### Fixed

- Fixed desktop notifications so clicking them opened the app and navigated to the relevant chat.
- Fixed emoji autocomplete leaving shortcode text behind when messages contained emoji or accented characters.
- Fixed read and delivery status synchronization across linked devices and group participants.
- Fixed excessive memory usage when opening chats, viewing media, and syncing with WhatsApp.
- Fixed typing stalls when updating the message composer and loading mention suggestions.
- Fixed the conversation jumping to the bottom when selecting a message to reply to.
- Fixed voice playback controls triggering replies and improved the visibility of read ticks and outgoing poll results.
- Fixed a Markdown rendering crash when typing a number followed by a space and a letter in the message box.

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
