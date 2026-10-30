# Local 57-player stress test

A local observation test using headless clients and an existing pre-renewal rAthena
database with 57 non-GM accounts: `stress01` through `stress57`, with three
characters of each of the nineteen standard first and second classes.
Each has a character in slot 0, initially level 60 (first class) or 70 (second
class), job level 50, with equipment, learned skills, potions and resurrection
tokens. Non-test accounts are untouched.

## Setup

Use the compatible server described in [rAthena Setup](../../docs/rathena-setup.md).
The tools need Python 3 with PyYAML, the `mariadb` CLI, Bash, and a Linux systemd
user session with cgroup memory control. Set these from the Goro repository:

```sh
export GORO_STRESS_RATHENA_DIR="$HOME/src/rathena"
export GORO_STRESS_DATA_DIR="/path/to/OldRO"
export GORO_STRESS_STATE="/tmp/goro-stress"

python3 scripts/stress-test/provision.py           # read-only validation
python3 scripts/stress-test/provision.py --apply   # create/reset test characters
GOMAXPROCS=2 CGO_ENABLED=0 go build -p 1 -tags nofakecgo -o "$GORO_STRESS_STATE/goro" .
```

Install PyYAML in a virtual environment or with your distribution's package
manager if it is not already available. The rAthena checkout defaults to the
sibling `../rathena`, and runtime state defaults to `/tmp/goro-stress`.
`GORO_STRESS_DATA_DIR` is required by the launcher. The data's first login-server
entry must point to the local server at `127.0.0.1:6900`.

Provisioning reads database connection settings from rAthena's
`conf/inter_athena.conf` and `conf/import/inter_conf.txt`; only a local database
is accepted. It requires the test accounts to be offline. Existing test names,
non-GM groups and ownership markers are checked before resetting skills,
inventory, stats and positions. Original affected rows are saved under the
runtime directory. Random passwords and mode-600 client configs stay there;
missing configs cause passwords to be regenerated for those test accounts.
Run the Python tools normally, without `-O`, so their validation assertions run.

The same command generates `login.conf`, `char.conf` and `map.conf` overrides
under the runtime directory. With any existing local servers stopped, run these
in three separate terminals from the rAthena checkout:

```sh
./login-server --login-config "$GORO_STRESS_STATE/login.conf"
./char-server --char-config "$GORO_STRESS_STATE/char.conf"
./map-server --map-config "$GORO_STRESS_STATE/map.conf"
```

The overrides use localhost and load `mobs.txt` without editing rAthena's
configuration. Restart the map server to change these static spawns; reloading
an NPC file does not remove its previous static monster spawns.

## Roster

| Accounts | Class | Combat |
| --- | --- | --- |
| 01, 20, 39 | Swordsman | Bash, Provoke, Magnum Break |
| 02, 21, 40 | Mage | Three bolts, Soul Strike, Fire Ball, Frost Diver |
| 03, 22, 41 | Archer | Double Strafe, Arrow Repel |
| 04, 23, 42 | Acolyte | Holy Light, Decrease AGI, Heal, Blessing, Increase AGI, Angelus |
| 05, 24, 43 | Merchant | Mammonite, Cart Revolution |
| 06, 25, 44 | Thief | Envenom, Steal, Double Attack |
| 07, 26, 45 | Knight | Pierce, Spear Stab, Spear Boomerang, Bowling Bash |
| 08, 27, 46 | Priest | Holy Light, Heal, Blessing, Impositio Manus, Kyrie Eleison, Aspersio, Angelus, Magnificat |
| 09, 28, 47 | Wizard | Jupitel Thunder, Earth Spike, Fire Ball, Frost Diver, Soul Strike |
| 10, 29, 48 | Blacksmith | Mammonite, Cart Revolution, Adrenaline Rush, Weapon Perfection |
| 11, 30, 49 | Hunter | Blitz Beat with a falcon, Double Strafe, Arrow Repel |
| 12, 31, 50 | Assassin | Sonic Blow with a katar, Envenom, Steal |
| 13, 32, 51 | Crusader | Holy Cross, Shield Boomerang, Shield Charge |
| 14, 33, 52 | Monk | Ki Explosion, Holy Light, Triple Attack, Heal, Blessing, Increase AGI, Angelus |
| 15, 34, 53 | Sage | Earth Spike, three bolts, Frost Diver, Soul Strike |
| 16, 35, 54 | Rogue | Envenom, Steal, Mug, Double Attack, Close Confine |
| 17, 36, 55 | Alchemist | Acid Terror, Cart Revolution, Aid Potion, Alchemical Weapon |
| 18, 37, 56 | Bard (male) | Melody Strike, Magic Strings |
| 19, 38, 57 | Dancer (female) | Slinging Arrow, Gypsy's Kiss |

These are the six standard first classes and thirteen standard second classes.
There are no baby or rebirth classes. `roster.lua` defines their script behavior.
Skill prerequisites and weapon restrictions were checked against the local
rAthena databases. Second classes have spent the required 49 first-class skill
points. The Lua API now accepts self-targeted skills through the existing
`goro.skill(goro.player().id, name)` call, using normal client skill handling.
Ground-targeted spells are still outside this script's scope. Carts, arrows,
holy water, acid bottles and coating bottles supply the relevant requirements.

## Parties

These are actual local-server parties stored in SQL. Membership loads on the
next login. All 57 characters were offline when assigned. The same three-party
composition is repeated for accounts 01–19, 20–38 and 39–57, making nine parties
of seven, six and six members per group. Each party has its own healer.

