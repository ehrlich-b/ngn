"""Post-screen/confirmation failure only: third controlled source-weight cell.

Reuses the tested trainer and final115/parity routine. No corpus reconstruction,
training-duration variant, checkpoint selection, automatic gate or adoption.
"""
import importlib.util
from pathlib import Path

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session94'


def main():
    path = D / 'source/experiments/2026-10-15-session87-train-mixture.py'
    spec = importlib.util.spec_from_file_location('controlled_mixture', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    module.D = D / 'gpu-next'
    module.trial('rebalanced-t74')


if __name__ == '__main__':
    main()
