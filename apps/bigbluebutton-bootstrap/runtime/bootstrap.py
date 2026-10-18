#!/usr/bin/env python3
"""Initialize only recognized fresh BBB databases; never drop meeting data."""
import argparse
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import time
import urllib.request
import yaml

os.umask(0o077)
SOURCE = Path('/usr/share/bbb-graphql-server')
MARKER = 'public.bigbluebutton_schema'

def schema_contents(path):
    # PGO and other installers may use a dedicated role instead of blocknote_app.
    role = os.environ.get('PGUSER', 'bbb').replace('"', '""')
    return path.read_text().replace('TO blocknote_app', f'TO "{role}"')


def psql(database, sql=None, file=None):
    command = ['psql', '-X', '-v', 'ON_ERROR_STOP=1', '-At', '-d', database]
    if file:
        command += ['--single-transaction', '-f', str(file)]
    else:
        command += ['-c', sql]
    return subprocess.check_output(command, text=True).strip()

def wait_database(database):
    for _ in range(120):
        if subprocess.run(['pg_isready', '-q', '-d', database]).returncode == 0:
            return
        time.sleep(5)
    raise RuntimeError(f'{database}: PostgreSQL did not become ready')

def schema_ready(database):
    wait_database(database)
    for _ in range(180):
        if psql(database, "SELECT to_regclass('public.bigbluebutton_schema')"):
            path = SOURCE / 'bbb_schema.sql' if database == 'bbb_graphql' else Path('/usr/share/bbb-container/blocknote_schema.sql')
            sql = schema_contents(path)
            if psql(database, 'SELECT digest FROM public.bigbluebutton_schema') != hashlib.sha256(sql.encode()).hexdigest():
                raise RuntimeError(f'{database}: schema does not match this image; explicit migration required')
            return
        time.sleep(5)
    raise RuntimeError(f'{database}: schema initialization did not complete')

def schema():
    for database, path in [
        ('bbb_graphql', SOURCE / 'bbb_schema.sql'),
        ('blocknote_app', Path('/usr/share/bbb-container/blocknote_schema.sql')),
    ]:
        wait_database(database)
        sql = schema_contents(path)
        digest = hashlib.sha256(sql.encode()).hexdigest()
        if psql(database, "SELECT to_regclass('public.bigbluebutton_schema')"):
            if psql(database, 'SELECT digest FROM public.bigbluebutton_schema') != digest:
                raise RuntimeError(f'{database}: source schema changed; drain meetings and perform an explicit migration')
            print(f'{database}: matching schema already initialized', flush=True)
            continue
        if psql(database, "SELECT count(*) FROM pg_tables WHERE schemaname='public'") != '0':
            raise RuntimeError(f'{database}: refusing to initialize an unrecognized existing database')
        target = Path('/tmp') / f'{database}.sql'
        target.write_text(sql + "\nCREATE TABLE public.bigbluebutton_schema (digest text NOT NULL);\n"
                          + f"INSERT INTO public.bigbluebutton_schema VALUES ('{digest}');\n")
        psql(database, file=target)
        print(f'{database}: initialized atomically', flush=True)

def metadata():
    endpoint = os.environ.get('HASURA_ENDPOINT', 'http://hasura:8085').rstrip('/')
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    for _ in range(180):
        try:
            with opener.open(endpoint + '/healthz', timeout=3) as response:
                if response.status == 200:
                    break
        except (OSError, TimeoutError):
            pass
        time.sleep(5)
    else:
        raise RuntimeError('Hasura did not become ready')
    dest = Path('/tmp/hasura-project')
    shutil.copytree(SOURCE, dest, dirs_exist_ok=True)
    file = dest / 'metadata/databases/databases.yaml'
    databases = yaml.safe_load(file.read_text())
    for database in databases:
        if database['name'] == 'BigBlueButton':
            database['configuration']['connection_info']['database_url'] = {'from_env': 'BBB_DATABASE_URL'}
    file.write_text(yaml.safe_dump(databases, sort_keys=False))
    config = yaml.safe_load((dest / 'config.yaml').read_text())
    config.update(endpoint=endpoint, admin_secret=os.environ['HASURA_SECRET'])
    (dest / 'config.yaml').write_text(yaml.safe_dump(config))
    for arguments in [('metadata', 'apply'), ('metadata', 'inconsistency', 'status')]:
        subprocess.run(['hasura', *arguments, '--project', str(dest), '--skip-update-check'], check=True)

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('operation', choices=['all', 'schema', 'metadata', 'bbb_graphql', 'blocknote_app'],
                    help='initialize fresh schemas/metadata, or wait for a matching workload schema')
mode = parser.parse_args().operation
if mode == 'all':
    schema()
    metadata()
elif mode == 'schema':
    schema()
elif mode == 'metadata':
    metadata()
elif mode in ('bbb_graphql', 'blocknote_app'):
    schema_ready(mode)
else:
    raise ValueError(mode)
