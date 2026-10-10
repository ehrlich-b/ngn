"""Use the established paired terminal parser for this isolated time gate."""
import json
import runpy
from pathlib import Path

namespace = runpy.run_path('paired-parser.py')
result = namespace['result']
result['verdict'] = ('eligible' if result['verdict'] == 'certification-eligible'
                     else 'shelved')
reported_lower = namespace['reported'][1]
if reported_lower <= 0:
    result['verdict'] = 'shelved'
result['decision'] = 'completedH1 AND printed/unrounded lower95>0 AND zero faults/intact inputs'
Path('gate-summary.json').write_text(json.dumps(result, indent=2) + '\n')
