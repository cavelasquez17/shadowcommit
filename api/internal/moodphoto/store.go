// Package moodphoto stores private, normalized mood-entry images outside the static web root.
package moodphoto

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxUploadBytes    = 5 << 20
	MaxImageDimension = 1280
	maxSourcePixels   = 20_000_000
)

var keyPattern = regexp.MustCompile(`^[a-f0-9]{48}\.(jpg|png)$`)

// Store keeps processed files under a configured, non-public directory.
type Store struct {
	dir string
}

// PendingDelete temporarily hides a photo so a database delete can either commit or restore it.
type PendingDelete struct {
	original string
	staged   string
}

// New returns a store rooted at dir. The directory is created lazily on first write.
func New(dir string) *Store { return &Store{dir: dir} }

// WithExclusive serializes photo writes/deletes with a backup snapshot. The lock is shared across
// processes, so the one-shot backup command and the running web server agree on the same photo set.
func (s *Store) WithExclusive(fn func() error) error {
	if strings.TrimSpace(s.dir) == "" {
		return errors.New("mood photo directory is not configured")
	}
	if err := s.prepareDir(); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(s.dir, ".moodphoto.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open mood photo lock: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock mood photo directory: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) //nolint:errcheck
	return fn()
}

// Save validates, decodes, downsizes, and re-encodes an image. The client filename and
// Content-Type are deliberately not accepted as inputs; the returned key is server-generated.
func (s *Store) Save(src io.Reader) (key, contentType string, err error) {
	if strings.TrimSpace(s.dir) == "" {
		return "", "", errors.New("mood photo directory is not configured")
	}
	raw, err := io.ReadAll(io.LimitReader(src, MaxUploadBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("read image: %w", err)
	}
	if len(raw) > MaxUploadBytes {
		return "", "", errTooLarge
	}

	sniffed := http.DetectContentType(raw)
	var outputExt string
	switch sniffed {
	case "image/jpeg":
		outputExt = ".jpg"
	case "image/png":
		outputExt = ".png"
	case "image/webp":
		outputExt = ".jpg" // WebP is decoded and normalized to JPEG; no metadata is copied.
	default:
		return "", "", errUnsupportedType
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return "", "", fmt.Errorf("decode image header: %w", errInvalidImage)
	}
	if !matchesFormat(sniffed, format) {
		return "", "", errUnsupportedType
	}
	if config.Width < 1 || config.Height < 1 || int64(config.Width) > maxSourcePixels/int64(config.Height) {
		return "", "", errInvalidDimensions
	}

	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil || !matchesFormat(sniffed, format) {
		return "", "", fmt.Errorf("decode image: %w", errInvalidImage)
	}
	normalized := resize(img, MaxImageDimension)
	var encoded bytes.Buffer
	if outputExt == ".png" {
		if err := png.Encode(&encoded, normalized); err != nil {
			return "", "", fmt.Errorf("encode PNG: %w", err)
		}
		contentType = "image/png"
	} else {
		background := image.NewRGBA(normalized.Bounds())
		draw.Draw(background, background.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		draw.Draw(background, background.Bounds(), normalized, normalized.Bounds().Min, draw.Over)
		if err := jpeg.Encode(&encoded, background, &jpeg.Options{Quality: 88}); err != nil {
			return "", "", fmt.Errorf("encode JPEG: %w", err)
		}
		contentType = "image/jpeg"
	}

	if err := s.prepareDir(); err != nil {
		return "", "", err
	}
	key, err = randomKey(outputExt)
	if err != nil {
		return "", "", fmt.Errorf("generate photo key: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(s.dir, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", "", fmt.Errorf("create mood photo: %w", err)
	}
	if _, err = file.Write(encoded.Bytes()); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(filepath.Join(s.dir, key))
		return "", "", fmt.Errorf("write mood photo: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(filepath.Join(s.dir, key))
		return "", "", fmt.Errorf("close mood photo: %w", closeErr)
	}
	return key, contentType, nil
}

func (s *Store) prepareDir() error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("create mood photo directory: %w", err)
	}
	if err := os.Chmod(s.dir, 0700); err != nil {
		return fmt.Errorf("secure mood photo directory: %w", err)
	}
	return nil
}

var (
	errTooLarge          = errors.New("image exceeds the upload limit")
	errUnsupportedType   = errors.New("unsupported image type")
	errInvalidImage      = errors.New("invalid or corrupt image")
	errInvalidDimensions = errors.New("invalid or excessive image dimensions")
)

// IsTooLarge reports whether Save rejected an image above the upload limit.
func IsTooLarge(err error) bool { return errors.Is(err, errTooLarge) }

// IsInvalid reports errors caused by unsupported, corrupt, or excessive image data.
func IsInvalid(err error) bool {
	return errors.Is(err, errUnsupportedType) || errors.Is(err, errInvalidImage) || errors.Is(err, errInvalidDimensions)
}

// Open resolves only store-generated keys and never accepts a path from a request.
func (s *Store) Open(key string) (*os.File, string, error) {
	if !keyPattern.MatchString(key) {
		return nil, "", os.ErrNotExist
	}
	contentType := "image/jpeg"
	if strings.HasSuffix(key, ".png") {
		contentType = "image/png"
	}
	file, err := os.Open(filepath.Join(s.dir, key))
	return file, contentType, err
}

// Delete removes a stored image. Missing files are already deleted and are treated as success.
func (s *Store) Delete(key string) error {
	if !keyPattern.MatchString(key) {
		return os.ErrNotExist
	}
	err := os.Remove(filepath.Join(s.dir, key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// StageDelete atomically moves a photo to a private temporary name. The caller must either
// Commit the removal after its database transaction succeeds or Restore the original key.
func (s *Store) StageDelete(key string) (*PendingDelete, error) {
	if !keyPattern.MatchString(key) {
		return nil, os.ErrNotExist
	}
	marker, err := randomKey(".pending")
	if err != nil {
		return nil, fmt.Errorf("generate pending-delete key: %w", err)
	}
	original := filepath.Join(s.dir, key)
	staged := filepath.Join(s.dir, ".delete-"+marker)
	if err := os.Rename(original, staged); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &PendingDelete{}, nil
		}
		return nil, fmt.Errorf("stage mood photo deletion: %w", err)
	}
	return &PendingDelete{original: original, staged: staged}, nil
}

// Commit permanently removes a staged photo. A caller may Restore it if this returns an error.
func (p *PendingDelete) Commit() error {
	if p.staged == "" {
		return nil
	}
	err := os.Remove(p.staged)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Restore moves a staged photo back to its original opaque key.
func (p *PendingDelete) Restore() error {
	if p.staged == "" {
		return nil
	}
	if err := os.Rename(p.staged, p.original); err != nil {
		return fmt.Errorf("restore staged mood photo: %w", err)
	}
	return nil
}

// BackupLocked writes a gzip-compressed tar snapshot of valid server-generated image files.
// Call it only inside WithExclusive so concurrent create/delete operations cannot make the
// archive disagree with its matching database snapshot.
func (s *Store) BackupLocked(dest string) error {
	if strings.TrimSpace(s.dir) == "" {
		return errors.New("mood photo directory is not configured")
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("list mood photos: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".mood-photo-backup-*")
	if err != nil {
		return fmt.Errorf("create temporary photo backup: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck
	if err := tmp.Chmod(0640); err != nil {
		tmp.Close() //nolint:errcheck
		return fmt.Errorf("secure temporary photo backup: %w", err)
	}
	zipper := gzip.NewWriter(tmp)
	tarWriter := tar.NewWriter(zipper)
	for _, entry := range entries {
		if !keyPattern.MatchString(entry.Name()) || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if err := addToArchive(tarWriter, filepath.Join(s.dir, entry.Name()), entry.Name()); err != nil {
			tarWriter.Close() //nolint:errcheck
			zipper.Close()    //nolint:errcheck
			tmp.Close()       //nolint:errcheck
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		zipper.Close() //nolint:errcheck
		tmp.Close()    //nolint:errcheck
		return fmt.Errorf("finish photo archive: %w", err)
	}
	if err := zipper.Close(); err != nil {
		tmp.Close() //nolint:errcheck
		return fmt.Errorf("compress photo archive: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close() //nolint:errcheck
		return fmt.Errorf("sync photo archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close photo archive: %w", err)
	}
	if err := verifyArchive(tmpName); err != nil {
		return err
	}
	if err := os.Link(tmpName, dest); err != nil {
		return fmt.Errorf("publish photo archive: %w", err)
	}
	return nil
}

func addToArchive(writer *tar.Writer, path, name string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open mood photo for backup: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat mood photo for backup: %w", err)
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("build mood photo archive header: %w", err)
	}
	header.Name = name
	header.Mode = 0600
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("write mood photo archive header: %w", err)
	}
	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("write mood photo to archive: %w", err)
	}
	return nil
}

func verifyArchive(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open photo backup for verification: %w", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open compressed photo backup: %w", err)
	}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			reader.Close() //nolint:errcheck
			return fmt.Errorf("verify photo archive: %w", err)
		}
		if !keyPattern.MatchString(header.Name) || header.Typeflag != tar.TypeReg {
			reader.Close() //nolint:errcheck
			return errors.New("photo archive contains an invalid entry")
		}
		if _, err := io.Copy(io.Discard, tarReader); err != nil {
			reader.Close() //nolint:errcheck
			return fmt.Errorf("read photo archive entry: %w", err)
		}
	}
	if err := reader.Close(); err != nil {
		return fmt.Errorf("finish photo archive verification: %w", err)
	}
	return nil
}

func randomKey(ext string) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ext, nil
}

func matchesFormat(contentType, format string) bool {
	switch contentType {
	case "image/jpeg":
		return format == "jpeg"
	case "image/png":
		return format == "png"
	case "image/webp":
		return format == "webp"
	default:
		return false
	}
}

func resize(src image.Image, maxDimension int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxDimension && h <= maxDimension {
		return src
	}
	scale := float64(maxDimension) / float64(max(w, h))
	dstW := max(1, int(float64(w)*scale))
	dstH := max(1, int(float64(h)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)
	return dst
}
