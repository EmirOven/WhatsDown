# Changelog from v0.1.0 to v0.2.0 (Colourful)

## Added

- The new Colourful UI is here! Bored of that Gnome-ish gray on the background of your most precious chats? Now you can set the background of any chat independently to be a solid colour, or your favourite image!
- Tinting is a new opt-in customization option that takes the background colour of your chat, and tints your text box and title bar with that colour! It is opt-in currently because it has some non-breaking visual bugs, and will become opt-out in future versions.
- The Colourful update comes with a brand new way of visually memorizing your chats. Colour Handles are thin lines on the left of your chats, inhereting the background colour of your chat. If checking a chat's avatar or name takes too much time, just look at its colour!
- A group creator: You can now create group chats directly from WhatsDown.
- Group settings: You do not need to use your phone to moderate your groups now, you can change group's settings, or remove/add users from your computer.
- Autocomplete for tags: When you are tagging your friend in a group with @username (or annoying everyone by tagging @all), you now get autocompletion after you type @. The autocomplete does a fuzzy search of all participants, and you can select by clicking or with arrow keys and enter.
- Added rendering for @ tags on message bubbles. They previously showed the internal ID of the user instead of their display name, and they were not interactable.

## Fixed

- Fixed a bug where the app would try to scroll down on long chats for up to 1500ms, so users couldn't scroll up for a few seconds after switching to a chat.
- Duplicate chats should (hopefully) be gone now. Please create an issue report if you encounter it.
- Fixed the rendering of communities in the chats list, where their avatars were not a 1:1 circle and they were not properly aligned.
- Fixed the rendering of avatars as well, now they shouldn't re-render every time the view updates.
- Fixed a bug where if you switched from a chat with an avatar to a chat without one, the avatar on the title bar wouldn't update and show the avatar of the previous chat instead.