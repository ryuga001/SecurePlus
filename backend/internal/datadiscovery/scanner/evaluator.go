package scanner

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"dpdp-backend/internal/db"
	dto "dpdp-backend/internal/delivery/dto/evaluation"
	"dpdp-backend/internal/delivery/services/inspection"
	"dpdp-backend/internal/delivery/services/policy"
	"dpdp-backend/internal/delivery/utils"
)

const (
	RegexOverlap         = 4 << 10
	MaxKeywordBytes      = 64 << 10
	MaxRegexInstructions = 64 << 10
	MaxSampleOffsets     = 5

	normalizePiece = 64 << 10
	maxPending     = 1 << 10
)

var (
	ErrNoRules       = errors.New("policy has no evaluable rules")
	ErrRulesTooLarge = errors.New("rule set exceeds scanner limits")
	ErrEvaluation    = errors.New("rule evaluation failed")
)

type ruleRef struct {
	id   int
	name string
	kind string
}

type Findings struct {
	Total     int64
	Extracted int64
	Items     db.FileFindings
}

type Evaluator struct {
	rules       []ruleRef
	automaton   dto.Automaton
	keywordRule []int
	keywordLen  []int
	regexes     []*regexp.Regexp
	regexRule   []int
	shards      [][]int
	chunkSize   int
	workers     int

	tasks chan func()
	wg    sync.WaitGroup
	once  sync.Once
	pool  sync.Pool
}

func NewEvaluator(records []dto.RuleRecord, chunkSize, workers int) (*Evaluator, error) {
	if err := checkRuleLimits(records); err != nil {
		return nil, err
	}

	compiled, err := policy.NewCompiler(len(records)).Build(dto.PolicySet{Rules: records})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRulesTooLarge, err)
	}

	workers = max(workers, 1)

	e := &Evaluator{chunkSize: max(chunkSize, RegexOverlap), workers: workers}

	for _, keyword := range compiled.Rules.Keywords {
		e.keywordRule = append(e.keywordRule, len(e.rules))
		e.keywordLen = append(e.keywordLen, len(keyword.Normalized))
		e.rules = append(e.rules, ruleRef{id: keyword.Rule.RuleID, name: keyword.Rule.RuleName, kind: keyword.Rule.RuleType})
	}

	if len(compiled.Rules.Keywords) > 0 {
		e.automaton = compiled.Rules.Automaton
	}

	for _, regex := range compiled.Rules.Regexes {
		e.regexes = append(e.regexes, regex.Pattern)
		e.regexRule = append(e.regexRule, len(e.rules))
		e.rules = append(e.rules, ruleRef{id: regex.Rule.RuleID, name: regex.Rule.RuleName, kind: regex.Rule.RuleType})
	}

	if len(e.rules) == 0 {
		return nil, ErrNoRules
	}

	e.shards = shardRegexes(len(e.regexes), workers, e.automaton != nil)

	e.pool.New = func() any { return e.newFileScan() }

	if workers > 1 {
		e.tasks = make(chan func())

		for range workers {
			e.wg.Add(1)

			go func() {
				defer e.wg.Done()

				for task := range e.tasks {
					task()
				}
			}()
		}
	}

	return e, nil
}

func (e *Evaluator) Close() {
	e.once.Do(func() {
		if e.tasks != nil {
			close(e.tasks)
			e.wg.Wait()
		}
	})
}

func (e *Evaluator) Workers() int {
	return e.workers
}

func (e *Evaluator) Begin(ctx context.Context) *FileScan {
	scan := e.pool.Get().(*FileScan)
	scan.reset(ctx)

	return scan
}

