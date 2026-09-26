"""Provision the 57-character pre-renewal test roster. Dry run unless --apply."""
import argparse
import json
import math
import os
import secrets
from pathlib import Path

import yaml

from database import query, STATE, ROOT

# Class, job ID, weapon, attacks, passives, ranged behavior, targeted support.
roster = [
    ('Swordman', 1, 1104, {'SM_BASH': 3, 'SM_PROVOKE': 3, 'SM_MAGNUM': 3}, {'SM_SWORD': 10, 'SM_RECOVERY': 10}, False, {}),
    ('Mage', 2, 1602, {'MG_FIREBOLT': 3, 'MG_COLDBOLT': 3, 'MG_LIGHTNINGBOLT': 3, 'MG_SOULSTRIKE': 3, 'MG_FIREBALL': 3, 'MG_FROSTDIVER': 3}, {'MG_SRECOVERY': 10}, True, {}),
    ('Archer', 3, 1704, {'AC_DOUBLE': 5, 'AC_CHARGEARROW': 1}, {'AC_OWL': 5, 'AC_VULTURE': 5}, True, {}),
    ('Acolyte', 4, 1504, {'AL_HOLYLIGHT': 1, 'AL_DECAGI': 3}, {}, True, {'AL_HEAL': 5, 'AL_BLESSING': 3, 'AL_INCAGI': 3}),
    ('Merchant', 5, 1301, {'MC_MAMMONITE': 3, 'MC_CARTREVOLUTION': 1}, {'MC_INCCARRY': 5, 'MC_PUSHCART': 10}, False, {}),
    ('Thief', 6, 1204, {'TF_POISON': 3, 'TF_STEAL': 5}, {'TF_DOUBLE': 10, 'TF_MISS': 10}, False, {}),
    ('Knight', 7, 1404, {'KN_PIERCE': 3, 'KN_SPEARSTAB': 3, 'KN_SPEARBOOMERANG': 3, 'KN_BOWLINGBASH': 3}, {'KN_SPEARMASTERY': 5, 'SM_RECOVERY': 10}, False, {}),
    ('Priest', 8, 1504, {'AL_HOLYLIGHT': 1}, {'MG_SRECOVERY': 10}, True, {'AL_HEAL': 5, 'AL_BLESSING': 3, 'PR_IMPOSITIO': 3, 'PR_KYRIE': 3, 'PR_ASPERSIO': 3}),
    ('Wizard', 9, 1602, {'WZ_JUPITEL': 3, 'WZ_EARTHSPIKE': 3, 'MG_FIREBALL': 3, 'MG_FROSTDIVER': 3, 'MG_SOULSTRIKE': 3}, {'MG_SRECOVERY': 10}, True, {}),
    ('Blacksmith', 10, 1301, {'MC_MAMMONITE': 3, 'MC_CARTREVOLUTION': 1}, {'BS_WEAPONRESEARCH': 10, 'MC_INCCARRY': 5, 'MC_PUSHCART': 10}, False, {}),
    ('Hunter', 11, 1704, {'HT_BLITZBEAT': 3, 'AC_DOUBLE': 5, 'AC_CHARGEARROW': 1}, {'AC_OWL': 5, 'AC_VULTURE': 5, 'HT_STEELCROW': 5}, True, {}),
    ('Assassin', 12, 1250, {'AS_SONICBLOW': 3, 'TF_POISON': 3, 'TF_STEAL': 5}, {'AS_KATAR': 5, 'TF_MISS': 10}, False, {}),
    ('Crusader', 14, 1104, {'CR_HOLYCROSS': 3, 'CR_SHIELDBOOMERANG': 3, 'CR_SHIELDCHARGE': 3}, {'SM_SWORD': 5, 'SM_RECOVERY': 10}, False, {}),
    ('Monk', 15, 1801, {'MO_BALKYOUNG': 1, 'AL_HOLYLIGHT': 1}, {'MO_TRIPLEATTACK': 5}, False, {'AL_HEAL': 5, 'AL_BLESSING': 3, 'AL_INCAGI': 3}),
    ('Sage', 16, 1550, {'WZ_EARTHSPIKE': 3, 'MG_FIREBOLT': 3, 'MG_COLDBOLT': 3, 'MG_LIGHTNINGBOLT': 3, 'MG_FROSTDIVER': 3, 'MG_SOULSTRIKE': 3}, {'MG_SRECOVERY': 10}, True, {}),
    ('Rogue', 17, 1204, {'TF_POISON': 3, 'RG_CLOSECONFINE': 1, 'TF_STEAL': 5, 'RG_STEALCOIN': 3}, {'TF_DOUBLE': 10, 'TF_MISS': 10}, False, {}),
    ('Alchemist', 18, 1301, {'AM_ACIDTERROR': 3, 'MC_CARTREVOLUTION': 1}, {'MC_INCCARRY': 5, 'MC_PUSHCART': 10}, False, {'AM_POTIONPITCHER': 3, 'AM_CP_WEAPON': 1}),
    ('Bard', 19, 1901, {'BA_MUSICALSTRIKE': 3}, {'BA_MUSICALLESSON': 5}, True, {}),
    ('Dancer', 20, 1950, {'DC_THROWARROW': 3}, {'DC_DANCINGLESSON': 5}, True, {}),
]
buffs_by_role = {
    'Acolyte': {'AL_ANGELUS': 3},
    'Priest': {'AL_ANGELUS': 3, 'PR_MAGNIFICAT': 3},
    'Monk': {'AL_ANGELUS': 3},
    'Blacksmith': {'BS_ADRENALINE': 3, 'BS_WEAPONPERFECT': 5},
    'Bard': {'BA_POEMBRAGI': 3},
    'Dancer': {'DC_SERVICEFORYOU': 3},
}
parties = [
    ('Goro Alpha', 8, [1, 8, 9, 10, 11, 12, 18]),
    ('Goro Beta', 4, [2, 3, 4, 5, 6, 19]),
    ('Goro Gamma', 14, [7, 13, 14, 15, 16, 17]),
]
class_count = len(roster)
roster = roster * 3
parties = [(name + (f' {wave + 1}' if wave else ''), leader + class_count * wave, [i + class_count * wave for i in members]) for wave in range(3) for name, leader, members in parties]

