"""Compare NGN source with pinned references; never use references to generate code.

Run with a WSL `go list -deps -json .` receipt, a Git revision and a local
reference directory containing manifest.json and family/source paths.
"""
import argparse
import difflib
import json
import pathlib
import re
import subprocess

LEX = re.compile(r'//[^\n]*|/\*.*?\*/|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'|`[^`]*`|'
                 r'0[xX][0-9a-fA-F]+|\d+(?:\.\d+)?|[A-Za-z_]\w*|'
                 r'<<=|>>=|:=|==|!=|<=|>=|\+\+|--|&&|\|\||<<|>>|&\^|\+=|-=|\*=|/=|[^\s]', re.S)
KEYWORDS = set('break case chan const continue default defer else fallthrough for func go goto '
               'if import interface map package range return select struct switch type var '
               'true false nil do while class enum namespace template public private protected '
               'static inline constexpr sizeof auto void unsigned signed union'.split())

def tokenize(source):
    tokens, lines = [], []
    line, previous = 1, 0
    for match in LEX.finditer(source):
        line += source.count('\n', previous, match.start())
        previous = match.start()
        value = match.group()
        if value.startswith(('//', '/*')):
            continue
        if re.fullmatch(r'[A-Za-z_]\w*', value) and value not in KEYWORDS:
            value = '_'
        elif re.fullmatch(r'0[xX][0-9a-fA-F]+|\d+', value):
            value = str(int(value, 16 if value.lower().startswith('0x') else 10))
        tokens.append(value)
        lines.append(line)
    return tokens, lines

def code_lines(source):
    # Preserve identifiers for the line comparison, and strip lexical comments.
    source = LEX.sub(lambda m: '' if m.group().startswith(('//', '/*')) else m.group(), source)
    return [re.sub(r'\s+', '', line) for line in source.splitlines() if line.strip()]

class Substrings:
    """A suffix automaton measures every exact token run in linear scan time."""
    def __init__(self, tokens):
        self.states = [dict(next={}, link=-1, length=0, end=-1)]
        last = 0
        for index, token in enumerate(tokens):
            new = len(self.states)
            self.states.append(dict(next={}, link=0, length=self.states[last]['length']+1, end=index))
            p = last
            while p >= 0 and token not in self.states[p]['next']:
                self.states[p]['next'][token] = new
                p = self.states[p]['link']
            if p >= 0:
                q = self.states[p]['next'][token]
                if self.states[p]['length']+1 == self.states[q]['length']:
                    self.states[new]['link'] = q
                else:
                    clone = len(self.states)
                    self.states.append(dict(next=dict(self.states[q]['next']), link=self.states[q]['link'],
                                            length=self.states[p]['length']+1, end=self.states[q]['end']))
                    while p >= 0 and self.states[p]['next'].get(token) == q:
                        self.states[p]['next'][token] = clone
                        p = self.states[p]['link']
                    self.states[q]['link'] = self.states[new]['link'] = clone
            last = new

    def measure(self, tokens):
        state = length = longest = end = ref_end = 0
        changes = [0] * (len(tokens)+1)
        for index, token in enumerate(tokens):
            while state and token not in self.states[state]['next']:
                state = self.states[state]['link']
                length = min(length, self.states[state]['length'])
            if token in self.states[state]['next']:
                state = self.states[state]['next'][token]
                length += 1
            else:
                length = 0
            if length > longest:
                longest, end, ref_end = length, index, self.states[state]['end']
            if length >= 20:
                changes[index-length+1] += 1
                changes[index+1] -= 1
        active = 0
        covered = []
        for change in changes[:-1]:
            active += change
            covered.append(active > 0)
        return longest, end, ref_end, covered

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--deps', required=True)
    parser.add_argument('--revision', default='indep/main')
    parser.add_argument('--references', required=True)
    parser.add_argument('--out', required=True)
    args = parser.parse_args()
    root = pathlib.Path(args.references)
    manifest = json.loads((root/'manifest.json').read_text())
    decoder = json.JSONDecoder()
    receipt = pathlib.Path(args.deps).read_text().strip()
    files = []
    while receipt:
        package, consumed = decoder.raw_decode(receipt)
        receipt = receipt[consumed:].lstrip()
        name = package['ImportPath']
        if name == 'github.com/ehrlich-b/ngn' or name.startswith('github.com/ehrlich-b/ngn/'):
            directory = name.removeprefix('github.com/ehrlich-b/ngn').lstrip('/')
            files.extend(str(pathlib.PurePosixPath(directory)/f) for f in package.get('GoFiles', []))
    # Matching core files get Zahak comparisons; the common search/runtime corpus
    # also checks auxiliary engine files for cross-file similarities.
    selected = []
    for family in manifest:
        for name in manifest[family]['files']:
            if family == 'zahak' and (name.startswith('engine/') or name == 'search/search.go'):
                selected.append((family, name))
            elif family.startswith('counter') and any(part in name.split('/') for part in ['engine', 'common', 'eval', 'evalpesto']):
                selected.append((family, name))
            elif family in ['stash', 'stockfish'] and pathlib.PurePosixPath(name).stem in ['search', 'timeman', 'movepick', 'history', 'evaluate', 'bitboard']:
                selected.append((family, name))
            elif family == 'disservin':
                selected.append((family, name))
    refs = []
    for family, name in selected:
        source = (root/family/name).read_text(errors='replace')
        tokens, lines = tokenize(source)
        refs.append((family+'/'+name, tokens, lines, Substrings(tokens), code_lines(source)))
    rows = []
    for file in sorted(files):
        if args.revision == 'working':
            source = pathlib.Path(file).read_text()
        else:
            source = subprocess.check_output(['git', 'show', args.revision+':'+file], text=True)
        tokens, line_numbers = tokenize(source)
        lines = code_lines(source)
        token_best = (0, '', 0, 0, 0)
        line_best = (0, '', 0.0)
        coverage = [False]*len(tokens)
        for name, rt, rl, automaton, ref_lines in refs:
            run, end, ref_end, covered = automaton.measure(tokens)
            coverage = [a or b for a, b in zip(coverage, covered)]
            if run > token_best[0]:
                token_best = (run, name, line_numbers[end-run+1], line_numbers[end], rl[ref_end-run+1])
            matcher = difflib.SequenceMatcher(None, lines, ref_lines, autojunk=False)
            blocks = matcher.get_matching_blocks()
            longest = max(b.size for b in blocks)
            fraction = 100*sum(b.size for b in blocks)/max(1, len(lines))
            if (longest, fraction) > (line_best[0], line_best[2]):
                line_best = (longest, name, fraction)
        rows.append(dict(file=file, tokens=len(tokens), longest_tokens=token_best[0],
                         token_coverage_percent=round(100*sum(coverage)/max(1,len(tokens)),2),
                         token_reference=token_best[1], ngn_start_line=token_best[2],
                         ngn_end_line=token_best[3], reference_start_line=token_best[4],
                         longest_lines=line_best[0], line_reference=line_best[1],
                         line_coverage_percent=round(line_best[2],2)))
        print(file, token_best[:4], rows[-1]['token_coverage_percent'], flush=True)
    pathlib.Path(args.out).write_text(json.dumps(dict(revision=args.revision, references=manifest,
        method='Identifiers normalized, keywords/operators/literals retained; integers canonicalized. '
        'Token coverage is the union of ALL exact runs >=20 across the corpus; line comparison '
        'retains identifiers, strips whitespace/comments, SequenceMatcher autojunk=False.', files=rows), indent=2)+'\n')

if __name__ == '__main__':
    main()
