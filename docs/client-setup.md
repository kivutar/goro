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

## GRF archives

To choose archives and their priority, put `DATA.INI` in the data folder:

```ini
[Data]
0=custom.grf
1=rdata.grf
2=data.grf
```

List the archives your installation uses. Lower numbers have higher priority;
the first archive containing a requested file wins. Numbers are sorted
numerically, regardless of line order. Only listed archives are loaded, including
any `.gpf` patches. Relative paths are resolved from the data folder; absolute
paths also work. `DATA.INI`, its section name, and relative archive paths are
looked up case-insensitively. Both Windows and Unix separators are accepted in
relative paths. Blank entries are ignored. Comment lines can start with `;` or `#`.
These comments can also follow a section header, such as `[Data] ; archives`.
UTF-8 files (with or without a BOM) and UTF-16 files with a BOM are supported.

If `DATA.INI` is absent, Goro loads existing archives in this order:

1. `fdata.grf`
2. `rdata.grf`
3. `sdata.grf`
4. `data.grf`

`data.grf` is the main archive; the [rAthena documentation](https://github.com/rathena/rathena/wiki/DATA.INI)
describes `rdata.grf` and the older `sdata.grf` as Sakray archives. Goro also
includes `fdata.grf` as a compatibility layer when present.

Other archives, including `event.grf`, require an explicit `DATA.INI` entry.
An empty `[Data]` section selects no archives. Invalid configuration or an
unreadable selected archive produces a startup error. Loose files in the data
folder still take priority over archive entries.
