package moodphoto

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveResizesPNGAndCreatesPrivateOpaqueFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := New(dir)

	source := image.NewNRGBA(image.Rect(0, 0, 1600, 800))
	source.Set(10, 10, color.NRGBA{R: 20, G: 80, B: 140, A: 255})
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatal(err)
	}

	key, contentType, err := store.Save(&input)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if contentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", contentType)
	}
	if strings.ContainsAny(key, `/\\`) || !keyPattern.MatchString(key) {
		t.Fatalf("key is not an opaque filename: %q", key)
	}
	info, err := os.Stat(filepath.Join(dir, key))
	if err != nil {
		t.Fatalf("stat saved image: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %o, want 600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("directory permissions = %o, want 700", dirInfo.Mode().Perm())
	}

	file, gotType, err := store.Open(key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer file.Close()
	if gotType != contentType {
		t.Fatalf("opened content type = %q, want %q", gotType, contentType)
	}
	decoded, format, err := image.Decode(file)
	if err != nil {
		t.Fatalf("decode normalized image: %v", err)
	}
	if format != "png" || decoded.Bounds().Dx() != MaxImageDimension || decoded.Bounds().Dy() != 640 {
		t.Fatalf("normalized image = %s %v, want png 1280x640", format, decoded.Bounds())
	}
}

func TestSaveReencodesJPEGAndStripsEXIF(t *testing.T) {
	t.Parallel()
	source := image.NewRGBA(image.Rect(0, 0, 40, 30))
	var jpegBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, source, nil); err != nil {
		t.Fatal(err)
	}
	withEXIF := addJPEGAPP1(jpegBytes.Bytes(), exifGPSMetadata())

	store := New(t.TempDir())
	key, contentType, err := store.Save(bytes.NewReader(withEXIF))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if contentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", contentType)
	}
	got, err := os.ReadFile(filepath.Join(store.dir, key))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte("Exif\x00\x00")) || bytes.Contains(got, exifGPSMetadata()) {
		t.Fatal("normalized photo retained EXIF/GPS metadata")
	}
}

func TestSaveAcceptsWebPAndNormalizesToJPEG(t *testing.T) {
	t.Parallel()
	const onePixelWebP = "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA"
	webpBytes, err := base64.StdEncoding.DecodeString(onePixelWebP)
	if err != nil {
		t.Fatal(err)
	}
	store := New(t.TempDir())
	key, contentType, err := store.Save(bytes.NewReader(webpBytes))
	if err != nil {
		t.Fatalf("Save WebP: %v", err)
	}
	if contentType != "image/jpeg" || !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("normalized WebP = key %q, type %q; want JPEG", key, contentType)
	}
	file, err := os.Open(filepath.Join(store.dir, key))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, format, err := image.Decode(file); err != nil || format != "jpeg" {
		t.Fatalf("normalized WebP decode format = %q, err %v; want jpeg", format, err)
	}
}

func TestSaveRejectsOversizedInput(t *testing.T) {
	t.Parallel()
	store := New(t.TempDir())
	_, _, err := store.Save(bytes.NewReader(make([]byte, MaxUploadBytes+1)))
	if !IsTooLarge(err) {
		t.Fatalf("Save error = %v, want size-limit error", err)
	}
}

func TestSaveRejectsUnsupportedAndCorruptImages(t *testing.T) {
	t.Parallel()
	store := New(t.TempDir())
	cases := []struct {
		name string
		data []byte
	}{
		{name: "non-image", data: []byte("not a photo, even if its filename says .jpg")},
		{name: "truncated jpeg", data: []byte{0xff, 0xd8, 0xff, 0xd9}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := store.Save(bytes.NewReader(tc.data)); err == nil {
				t.Fatal("Save accepted invalid image data")
			}
		})
	}
}

func TestOpenAndDeleteRejectPathTraversal(t *testing.T) {
	t.Parallel()
	store := New(t.TempDir())
	if _, _, err := store.Open("../../secret.jpg"); !os.IsNotExist(err) {
		t.Fatalf("Open traversal error = %v, want not-exist", err)
	}
	if err := store.Delete("../../secret.jpg"); !os.IsNotExist(err) {
		t.Fatalf("Delete traversal error = %v, want not-exist", err)
	}
}

func TestDeleteRemovesStoredPhotoAndIsIdempotent(t *testing.T) {
	t.Parallel()
	store := New(t.TempDir())
	source := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatal(err)
	}
	key, _, err := store.Save(&input)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.dir, key)); !os.IsNotExist(err) {
		t.Fatalf("deleted image still exists: %v", err)
	}
	if err := store.Delete(key); err != nil {
		t.Fatalf("second Delete should be idempotent: %v", err)
	}
}

func TestBackupLockedArchivesOnlyOpaqueImagesAndRefusesOverwrite(t *testing.T) {
	t.Parallel()
	store := New(t.TempDir())
	backupDir := t.TempDir()
	archivePath := filepath.Join(backupDir, "moods.tar.gz")
	var savedKey string
	if err := store.WithExclusive(func() error {
		var err error
		savedKey, _, err = store.Save(bytes.NewReader(testPNG(t)))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(store.dir, "not-a-photo"), []byte("skip"), 0600); err != nil {
			return err
		}
		return store.BackupLocked(archivePath)
	}); err != nil {
		t.Fatalf("backup photos: %v", err)
	}
	if info, err := os.Stat(archivePath); err != nil {
		t.Fatalf("stat archive: %v", err)
	} else if info.Mode().Perm() != 0640 {
		t.Errorf("archive permissions = %o, want 640", info.Mode().Perm())
	}

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	archive := tar.NewReader(reader)
	header, err := archive.Next()
	if err != nil {
		t.Fatalf("first archive entry: %v", err)
	}
	if header.Name != savedKey || header.Typeflag != tar.TypeReg {
		t.Fatalf("archive entry = %+v, want only generated image %s", header, savedKey)
	}
	if _, err := io.Copy(io.Discard, archive); err != nil {
		t.Fatalf("read image from archive: %v", err)
	}
	if _, err := archive.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("archive contains another entry: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}

	if err := store.WithExclusive(func() error { return store.BackupLocked(archivePath) }); err == nil {
		t.Fatal("backup must not overwrite an existing archive")
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func addJPEGAPP1(jpegBytes, metadata []byte) []byte {
	segmentLength := len(metadata) + 2
	segment := []byte{0xff, 0xe1, byte(segmentLength >> 8), byte(segmentLength)}
	segment = append(segment, metadata...)
	withMetadata := make([]byte, 0, len(jpegBytes)+len(segment))
	withMetadata = append(withMetadata, jpegBytes[:2]...)
	withMetadata = append(withMetadata, segment...)
	withMetadata = append(withMetadata, jpegBytes[2:]...)
	return withMetadata
}

func exifGPSMetadata() []byte {
	// Little-endian TIFF with an IFD0 GPSInfo pointer and a GPSLatitude rational triplet.
	tiff := []byte{
		'I', 'I', 0x2a, 0, 8, 0, 0, 0,
		1, 0, 0x25, 0x88, 4, 0, 1, 0, 0, 0, 26, 0, 0, 0, 0, 0, 0, 0,
		1, 0, 2, 0, 5, 0, 3, 0, 0, 0, 44, 0, 0, 0, 0, 0, 0, 0,
		51, 0, 0, 0, 1, 0, 0, 0,
		30, 0, 0, 0, 1, 0, 0, 0,
		7, 0, 0, 0, 1, 0, 0, 0,
	}
	return append([]byte("Exif\x00\x00"), tiff...)
}
