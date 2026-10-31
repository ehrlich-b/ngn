"""Run the width probe and symmetric A/A before the frozen measurement."""
import importlib.util
import json
from pathlib import Path
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session87'


def main():
    subprocess.run(['python3',str(D/'width-probe.py')],check=True)
    spec = importlib.util.spec_from_file_location('queue',D/'campaign_queue.py')
    queue = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(queue)
    aa = json.loads((D/'width-probe.json').read_text())['aa']
    result = queue.run(aa,R/'session80/controls.json',R/'status.jsonl')
    assert result['verdict'] == 'AA_PASS' and not result['faults'], 'Width A/A failed; no measurement'


if __name__ == '__main__':
    main()