| Party | Leader | Members |
| --- | --- | --- |
| Goro Alpha / Alpha 2 / Alpha 3 | Priest | Swordsman, Priest, Wizard, Blacksmith, Hunter, Assassin, Bard |
| Goro Beta / Beta 2 / Beta 3 | Acolyte | Mage, Archer, Acolyte, Merchant, Thief, Dancer |
| Goro Gamma / Gamma 2 / Gamma 3 | Monk | Knight, Crusader, Monk, Sage, Rogue, Alchemist |

Each healer periodically casts Angelus; the Priest also casts Magnificat.
The server applies these effects to party members within its normal range.
Targeted support selects visible living party members or the caster, and healing
selects an injured member. Aid Potion targets another party member. Parties
permit members to pick up each other's drops, with individual EXP and loot
allocation. Bard and Dancer performances use their normal server area rules.

All run `combat.lua`: they stay near one location, choose targets independently,
loot between fights, use potions, rest when resources run low, and use a token
if killed. Casters stay at casting range; support classes help their own
parties. Melee classes, Archers and Hunters use normal attacks between skills.
They make at most two pickup attempts between fights so all classes participate
in combat even when drops accumulate. Request counters in stdout report script
activity; they are not counts of confirmed hits or successful pickups.
Attack skills rotate so every configured skill gets a turn. Buffs refresh on
timers derived from the server's durations; performances run for their full
duration. Action waits use server cast times and delays at the learned levels,
without assuming DEX or buff reductions. These are conservative timing estimates,
not an API exposing server cooldowns. A successful Lua request does not confirm
server acceptance: targets may die mid-cast and supplies can run out.

## Validation

Run these checks with the bots stopped, from the Goro repository:

```sh
python3 scripts/stress-test/verify.py
lua scripts/stress-test/check-behavior.lua
GOMAXPROCS=2 CGO_ENABLED=0 go test -p 1 -tags nofakecgo ./game -run TestLuaBot
```

The database verifier checks the 57 characters' learned skills, party membership,
equipment, carrying weight and private configs. Provisioning validates the
server skill prerequisites. The Lua simulation checks attack rotation, self
buffs and party support without opening clients. Go tests exercise the Lua
self-cast packet and ordinary targeting behavior.

A live run with 16 bots and 108 extra monsters was reported smooth on a machine
with 15 GiB RAM. The earlier 57-bot attempt exhausted host RAM during login,
before all bots connected. Established bots reached roughly 340–360 MiB RSS in
the kernel snapshot; this motivated the launcher's memory limits. These are
manual observation results, not a performance guarantee for another machine.

## Watch

Connect normally to **Local**, using your usual account. On a GM character:

```text
@warp prt_fild08 216 255
```

The temporary server configuration adds **108 monsters**, twelve of each:
Spore, Wolf, Poporing, Peco Peco, Smokie, Bigfoot, Coco, Horn and Elder Willow.
They span levels 14–25, small through large sizes, and Water, Earth, Poison and
Fire elements. All spawn within six cells of the viewing point and respawn after
3–5 seconds. This triples the previous 36 extra monsters; normal map spawns
remain in place. Monsters can wander normally after spawning.
The bots stay within ten cells of the center. Keep effects enabled in the
graphical client so this exercises skill effects as well as actors and drops.

## Stop and restart the bots

While a batch runs in tmux, stop it without stopping the RO server:

```sh
tmux send-keys -t goro-stress-bots C-c
```

Keep the bots stopped while inspecting logs, to let the CPU cool down.

Start a batch in your terminal; Ctrl+C stops its children:

```sh
scripts/stress-test/run.sh       # 12 bots (default)
scripts/stress-test/run.sh 16    # largest batch allowed by the current estimate
```

Or leave it running in tmux:

```sh
tmux new-session -d -s goro-stress-bots -c "$PWD" scripts/stress-test/run.sh
```

The launcher spaces logins four seconds apart to respect the local server's
connection flood guard. It checks that all requested account configs exist
before starting and permits only one batch at a time. The launcher now estimates 400 MiB per bot and refuses
counts above the 6.5 GiB batch budget. It also requires at least 8.5 GiB available
before launch, leaving 2 GiB beyond the cap for desktop headroom.

The whole batch runs inside a systemd user scope with `MemoryMax=6656M` (6.5 GiB),
`MemorySwapMax=0`, and `OOMPolicy=kill`. If it exhausts that limit, the entire
bot scope is killed. The launcher fails if systemd cannot establish the scope.
The server and graphical client are outside it. The scope's cgroup limits were
verified, and the 16-bot batch ran inside it.

Bots retain one Go worker each, lower scheduling priority (`nice 10`), and
`GOMEMLIMIT=192MiB`. That Go setting is a soft limit and does not cap RSS. The
57 accounts, nine parties and 108 extra mobs remain prepared, but running all
57 at the observed footprint needs more RAM or lower bot memory use.

Credentials (mode 600), the built client, server overrides, and logs are in
`/tmp/goro-stress/`. Logs are overwritten on each batch; set
`GORO_STRESS_LOG_LEVEL=debug` for packet-handling details during diagnosis.
The `/tmp` files must be recreated if the system removes them. These characters
level up and consume supplies normally; this is a manual observation test, not
an indefinitely sustained or fixed-level benchmark.

## Server cleanup

Stop the bots before stopping the server. Disconnect your graphical client,
then stop the map, character and login servers in that order using Ctrl+C in
their terminals. Starting rAthena normally afterward omits the extra spawns.
The test accounts and parties remain in SQL for the next run.
