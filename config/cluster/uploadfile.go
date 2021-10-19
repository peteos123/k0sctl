package cluster

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// UploadFile describes a file to be uploaded for the host
type UploadFile struct {
	Name            string      `yaml:"name,omitempty"`
	Source          string      `yaml:"src" validate:"required"`
	DestinationDir  string      `yaml:"dstDir" validate:"required"`
	DestinationFile string      `yaml:"dst"`
	PermMode        interface{} `yaml:"perm"`
	PermString      string      `yaml:"-"`
	LocalPermString string      `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (u *UploadFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type uploadFile UploadFile
	yu := (*uploadFile)(u)

	if err := unmarshal(yu); err != nil {
		return err
	}

	switch t := u.PermMode.(type) {
	case int:
		if t < 0 {
			return fmt.Errorf("invalid uploadFile permission: %d: must be a positive value", t)
		}
		if t == 0 {
			return fmt.Errorf("invalid nil uploadFile permission")
		}
		u.PermString = fmt.Sprintf("%#o", t)
	case string:
		u.PermString = t
	default:
		u.PermString = ""
	}

	for i, c := range u.PermString {
		n, err := strconv.Atoi(string(c))
		if err != nil {
			return fmt.Errorf("failed to parse uploadFile permission %s: %w", u.PermString, err)
		}

		// These could catch some weird octal conversion mistakes
		if i == 1 && n < 4 {
			return fmt.Errorf("invalid uploadFile permission %s: owner would have unconventional access", u.PermString)
		}
		if n > 7 {
			return fmt.Errorf("invalid uploadFile permission %s: octal value can't have numbers over 7", u.PermString)
		}
	}

	return nil
}

func (u UploadFile) String() string {
	if u.Name == "" {
		return u.Source
	}
	return u.Name
}

// isGlob returns true if the string has glob characters. Some of the characters are probably not going
// to work as expected, as go's glob won't do everything a shell does.
func isGlob(s string) bool {
	return strings.ContainsAny(s, "*%?[]{}")
}

// Resolve returns a slice of UploadFiles that were found using the glob pattern or a slice
// containing the single UploadFile if it was absolute
func (u UploadFile) Resolve() ([]UploadFile, error) {
	var files []UploadFile
	if u.IsURL() {
		files = append(files, u)
		return files, nil
	}

	if isGlob(u.Source) {
		return u.glob(u.Source)
	}

	stat, err := os.Stat(u.Source)
	if err != nil {
		return files, err
	}

	if stat.IsDir() {
		return u.glob(path.Join(u.Source, "**/*"))
	}

	// it is a single file, capture the perms return self inside a slice
	u.LocalPermString = fmt.Sprintf("%#o", stat.Mode().Perm())
	return append(files, u), nil
}

func (u UploadFile) glob(src string) ([]UploadFile, error) {
	var files []UploadFile

	base, pattern := doublestar.SplitPattern(src)
	fsys := os.DirFS(base)
	sources, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return nil, err
	}

	if len(sources) > 1 && u.DestinationFile != "" {
		return files, fmt.Errorf("multiple files found for '%s' but no destination directory (dstDir) set", u)
	}

	for i, s := range sources {
		name := u.Name
		if len(sources) > 1 {
			name = fmt.Sprintf("%s: %s (%d of %d)", u.Name, s, i+1, len(sources))
		}

		stat, err := os.Stat(s)
		if err != nil {
			return files, err
		}

		files = append(files, UploadFile{
			Name:            name,
			Source:          s,
			DestinationDir:  path.Join(u.DestinationDir, strings.TrimPrefix(s, base)),
			DestinationFile: path.Base(s),
			PermMode:        u.PermMode,
			PermString:      u.PermString,
			LocalPermString: fmt.Sprintf("%#o", stat.Mode().Perm()),
		})
	}

	return files, nil
}

// IsURL returns true if the source is a URL
func (u UploadFile) IsURL() bool {
	return strings.Contains(u.Source, "://")
}

// Destination returns the target path and filename or an error if one couldn't be determined
func (u UploadFile) Destination() (string, string, error) {
	if u.DestinationDir != "" && u.DestinationFile != "" {
		return u.DestinationDir, u.DestinationFile, nil
	}

	if u.DestinationFile == "" {
		return u.DestinationDir, path.Base(u.Source), nil
	}

	return path.Dir(u.DestinationFile), path.Base(u.DestinationFile), nil
}
