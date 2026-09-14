# Client Setup

Goro needs a 2008-era Ragnarok Online data folder. For local testing, use:

- https://pso-hack.com/files/OldRO.zip

Extract it somewhere, then edit `data/clientinfo.xml` and point the login server
to your rAthena host:

```xml
<connection>
	<display>Goro Local</display>
	<desc>Local rAthena</desc>
	<address>127.0.0.1</address>
	<port>6900</port>
	<version>18</version>
	<langtype>1</langtype>
</connection>
```

Use `127.0.0.1` when rAthena runs on the same machine. For LAN testing, replace
it with the rAthena host IP, and make sure `char_ip` and `map_ip` match on the
rAthena side.

Put the `goro` executable in the extracted OldRO folder and run it from there:

```sh
./goro
```

If you keep the executable elsewhere, pass the data folder explicitly:

```sh
./goro --data-dir /path/to/OldRO
```

## Chat shortcuts

Press **Alt+M** in game to edit the ten **Alt+1 … Alt+0** shortcuts. They can
contain an emote (`/ho`), a command (`/sit`), or a chat message. **View** opens
the emote list to fill the selected field. Clear a field to disable its shortcut.

Bindings are saved in your user `goro.ini`, independently of skill/item hotkeys.
The number row uses physical keys, so AZERTY does not require Shift. AltGr is
reserved for typing. Shortcuts do not repeat while held or run while editing
another game form.
