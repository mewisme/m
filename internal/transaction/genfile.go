package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

const (
	journalHeadName = "journal.head"
	currentHeadName = "current.head"
)

type generationHead struct {
	Generation uint64 `json:"generation"`
	Checksum   string `json:"checksum"`
}

func checksumHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeGenerationHead(headPath string, gen uint64, data []byte) error {
	head := generationHead{Generation: gen, Checksum: checksumHex(data)}
	raw, err := json.Marshal(head)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.head", headPath, err)
	}
	raw = append(raw, '\n')
	return fsx.PublishFile(headPath, raw, 0o644)
}

func readGenerationHead(headPath string) (generationHead, error) {
	var head generationHead
	data, err := os.ReadFile(headPath)
	if err != nil {
		return head, err
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return head, apperr.Wrap(apperr.IO, "transaction.head", headPath, err)
	}
	return head, nil
}

func loadGenerationFile(path string, wantChecksum string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if wantChecksum != "" && checksumHex(data) != wantChecksum {
		return nil, apperr.New(apperr.Integrity, "transaction.head", path, "checksum mismatch")
	}
	return data, nil
}

func highestGeneration(dir, prefix, suffix string) uint64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var max uint64
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		mid := strings.TrimPrefix(name, prefix)
		mid = strings.TrimSuffix(mid, suffix)
		n, err := strconv.ParseUint(mid, 10, 64)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max
}

func scanGenerations(dir, prefix, suffix string, decode func([]byte) error) (uint64, []byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, nil, err
	}
	type candidate struct {
		gen  uint64
		data []byte
	}
	var cands []candidate
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		mid := strings.TrimPrefix(name, prefix)
		mid = strings.TrimSuffix(mid, suffix)
		gen, err := strconv.ParseUint(mid, 10, 64)
		if err != nil {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if decode != nil {
			if err := decode(data); err != nil {
				continue
			}
		}
		cands = append(cands, candidate{gen: gen, data: data})
	}
	if len(cands) == 0 {
		return 0, nil, os.ErrNotExist
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].gen > cands[j].gen })
	return cands[0].gen, cands[0].data, nil
}

func journalGenPath(root string, gen uint64) string {
	return filepath.Join(root, fmt.Sprintf("journal.%06d.json", gen))
}

func currentGenPath(projectRoot string, gen uint64) string {
	return filepath.Join(TxnRoot(projectRoot), fmt.Sprintf("current.%06d", gen))
}

func saveJournalGeneration(root string, gen int, data []byte) (int, error) {
	next := uint64(1)
	if gen > 0 {
		next = uint64(gen) + 1
	}
	path := journalGenPath(root, next)
	if err := fsx.WriteGenerationExclusive(path, data, 0o644); err != nil {
		return 0, apperr.Wrap(apperr.IO, "transaction.journal", path, err)
	}
	headPath := filepath.Join(root, journalHeadName)
	if err := writeGenerationHead(headPath, next, data); err != nil {
		return 0, err
	}
	return int(next), nil
}

func loadJournalGeneration(root string) (*Document, error) {
	if doc, err := loadJournalFromHead(root); err != nil || doc != nil {
		return doc, err
	}
	return loadJournalLegacy(root)
}

func loadJournalFromHead(root string) (*Document, error) {
	headPath := filepath.Join(root, journalHeadName)
	head, err := readGenerationHead(headPath)
	if err == nil {
		data, err := loadGenerationFile(journalGenPath(root, head.Generation), head.Checksum)
		if err == nil {
			return Decode(data)
		}
	}
	gen, data, err := scanGenerations(root, journalGenPrefix, journalGenSuffix, func(b []byte) error {
		_, decErr := Decode(b)
		return decErr
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "transaction.load", root, err)
	}
	doc, decErr := Decode(data)
	if decErr != nil {
		return nil, decErr
	}
	_ = gen
	return doc, nil
}

const (
	journalGenPrefix = "journal."
	journalGenSuffix = ".json"
)

func loadJournalLegacy(root string) (*Document, error) {
	for _, name := range []string{JournalName, JournalNameV3, JournalNameV2, JournalNameV1} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, apperr.Wrap(apperr.IO, "transaction.load", root, err)
		}
		return Decode(data)
	}
	return nil, nil
}

func currentJournalGen(root string) int {
	return int(highestGeneration(root, journalGenPrefix, journalGenSuffix))
}

func writeCurrentGeneration(projectRoot, id string) error {
	dir := TxnRoot(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.current", dir, err)
	}
	next := highestGeneration(dir, "current.", "") + 1
	body := []byte(id + "\n")
	genPath := currentGenPath(projectRoot, next)
	if err := fsx.PublishFile(genPath, body, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.current", genPath, err)
	}
	headPath := filepath.Join(dir, currentHeadName)
	if err := writeGenerationHead(headPath, next, body); err != nil {
		return err
	}
	return fsx.PublishFile(CurrentPath(projectRoot), body, 0o644)
}

func readCurrentGeneration(projectRoot string) (string, error) {
	dir := TxnRoot(projectRoot)
	headPath := filepath.Join(dir, currentHeadName)
	head, err := readGenerationHead(headPath)
	if err == nil {
		data, err := loadGenerationFile(currentGenPath(projectRoot, head.Generation), head.Checksum)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}
	_, data, err := scanGenerations(dir, "current.", "", nil)
	if err != nil {
		if os.IsNotExist(err) {
			return readCurrentLegacy(projectRoot)
		}
		return "", apperr.Wrap(apperr.IO, "transaction.current", dir, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func readCurrentLegacy(projectRoot string) (string, error) {
	data, err := os.ReadFile(CurrentPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", apperr.Wrap(apperr.IO, "transaction.current", projectRoot, err)
	}
	return strings.TrimSpace(string(data)), nil
}
