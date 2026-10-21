# Showcase

A tour of everything an extension can do in [rhun](https://rhun.app), in one small Go program.
Install it from rhun: the extensions button in the title bar (the puzzle piece), **Install from
GitHub…**, then `rhun-editor/showcase`. Then run **Showcase Tour** (ctrl+alt+t) and pick a demo.

| Command | What it shows | Protocol |
| --- | --- | --- |
| Insert Greeting | types the Greeting setting at the cursor | `text`, `insert` |
| Change Case of Selection | upper, lower, title or reverse, as the setting says; one undo step | `selection`, `range`, `replace` |
| Wrap Selection in Brackets | `()` `[]` `{}` `<>`, as the setting says | `range`, `replace` |
| Document Statistics | lines, words, characters, and the edits seen so far | `text`, `edits`, `toast` |
| Mark TODO and FIXME | FIXME as an error, TODO a warning, NOTE information, IDEA a hint; gutter dots, underlines, the count in the status bar, the message on hover | `diagnostics` |
| Clear Marks | removes them | `diagnostics` |
| Go to Next TODO | selects the next TODO or FIXME | `select` |
| Say Hello to Someone | asks a name, then greets it | `ask`, `insert` |
| Open the Project README | opens README.md at its first line | `open` |
| Suggest a Terminal Command | puts a command in rhun's prompt; you run it with Enter | `term` |
| Toggle Word Wrap | runs rhun's own command | `cmd` |
| Use the Showcase Tab Width | sets this file's tab width from the setting | `option` |
| Add a Command While Running | adds **Insert the Time** to the palette until the extension stops | `register` |
| Showcase Tour | a pick list of all of the above | `choose` |

Without being asked, it also:

- marks TODO, FIXME, NOTE and IDEA in every open file as you type (setting: Mark as you type);
- shows the active file's word count in the status bar (setting: Word count in the status bar);
- tells how often a word appears when the pointer rests on it (`hover`);
- ships a theme, **Showcase Dusk** (Select Theme), and a grammar for `*.showcase` files.

From its manifest alone (protocol 1.1), with no code:

- a status bar item, **$(spark) Showcase**, that starts the tour, and a button for it at the end
  of the tab strip;
- editor menu items (Insert Greeting; Wrap Selection in Brackets when text is selected) and an
  explorer item on the project folder (Open the Project README);
- a chord, ctrl+k ctrl+g, for Insert Greeting, and icons for its commands;
- a context key `showcase.mode` for when clauses;
- language keys for `*.showcase` files: `step` indents the next line, `done` outdents, `else`
  sits in between, `#>` lines continue, `#|| |#` doc comments, `#region` folds;
- a file icon theme, **Showcase Icons** (Select File Icon Theme).

## Settings

On its page (Extensions, then Details): Greeting, Change case to, Brackets, Mark as you type,
Word count in the status bar, Tab width. They arrive as `setting` events; rhun keeps them.

## Permissions

It can read and change the files you have open, read the files in your project, and suggest
terminal commands that you run yourself. No internet, no other programs.

## Build and test

`go test ./...` covers the text work. `tests/run.sh [RHUN]` builds it, lays it out as rhun
installs it and drives a headless rhun through every command. A tag `vX.Y.Z` publishes a
release in the layout rhun installs from (`.github/workflows/release.yml`).
