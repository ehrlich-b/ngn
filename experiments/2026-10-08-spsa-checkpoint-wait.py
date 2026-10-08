"""Wait for a durable SPSA checkpoint using Linux notifications, never polling."""
import ctypes
import datetime
import json
import os
from pathlib import Path
import select
import struct
import sys
import time


def wait(path, pid, minimum, timeout):
    libc = ctypes.CDLL(None, use_errno=True)
    fd = libc.inotify_init1(os.O_CLOEXEC | os.O_NONBLOCK)
    if fd < 0:
        raise OSError(ctypes.get_errno(), 'inotify_init1')
    try:
        # Go closes the checkpoint file after encoding; wait for that close.
        mask = 0x8 | 0x400 | 0x800  # CLOSE_WRITE, DELETE_SELF, MOVE_SELF
        if libc.inotify_add_watch(fd, os.fsencode(path), mask) < 0:
            raise OSError(ctypes.get_errno(), 'inotify_add_watch')
        pidfd = os.pidfd_open(pid)
        try:
            poll = select.poll()
            poll.register(fd, select.POLLIN)
            poll.register(pidfd, select.POLLIN)
            deadline = time.monotonic() + timeout
            while True:
                os.kill(pid, 0)
                rows = path.read_text().splitlines()
                last = json.loads(rows[-1]) if rows else {}
                if int(last.get('iter', -1)) >= minimum:
                    return dict(pid=pid, minimum_iteration=minimum, checkpoint=last,
                                mechanism='inotify-close-write+pidfd',
                                utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise TimeoutError('no new durable checkpoint before deadline')
                events = poll.poll(max(1, int(remaining * 1000)))
                if not events:
                    raise TimeoutError('no new durable checkpoint before deadline')
                if any(handle == pidfd for handle, _ in events):
                    raise RuntimeError('tuner wrapper exited before new checkpoint')
                raw = os.read(fd, 65536)
                offset = 0
                while offset < len(raw):
                    _, event_mask, _, name_len = struct.unpack_from('iIII', raw, offset)
                    if event_mask & (0x400 | 0x800 | 0x8000):
                        raise RuntimeError('checkpoint file replaced or removed')
                    offset += 16 + name_len
        finally:
            os.close(pidfd)
    finally:
        os.close(fd)


if __name__ == '__main__':
    print(json.dumps(wait(Path(sys.argv[1]), int(sys.argv[2]), int(sys.argv[3]), float(sys.argv[4]))))
