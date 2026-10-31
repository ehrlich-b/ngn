"""One controlled target-blend cell, after the exclusive speed queue shelves.

Reuse the owned, tested trainer and final115/export/parity procedure. Relative
to the completed volume control, only WDL lambda changes from .75 to .50.
"""
import importlib.util
import json
from pathlib import Path

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session102-speed'


def main():
    manifest = json.loads((D/'recipe/wdl050/manifest.json').read_text())
    source = Path(manifest['trainer_source'])
    spec = importlib.util.spec_from_file_location('owned_recipe',R/'session87/train-mixture.py')
    trial = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(trial)
    trial.D = D/'recipe'
    trial.SOURCE = source
    trial.trial('wdl050')


if __name__=='__main__':
    main()
