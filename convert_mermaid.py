import os
import glob
import re

for f in glob.glob('docs/architecture/*.md'):
    with open(f, 'r', encoding='utf-8') as file:
        content = file.read()
    
    # Replace Draft v1.0 with Approved
    content = content.replace('Status:** Draft v1.0', 'Status:** Approved')
    
    def replacer(match):
        block = match.group(1)
        if '↓' not in block:
            return match.group(0)
        
        lines = [line.strip() for line in block.split('\n') if line.strip()]
        
        nodes = []
        for line in lines:
            if line == '↓':
                continue
            nodes.append(line)
        
        mermaid = '```mermaid\ngraph TD;\n'
        for i in range(len(nodes) - 1):
            n1 = nodes[i].replace('"', '')
            n2 = nodes[i+1].replace('"', '')
            mermaid += f'    "{n1}" --> "{n2}";\n'
        mermaid += '```'
        return mermaid
        
    new_content = re.sub(r'```(?:text)?\n(.*?)\n```', replacer, content, flags=re.DOTALL)
    
    if new_content != content:
        with open(f, 'w', encoding='utf-8') as file:
            file.write(new_content)
        print(f'Updated {f}')
