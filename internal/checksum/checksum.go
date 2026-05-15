package checksum

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

type SumEntry struct {
	Filename string `json:"filename"`
	SHA256   string `json:"sha256"`
}

func FileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ParseSHA256Sums(r io.Reader) ([]SumEntry, error) {
	scanner := bufio.NewScanner(r)
	var entries []SumEntry
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("line %d: expected sha256 and filename", lineNumber)
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != sha256.Size*2 {
			return nil, fmt.Errorf("line %d: invalid sha256 length", lineNumber)
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return nil, fmt.Errorf("line %d: invalid sha256: %w", lineNumber, err)
		}
		filename := strings.TrimPrefix(strings.Join(fields[1:], " "), "*")
		filename = strings.TrimSpace(filename)
		if filename == "" {
			return nil, fmt.Errorf("line %d: empty filename", lineNumber)
		}
		entries = append(entries, SumEntry{Filename: filename, SHA256: sum})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
