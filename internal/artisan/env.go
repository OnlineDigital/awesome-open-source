package artisan

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// LoadEnv loads literal dotenv values. Existing process variables win; no shell
// execution or variable interpolation occurs. Secrets are never printed.
func LoadEnv(path string) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		text = strings.TrimPrefix(text, "export ")
		key, value, ok := strings.Cut(text, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || !envName.MatchString(key) {
			return fmt.Errorf("invalid .env entry at line %d", line)
		}
		if len(value) > 0 && (value[0] == '\'' || value[0] == '"') {
			quote := value[0]
			end := strings.LastIndexByte(value, quote)
			if end == 0 {
				return fmt.Errorf("unclosed .env quote at line %d", line)
			}
			tail := strings.TrimSpace(value[end+1:])
			if tail != "" && !strings.HasPrefix(tail, "#") {
				return fmt.Errorf("invalid quoted .env entry at line %d", line)
			}
			quoted := value[:end+1]
			if quote == '"' {
				value, err = strconv.Unquote(quoted)
				if err != nil {
					return fmt.Errorf("invalid .env quote at line %d", line)
				}
			} else {
				value = quoted[1 : len(quoted)-1]
			}
		} else {
			if before, _, ok := strings.Cut(value, " #"); ok {
				value = strings.TrimSpace(before)
			}
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err = os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func generateKey(root string, force bool, out io.Writer) error {
	path := filepath.Join(root, ".env")
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to replace symlink .env")
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data, err = os.ReadFile(filepath.Join(root, ".env.example"))
		if os.IsNotExist(err) {
			data = []byte("APP_ENV=production\n")
			err = nil
		}
	}
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	index := -1
	for n, line := range lines {
		key, value, ok := strings.Cut(strings.TrimPrefix(strings.TrimSpace(line), "export "), "=")
		if ok && strings.TrimSpace(key) == "APP_KEY" {
			if index != -1 {
				return fmt.Errorf("duplicate APP_KEY entries in .env")
			}
			index = n
			value = strings.TrimSpace(value)
			if value != "" && value != "\"\"" && value != "''" && !force {
				return fmt.Errorf("APP_KEY already exists; use --force only to deliberately rotate it")
			}
		}
	}
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return err
	}
	entry := "APP_KEY=" + hex.EncodeToString(key)
	if index == -1 {
		lines = append(lines, entry)
	} else {
		lines[index] = entry
	}
	temp, err := os.CreateTemp(root, ".env-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err = temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err = temp.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "APP_KEY written to .env. Keep this file private and retain the key between deployments.")
	return err
}
