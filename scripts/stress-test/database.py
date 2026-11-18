"""Local rAthena database access shared by the stress-test tools."""

import os
import re
import subprocess
from pathlib import Path

ROOT = Path(os.environ.get("GORO_STRESS_RATHENA_DIR", Path(__file__).resolve().parents[3] / "rathena")).resolve()
STATE = Path(os.environ.get("GORO_STRESS_STATE", "/tmp/goro-stress")).resolve()


def query(sql):
    values = {}
    for name in ["conf/inter_athena.conf", "conf/import/inter_conf.txt"]:
        path = ROOT / name
        if not path.exists():
            continue
        for line in path.read_text().splitlines():
            match = re.match(r"^(\w+):\s*(.*?)\s*(?://.*)?$", line)
            if match:
                values[match[1]] = match[2]
    if values.get("map_server_ip") not in {"127.0.0.1", "localhost", "::1"}:
        raise ValueError("Provisioning requires a local rAthena database")
    env = os.environ.copy()
    env["MYSQL_PWD"] = values["map_server_pw"]
    result = subprocess.run(
        ["mariadb", "--batch", "-h", values["map_server_ip"],
         "-u", values["map_server_id"], values["map_server_db"]],
        input=sql, env=env, capture_output=True, text=True,
    )
    if result.returncode:
        # The client's error text can repeat SQL containing generated passwords.
        raise RuntimeError("MariaDB query failed; check the local server configuration")
    return result.stdout