def main(apply=False):
    data = ROOT / 'db/pre-re'
    skill_db = {s['Name']: s for s in yaml.safe_load((data / 'skill_db.yml').read_text())['Body']}
    trees = {s['Job']: s for s in yaml.safe_load((data / 'skill_tree.yml').read_text())['Body']}
    item_db = {i['Id']: i for f in ['item_db_equip.yml', 'item_db_etc.yml', 'item_db_usable.yml'] for i in yaml.safe_load((data / f).read_text())['Body']}
    assert query("SELECT COUNT(*) FROM login l JOIN `char` c USING(account_id) WHERE l.userid REGEXP '^stress[0-9][0-9]$' AND c.online<>0;").splitlines()[-1] == '0', 'Stop bots before provisioning'
    existing = {l.split('\t')[0]: l.split('\t')[1:] for l in query("SELECT l.userid,l.account_id,c.char_id FROM login l JOIN `char` c USING(account_id) WHERE l.userid REGEXP '^stress[0-9][0-9]$';").splitlines()[1:]}
    assert len(existing) in [0, 19, 57] and set(existing) == {f'stress{i:02}' for i in range(1, len(existing) + 1)}, 'Unexpected test accounts'
    assert query("SELECT COUNT(*) FROM login WHERE userid REGEXP '^stress[0-9][0-9]$' AND (email<>'stress@localhost' OR group_id<>0);").splitlines()[-1] == '0', 'A stress username belongs to a non-test account'
    assert int(query("SELECT COUNT(*) FROM login WHERE userid REGEXP '^stress[0-9][0-9]$';").splitlines()[-1]) == len(existing), 'Unexpected extra account or character slot'
    os.umask(0o077)
    if apply:
        STATE.mkdir(parents=True, exist_ok=True, mode=0o700)
    for table in ['char', 'skill', 'inventory']:
        q = f"SELECT t.* FROM `{table}` t JOIN `char` c ON t.char_id=c.char_id JOIN login l ON c.account_id=l.account_id WHERE l.userid REGEXP '^stress[0-9][0-9]$';"
        if apply and (not (STATE / f'before-scale-{table}.tsv').exists()):
            (STATE / f'before-scale-{table}.tsv').write_text(query(q))
    sql = ['START TRANSACTION;']
    configs = {}
    lua = ['-- Three copies of every class, matching stress01 through stress57. All skill levels come from the server roster.', 'return {']
    for index, (name, job, weapon, attacks, passives, ranged, support) in enumerate(roster, 1):
        wave = (index - 1) // class_count
        char_name = f'Stress {name}' + (f' {wave + 1}' if wave else '')
        assert len(char_name) < 24
        tree = {s['Name']: s for parent in [*trees[name].get('Inherit', {}), name] for s in trees[parent].get('Tree', [])}
        learned = {}
        buffs = buffs_by_role.get(name, {})

        def learn(skill, level):
            assert skill in tree, (name, skill, 'not in skill tree')
            assert level <= tree[skill]['MaxLevel'], (name, skill, level)
            if learned.get(skill, 0) >= level:
                return
            learned[skill] = level
            for req in tree[skill].get('Requires', []):
                learn(req['Name'], req['Level'])
        for skill, level in {'NV_BASIC': 9, **attacks, **passives, **support, **buffs}.items():
            learn(skill, level)
        # Second-job skills require 49 spent first-job points at job level 50.
        first_name = name if job <= 6 else next((parent for parent in trees[name]['Inherit'] if parent != 'Novice'))
        first_skills = {s['Name'] for s in trees[first_name]['Tree'] if not skill_db[s['Name']].get('Flags', {}).get('IsQuest')}

        def first_spent():
            return sum((level for skill, level in learned.items() if skill in first_skills))
        if job > 6:
            while first_spent() < 49:
                before = first_spent()
                for entry in trees[first_name]['Tree']:
                    skill = entry['Name']
                    if skill not in first_skills or learned.get(skill, 0) >= entry['MaxLevel']:
                        continue
                    saved = learned.copy()
                    learn(skill, learned.get(skill, 0) + 1)
                    if first_spent() > 49:
                        learned = saved
                    if first_spent() == 49:
                        break
                assert first_spent() > before, (name, 'cannot allocate first-job points')
            second_spent = sum((level for skill, level in learned.items() if skill not in first_skills and skill != 'NV_BASIC' and (not skill_db[skill].get('Flags', {}).get('IsQuest'))))
            assert 0 <= second_spent <= 49, (name, 'too many second-job points', second_spent)
            unspent = 49 - second_spent
        else:
            unspent = 49 - first_spent()
        assert unspent >= 0
        for skill in attacks:
            assert skill_db[skill].get('TargetType') == 'Attack' or skill == 'SM_MAGNUM', (name, skill)
        for skill in buffs:
            assert skill_db[skill].get('TargetType') == 'Self', (name, skill)
        for skill in support:
            assert skill_db[skill].get('TargetType') == 'Support', (name, skill)
        item = item_db[weapon]
        item_job = 'BardDancer' if name in ['Bard', 'Dancer'] else name
        assert item['Jobs'].get(item_job), (name, weapon)
        sex = 'M' if name == 'Bard' else 'F' if name == 'Dancer' else 'M' if index % 2 == 0 else 'F'
        assert item.get('Gender', 'Both') in ['Both', {'M': 'Male', 'F': 'Female'}[sex]]
        magic = name in ['Mage', 'Acolyte', 'Priest', 'Wizard', 'Sage']
        archer = name in ['Archer', 'Hunter', 'Bard', 'Dancer']
        stats = (15, 15, 35, 65, 50, 1) if magic else (20, 45, 30, 20, 60, 10) if archer else (55, 45, 40, 20, 40, 10)
        user = f'stress{index:02}'
        if user in existing:
            aid, cid = existing[user]
            sql += [f'SET @aid={int(aid)};', f'SET @cid={int(cid)};', f"UPDATE login SET sex='{sex}' WHERE account_id=@aid;", 'DELETE FROM skill WHERE char_id=@cid;', 'DELETE FROM inventory WHERE char_id=@cid;', 'DELETE FROM skillcooldown WHERE char_id=@cid;']
        else:
            password = secrets.token_hex(10)
            sql += [f"INSERT INTO login(userid,user_pass,sex,email,group_id,character_slots) VALUES('{user}','{password}','{sex}','stress@localhost',0,9);", 'SET @aid=LAST_INSERT_ID();', f"INSERT INTO `char`(account_id,name,sex) VALUES(@aid,'{char_name}','{sex}');", 'SET @cid=LAST_INSERT_ID();']
            configs[user] = f'[login]\nusername = {user}\npassword = {password}\nchar_slot = 0\nkeep_id = false\n'
        if user in existing:
            actual = query(f'SELECT name,char_num FROM `char` WHERE char_id={int(cid)};').splitlines()[-1].split('\t')
            assert actual == [char_name, '0'], ('unexpected test character', user)
            if not (STATE / f'{user}.ini').exists():
                password = secrets.token_hex(10)
                sql.append(f"UPDATE login SET user_pass='{password}' WHERE account_id=@aid;")
                configs[user] = f'[login]\nusername = {user}\npassword = {password}\nchar_slot = 0\nkeep_id = false\n'
        vals = {
            'name': char_name,
            'class': job,
            'char_num': 0,
            'base_level': 60 if job <= 6 else 70,
            'job_level': 50,
            'base_exp': 0,
            'job_exp': 0,
            'zeny': 1000000,
            'str': stats[0],
            'agi': stats[1],
            'vit': stats[2],
            'int': stats[3],
            'dex': stats[4],
            'luk': stats[5],
            'hp': 10000,
            'max_hp': 10000,
            'sp': 10000,
            'max_sp': 10000,
            'option': 16 if name == 'Hunter' else 8 if 'MC_CARTREVOLUTION' in attacks else 0,
            'status_point': 0,
            'skill_point': unspent,
            'weapon': 0,
            'shield': 0,
            'head_top': 0,
            'head_mid': 0,
            'head_bottom': 0,
            'sex': sex,
            'hair': 1 + index % 10,
            'hair_color': index % 8,
            'last_map': 'prt_fild08',
            'last_x': 212 + (index - 1) % 5 * 2,
            'last_y': 252 + (index - 1) // 5 % 4 * 2,
            'last_instanceid': 0,
            'save_map': 'prt_fild08',
            'save_x': 216,
            'save_y': 255,
        }
        sql += ['UPDATE `char` SET ' + ','.join((f'`{k}`=' + ("'" + v + "'" if isinstance(v, str) else str(v)) for k, v in vals.items())) + ' WHERE char_id=@cid AND online=0;']
        for skill, level in learned.items():
            sql.append(f"INSERT INTO skill(char_id,id,lv,flag) VALUES(@cid,{skill_db[skill]['Id']},{level},0);")
        items = [(weapon, 1, 34 if item['Locations'].get('Both_Hand') else 2), (2305, 1, 16), (2401, 1, 64), (2501, 1, 4), (503, 40 if name == 'Alchemist' else 20, 0), (505, 40, 0), (7621, 10, 0)]
        if archer:
            items.append((1750, 2500, 32768))
        if name == 'Crusader':
            items.append((2101, 1, 32))
        if name == 'Priest':
            items.append((523, 100, 0))
        if name == 'Alchemist':
            items.extend([(7136, 60, 0), (7139, 30, 0)])
        for item_id, amount, equip in items:
            assert item_id in item_db
            sql.append(f'INSERT INTO inventory(char_id,nameid,amount,equip,identify) VALUES(@cid,{item_id},{amount},{equip},1);')

        # Conservative action waits use learned levels before DEX reductions.
        def value_at(skill, key):
            value = skill_db[skill].get(key, 0)
            return next((v.get('Time', 0) for v in value if v['Level'] == learned[skill]), 0) if isinstance(value, list) else value
        waits = {skill: max(6, math.ceil((value_at(skill, 'CastTime') + value_at(skill, 'AfterCastActDelay') + 300) / 150)) for skill in [*attacks, *support, *buffs]}
        buff_entries = []
        for skill in buffs:
            # Performances finish fully; other buffs refresh shortly before expiry.
            duration = value_at(skill, 'Duration1')
            interval = math.ceil((duration + 5000 if skill_db[skill].get('Flags', {}).get('IsSong') else duration * 0.9) / 150)
            buff_entries.append('{skill=' + json.dumps(skill) + ', interval=' + str(interval) + '}')
        healing = next((skill for skill in support if skill in ['AL_HEAL', 'AM_POTIONPITCHER']), None)
        party = next((p for p, _, members in parties if index in members))
        lua.append('\t{ name=' + json.dumps(name) + ', party=' + json.dumps(party) + ', ranged=' + str(ranged).lower() + ', skills={' + ','.join((json.dumps(s) for s in attacks)) + '}, support={' + ','.join((json.dumps(s) for s in support if s != healing)) + '}, healing=' + json.dumps(healing or '') + ', buffs={' + ','.join(buff_entries) + '}, waits={' + ','.join((skill + '=' + str(wait) for skill, wait in waits.items())) + '} },')
        print(f'{user}: {name}, job={job}, weapon={weapon}, learned skills={len(learned)}')
    assert sorted((i for _, _, members in parties for i in members)) == list(range(1, len(roster) + 1))
    for party, leader, members in parties:
        assert leader in members and len(members) <= 12
        users = ','.join(("'stress%02d'" % i for i in members))
        rows = query(f"SELECT party_id,leader_id,leader_char FROM party WHERE name='{party}';").splitlines()[1:]
        assert len(rows) <= 1, ('duplicate party name', party)
        if rows:
            pid, aid, cid = map(int, rows[0].split('\t'))
            assert [str(aid), str(cid)] == existing[f'stress{leader:02}'], ('unexpected party leader', party)
            current = query(f'SELECT l.userid FROM `char` c JOIN login l USING(account_id) WHERE c.party_id={pid} ORDER BY l.userid;').splitlines()[1:]
            assert current == [f'stress{i:02}' for i in sorted(members)], ('unexpected party members', party)
        else:
            assert query(f'SELECT COUNT(*) FROM `char` c JOIN login l USING(account_id) WHERE l.userid IN ({users}) AND c.party_id<>0;').splitlines()[-1] == '0', ('already in another party', party)
            sql += [f"INSERT INTO party(name,exp,item,leader_id,leader_char) SELECT '{party}',0,1,c.account_id,c.char_id FROM `char` c JOIN login l USING(account_id) WHERE l.userid='stress{leader:02}' AND c.char_num=0;", 'SET @party=LAST_INSERT_ID();', f'UPDATE `char` c JOIN login l USING(account_id) SET c.party_id=@party WHERE l.userid IN ({users}) AND c.online=0;']
    # Some rAthena tables use MyISAM: validate the entire plan before any writes.
    sql += ['COMMIT;']
    if apply:
        for user, config in configs.items():
            path = STATE / f'{user}.ini'
            path.write_text(config)
            path.chmod(0o600)
        print(query('\n'.join(sql)))
        Path(__file__).with_name('roster.lua').write_text('\n'.join(lua + ['}']) + '\n')
        (STATE / 'roster.json').write_text(json.dumps(roster, indent=2) + '\n')
        for kind, addresses in [('login', ''), ('char', 'login_ip: 127.0.0.1\nchar_ip: 127.0.0.1\n'), ('map', 'char_ip: 127.0.0.1\nmap_ip: 127.0.0.1\n')]:
            config = f'import: conf/{kind}_athena.conf\n' + addresses + 'bind_ip: 127.0.0.1\n'
            if kind == 'map':
                config += 'npc: ' + str(Path(__file__).with_name('mobs.txt').resolve()) + '\n'
            (STATE / f'{kind}.conf').write_text(config)
        print('Prepared 57 accounts, nine parties and server overrides in', STATE)
    else:
        print('Dry run passed. Add --apply to provision.')
    return {'roster': roster, 'buffs_by_role': buffs_by_role, 'ROOT': ROOT, 'STATE': STATE, 'item_db': item_db, 'skill_db': skill_db}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--apply', action='store_true', help='create or reset only the offline stress accounts')
    main(parser.parse_args().apply)
