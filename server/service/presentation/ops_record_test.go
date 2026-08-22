package presentation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

type RecordOps struct {
	mu            sync.Mutex
	trace         []Op
	files         map[string][]byte
	dirs          map[string]bool
	links         map[string]string
	udcs          []string
	unbind        error
	writeFailures map[string][]error
	bound         string
	role          string
	resets        int
}

func NewRecordOps(udcs ...string) *RecordOps {
	if len(udcs) == 0 {
		udcs = []string{dwc2Device}
	}
	return &RecordOps{
		files:         map[string][]byte{},
		dirs:          map[string]bool{},
		links:         map[string]string{},
		udcs:          udcs,
		writeFailures: map[string][]error{},
	}
}

func (r *RecordOps) Trace() []Op {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Op(nil), r.trace...)
}

func (r *RecordOps) Links() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	links := make(map[string]string, len(r.links))
	for link, target := range r.links {
		links[link] = target
	}
	return links
}

func (r *RecordOps) Bound() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bound
}

func (r *RecordOps) Role() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.role
}

func (r *RecordOps) PHYResets() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resets
}

func (r *RecordOps) SetUDC(names ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.udcs = names
}

func (r *RecordOps) Seed(rel string, data []byte) error {
	if err := validateRel(rel); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.files[rel] = append([]byte(nil), data...)
	return nil
}

func (r *RecordOps) record(op Op) {
	r.trace = append(r.trace, op)
}

func (r *RecordOps) Mkdir(rel string) error {
	if err := validateRel(rel); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.record(Op{Kind: OpMkdir, Path: rel})
	r.dirs[rel] = true
	return nil
}

func (r *RecordOps) WriteFile(rel string, data []byte) error {
	if err := validateRel(rel); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if failures := r.writeFailures[rel]; len(failures) != 0 {
		err := failures[0]
		r.writeFailures[rel] = failures[1:]
		return err
	}
	stored := append([]byte(nil), data...)
	r.record(Op{Kind: OpWrite, Path: rel, Data: stored})
	r.files[rel] = stored
	return nil
}

func (r *RecordOps) FailWriteOnce(rel string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writeFailures[rel] = append(r.writeFailures[rel], err)
}

func (r *RecordOps) ReadFile(rel string) ([]byte, error) {
	if err := validateRel(rel); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	data, ok := r.files[rel]
	if !ok {
		for link, target := range r.links {
			prefix := link + "/"
			if strings.HasPrefix(rel, prefix) {
				data, ok = r.files[target+"/"+strings.TrimPrefix(rel, prefix)]
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("read %s: %w", rel, os.ErrNotExist)
	}
	return append([]byte(nil), data...), nil
}

func (r *RecordOps) Symlink(target, linkRel string) error {
	if err := validateRel(linkRel); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.record(Op{Kind: OpSymlink, Path: linkRel, Target: target})
	r.links[linkRel] = target
	return nil
}

func (r *RecordOps) Remove(rel string) error {
	if err := validateRemove(rel); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.record(Op{Kind: OpUnlink, Path: rel})
	delete(r.links, rel)
	return nil
}

func (r *RecordOps) ListUDC() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.udcs) != 1 {
		return nil, fmt.Errorf("%w: recorder holds %d entries %v", ErrUDCCount, len(r.udcs), r.udcs)
	}
	return append([]string(nil), r.udcs...), nil
}

func (r *RecordOps) BindUDC(name string) error {
	if name == "" {
		return fmt.Errorf("%w: an empty name is an unbind, not a bind", ErrUDCName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.record(Op{Kind: OpBind, Path: udcAttr, Data: []byte(name)})
	r.files[udcAttr] = []byte(name)
	r.bound = name
	return nil
}

func (r *RecordOps) FailUnbind(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unbind = err
}

func (r *RecordOps) UnbindUDC() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.unbind != nil {
		return r.unbind
	}
	r.record(Op{Kind: OpUnbind, Path: udcAttr})
	r.files[udcAttr] = []byte(emptyUDCName)
	r.bound = ""
	return nil
}

func (r *RecordOps) SetOTGRole(role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.record(Op{Kind: OpOTGRole, Data: []byte(role)})
	r.role = role
	return nil
}

func (r *RecordOps) ResetPHY(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("reset phy: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.resets++
	return nil
}
