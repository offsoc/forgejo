# Copyright 2025 The Forgejo Authors c/o Codeberg e.V. All rights reserved.
# SPDX-License-Identifier: GPL-3.0-or-later

import sys
import tomllib
import json

jsonEncoder = json.JSONEncoder()

def format_key_value(key, value):
    if value is None:
        return f'{key} ='
    if type(value) is str:
        return f'{key} = {value}'
    else:
        return f'{key} = {jsonEncoder.encode(value)}'


def convert_to_ini(t, key=None, section=None, file=None):
    def _print(*args):
        print(*args, file=file)
    is_section = any(type(_t) is dict for _t in t.values())
    description = t.get('description')
    if is_section:
        _print(';;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;')
    if description:
        _print(f";; {description.strip('\n').replace('\n', '\n;; ')}")
        if is_section:
            _print(';;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;')
    if is_section:
        if section is not None:
            _print(f';[{section}]')
        _print(';;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;')
        for k, _t in t.items():
            if type(_t) is not dict:
                continue
            convert_to_ini(_t,
                           key=k,
                           section=k if key is None else f'{key}.{k}',
                           file=file)
    else:
        default = t.get('default')
        _print(f';{format_key_value(key, default)}\n;;')


CONF_VARIABLES = ['AppPath', 'AppWorkPath', 'CustomPath', 'CustomConf', 'StaticRootPath']


def convert_to_md(t, key=None, keys=[], headingNum=1,
                  anchors={v: v for v in CONF_VARIABLES},
                  file=None):
    def _print(*args):
        print(*args, file=file)
    is_section = any(type(_t) is dict for _t in t.values())
    description = t.get('description')
    section = '.'.join(keys)
    if description:
        description = description.strip()
        if '## Default configuration' in description:
            for v in CONF_VARIABLES:
                description = description.replace(f'\n- _`{v}`_',
                        f'\n- <a name="{v}" href="#{v}">_`{v}`_</a>')
        else:
            for variable, anchor in anchors.items():
                description = description.replace(f'`{variable}`', f'<a href="#{anchor}">`{variable}`</a>')
                description = description.replace(f'`[{variable}]`', f'<a href="#{anchor}">`[{variable}]`</a>')
    if is_section:
        if key is not None and len(t) > 1:
            heading = t.get('heading')
            if heading is None:
                heading = section.replace('-', ' ').replace('_', ' ').replace('.', ' ').capitalize()
            _print(f'\n{"#" * headingNum} <a name="{section}" href="#{section}">{heading}</a>')
            if description:
                _print(description)
            _print(f"\n```ini\n[{section}]\n```")
        elif description:
            _print(description)
        anchors_parallel = {'.'.join(keys[i:] + [k]): '.'.join(keys + [k])
                            for i in range(len(keys) + 1)
                            for k, _t in t.items() if type(_t) is dict}
        anchors.update(anchors_parallel)
        for k, _t in t.items():
            if type(_t) is not dict:
                continue
            convert_to_md(_t,
                          key=k,
                          keys=keys + [k],
                          headingNum=headingNum + 1 if len(t) > 1 else headingNum,
                          anchors=anchors,
                          file=file)
    else:
        _print(f'\n- <a name="{section}" href="#{section}">`{section}`</a>:')
        if description:
            _print(f"  {'\n  '.join(description.split('\n'))}:")
        default = t.get('default')
        _print(f'  ```ini\n  {format_key_value(key, default)}\n  ```')


if __name__ == '__main__':
    with open('options/setting/config.toml', 'rb') as f:
        t = tomllib.load(f)

    with open('custom/conf/app.example.ini', 'w') as f:
        convert_to_ini(t, file=f)

    with open('options/setting/config-cheat-sheet.md', 'w') as f:
        f.write('''---
title: 'Configuration Cheat Sheet'
license: 'Apache-2.0'
origin_url: 'https://github.com/go-gitea/gitea/blob/e865de1e9d65dc09797d165a51c8e705d2a86030/docs/content/administration/config-cheat-sheet.en-us.md' 
---\n\n''')
        convert_to_md(t, file=f)
