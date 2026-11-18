"""Read-only validation of the provisioned local stress roster."""

import contextlib
import io
from collections import Counter

import yaml
from database import query, STATE
from provision import main
with contextlib.redirect_stdout(io.StringIO()):
    data = main()
jobs = yaml.safe_load((data['ROOT'] / 'db/pre-re/job_stats.yml').read_text())['Body']
rows = query("SELECT l.userid,c.char_id,c.`str`,c.`option`,c.party_id,c.online FROM `char` c JOIN login l USING(account_id) WHERE l.userid REGEXP '^stress[0-9][0-9]$' ORDER BY l.userid;").splitlines()[1:]
assert len(rows) == len(data['roster']) == 57
for row, role in zip(rows, data['roster']):
    user, cid, strength, option, pid, online = row.split('\t')
    assert online == '0' and int(pid) > 0
    name, job, weapon, attacks, passives, ranged, support = role
    learned = {int(r.split('\t')[0]): int(r.split('\t')[1]) for r in query(f'SELECT id,lv FROM skill WHERE char_id={cid};').splitlines()[1:]}
    active = {**attacks, **support, **data['buffs_by_role'].get(name, {})}
    for skill, level in active.items():
        assert learned[data['skill_db'][skill]['Id']] >= level, (name, skill)
    items = [list(map(int, r.split('\t'))) for r in query(f'SELECT nameid,amount,equip FROM inventory WHERE char_id={cid};').splitlines()[1:]]
    weight = sum((data['item_db'][item].get('Weight', 0) * amount for item, amount, _ in items))
    capacity = next((j['MaxWeight'] for j in jobs if j['Jobs'].get(name))) + int(strength) * 300 + 2000 * learned.get(36, 0)
    assert weight < capacity / 2, (name, 'overweight', weight, capacity)
    if 'MC_CARTREVOLUTION' in attacks:
        assert int(option) & 8 and learned.get(39, 0) > 0
    if 'HT_BLITZBEAT' in attacks:
        assert int(option) & 16 and learned.get(127, 0) > 0
    if 'CR_SHIELDCHARGE' in attacks:
        assert any((equip == 32 for _, _, equip in items))
    print(f'{user} {name}: {len(active)} active skills, party {pid}, weight {weight / capacity:.0%}, offline')
print(query("SELECT p.name,leader.name AS leader,COUNT(c.char_id) AS members,p.item FROM party p JOIN `char` c USING(party_id) JOIN `char` leader ON leader.char_id=p.leader_char WHERE p.name REGEXP '^Goro (Alpha|Beta|Gamma)( [23])?$' GROUP BY p.party_id ORDER BY p.name;"))
assert all((count == 3 for count in Counter((role[1] for role in data['roster'])).values()))
for index in range(1, 58):
    path = STATE / f'stress{index:02}.ini'
    assert path.is_file() and path.stat().st_mode & 0o777 == 0o600, path
print('All 57 credentials present with mode 600; three characters per class.')
