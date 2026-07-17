import os
import glob
import re

for f in glob.glob('docs/architecture/*.md'):
    with open(f, 'r', encoding='utf-8') as file:
        content = file.read()
    
    # We look for blocks like:
    # ```
    # A
    # 
    # ↓
    # 
    # B
    # ```
    
    def replacer(match):
        block = match.group(1)
        if '↓' not in block:
            return match.group(0)
        
        lines = [line.strip() for line in block.split('\n') if line.strip()]
        
        # Filter out arrows and construct mermaid
        nodes = []
        for line in lines:
            if line == '↓' or not line:
                continue
            nodes.append(line)
        
        mermaid = '```mermaid\ngraph TD;\n'
        for i in range(len(nodes) - 1):
            n1 = nodes[i].replace('"', '')
            n2 = nodes[i+1].replace('"', '')
            mermaid += f'    "{n1}" --> "{n2}";\n'
        mermaid += '```'
        return mermaid
        
    # Match any code block
    new_content = re.sub(r'```[a-zA-Z]*\r?\n(.*?)\r?\n```', replacer, content, flags=re.DOTALL)
    
    if new_content != content:
        with open(f, 'w', encoding='utf-8') as file:
            file.write(new_content)
        print(f'Updated {f}')