func (e *Evaluator) run(tasks []func() error) error {
	errs := make([]error, len(tasks))

	guarded := func(index int) {
		defer func() {
			if recovered := recover(); recovered != nil {
				errs[index] = fmt.Errorf("%w: %v", ErrEvaluation, recovered)
			}
		}()

		errs[index] = tasks[index]()
	}

	if e.tasks == nil || len(tasks) == 1 {
		for index := range tasks {
			guarded(index)
		}

		return errors.Join(errs...)
	}

	var wg sync.WaitGroup

	wg.Add(len(tasks))

	for index := range tasks {
		e.tasks <- func() {
			defer wg.Done()

			guarded(index)
		}
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (e *Evaluator) newFileScan() *FileScan {
	samples := make([][]int64, len(e.rules))
	for index := range samples {
		samples[index] = make([]int64, 0, MaxSampleOffsets)
	}

	return &FileScan{
		e:       e,
		window:  make([]byte, 0, e.chunkSize+RegexOverlap),
		join:    make([]byte, 0, normalizePiece+maxPending),
		pending: make([]byte, 0, maxPending),
		counts:  make([]int64, len(e.rules)),
		samples: samples,
		lastEnd: make([]int64, len(e.regexes)),
	}
}

type FileScan struct {
	e   *Evaluator
	ctx context.Context

	window []byte
	tail   int
	base   int64

	state      int
	join       []byte
	pending    []byte
	normalized []byte
	lowered    []byte

	counts  []int64
	samples [][]int64
	lastEnd []int64

	err error
}

func (f *FileScan) reset(ctx context.Context) {
	f.ctx = ctx
	f.window = f.window[:0]
	f.tail = 0
	f.base = 0
	f.state = 0
	f.pending = f.pending[:0]
	f.err = nil

	clear(f.counts)
	clear(f.lastEnd)

	for index := range f.samples {
		f.samples[index] = f.samples[index][:0]
	}
}

func (f *FileScan) Write(p []byte) (int, error) {
	if f.err != nil {
		return 0, f.err
	}

	written := 0

	for len(p) > 0 {
		n := min(cap(f.window)-len(f.window), len(p))
		f.window = append(f.window, p[:n]...)
		p = p[n:]
		written += n

		if len(f.window) == cap(f.window) {
			if err := f.evaluate(false); err != nil {
				f.err = err

				return written, err
			}
		}
	}

	return written, nil
}

func (f *FileScan) Finish() (Findings, error) {
	if f.err == nil {
		f.err = f.evaluate(true)
	}

	if f.err != nil {
		return Findings{}, f.err
	}

	findings := Findings{Extracted: f.base + int64(len(f.window))}

	for index, count := range f.counts {
		if count == 0 {
			continue
		}

		rule := f.e.rules[index]
		findings.Total += count
		findings.Items = append(findings.Items, db.FileFinding{
			RuleID:   rule.id,
			RuleName: rule.name,
			RuleType: rule.kind,
			Count:    count,
			Offsets:  append([]int64(nil), f.samples[index]...),
		})
	}

	return findings, nil
}

func (f *FileScan) Release() {
	f.ctx = nil
	f.e.pool.Put(f)
}

func (f *FileScan) evaluate(final bool) error {
	if err := f.ctx.Err(); err != nil {
		return err
	}

	window := f.window
	fresh := window[f.tail:]
	freshBase := f.base + int64(f.tail)

	commit := len(window)
	if !final {
		commit = max(len(window)-RegexOverlap, 0)
	}

	tasks := make([]func() error, 0, len(f.e.shards)+1)

	if f.e.automaton != nil {
		tasks = append(tasks, func() error {
			return f.scanKeywords(fresh, freshBase, final)
		})
	}

	for _, shard := range f.e.shards {
		tasks = append(tasks, func() error {
			return f.scanRegexes(shard, window, commit)
		})
	}

	if err := f.e.run(tasks); err != nil {
		if ctxErr := f.ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		return err
	}

	if final {
		return nil
	}

	keep := len(window) - commit
	copy(f.window, window[commit:])
	f.window = f.window[:keep]
	f.base += int64(commit)
	f.tail = keep

	return nil
}

func (f *FileScan) scanKeywords(fresh []byte, freshBase int64, final bool) error {
	for len(fresh) > 0 || (final && len(f.pending) > 0) {
		if err := f.ctx.Err(); err != nil {
			return err
		}

		n := min(len(fresh), normalizePiece)
		last := n == len(fresh)

		inputBase := freshBase - int64(len(f.pending))
		f.join = append(append(f.join[:0], f.pending...), fresh[:n]...)
		fresh = fresh[n:]
		freshBase += int64(n)

		cut := safeCut(f.join, final && last)

		f.normalized = norm.NFKC.Append(f.normalized[:0], f.join[:cut]...)
		f.lowered = appendLower(f.lowered[:0], f.normalized)

		exact := len(f.lowered) == cut
		lowered := f.lowered

		f.state = f.e.automaton.Scan(f.state, lowered, func(pattern, end int) {
			offset := inputBase
			if exact {
				offset += int64(end - f.e.keywordLen[pattern])
			}

			f.record(f.e.keywordRule[pattern], offset)
		})

		f.pending = append(f.pending[:0], f.join[cut:]...)

		if last {
			break
		}
	}

	return nil
}

func (f *FileScan) scanRegexes(shard []int, window []byte, commit int) error {
	for _, index := range shard {
		if err := f.ctx.Err(); err != nil {
			return err
		}

		pattern := f.e.regexes[index]
		rule := f.e.regexRule[index]

		start := 0
		if skip := f.lastEnd[index] - f.base; skip > 0 {
			start = int(min(skip, int64(len(window))))
		}

		for start <= len(window) {
			location := pattern.FindIndex(window[start:])
			if location == nil {
				break
			}

			matchStart, matchEnd := start+location[0], start+location[1]
			if matchStart >= commit {
				break
			}

			if matchEnd == matchStart {
				if matchEnd >= len(window) {
					break
				}

				_, size := utf8.DecodeRune(window[matchEnd:])
				start = matchEnd + max(size, 1)

				continue
			}

			f.record(rule, f.base+int64(matchStart))
			f.lastEnd[index] = f.base + int64(matchEnd)
			start = matchEnd
		}
	}

	return nil
}

func (f *FileScan) record(rule int, offset int64) {
	f.counts[rule]++

	if len(f.samples[rule]) < MaxSampleOffsets {
		f.samples[rule] = append(f.samples[rule], offset)
	}
}

func safeCut(input []byte, final bool) int {
	if final {
		return len(input)
	}

	cut := len(input)

	if start := lastRuneStart(input); start >= 0 && !utf8.FullRune(input[start:]) {
		cut = start
	}

	if boundary := norm.NFKC.LastBoundary(input[:cut]); boundary >= 0 {
		cut = boundary
	}

	if len(input)-cut > maxPending {
		return len(input)
	}

	return cut
}

func lastRuneStart(input []byte) int {
	for index := len(input) - 1; index >= 0 && index >= len(input)-utf8.UTFMax; index-- {
		if utf8.RuneStart(input[index]) {
			return index
		}
	}

	return -1
}

func appendLower(dst, src []byte) []byte {
	for index := 0; index < len(src); {
		character := src[index]

		if character < utf8.RuneSelf {
			if 'A' <= character && character <= 'Z' {
				character += 'a' - 'A'
			}

			dst = append(dst, character)
			index++

			continue
		}

		r, size := utf8.DecodeRune(src[index:])
		if r == utf8.RuneError && size <= 1 {
			dst = append(dst, character)
			index++

			continue
		}

		dst = utf8.AppendRune(dst, unicode.ToLower(r))
		index += size
	}

	return dst
}

func checkRuleLimits(records []dto.RuleRecord) error {
	keywordBytes := 0
	instructions := 0

	for _, record := range records {
		switch record.RuleType {
		case utils.MatcherKeyword:
			keywordBytes += len(inspection.Normalize(record.RuleValue))
		case utils.MatcherRegex:
			instructions += regexInstructions(record.RuleValue)
		}
	}

	if keywordBytes > MaxKeywordBytes || instructions > MaxRegexInstructions {
		return ErrRulesTooLarge
	}

	return nil
}

func regexInstructions(pattern string) int {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return 0
	}

	program, err := syntax.Compile(parsed.Simplify())
	if err != nil {
		return 0
	}

	return len(program.Inst)
}

func shardRegexes(count, workers int, keywords bool) [][]int {
	if count == 0 {
		return nil
	}

	slots := workers
	if keywords {
		slots--
	}

	slots = min(max(slots, 1), count)

	shards := make([][]int, slots)
	for index := range count {
		shards[index%slots] = append(shards[index%slots], index)
	}

	return shards
}
