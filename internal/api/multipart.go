package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
)

// WorkoutMultipart builds a streaming multipart/form-data body for POST /v1/workout.
// smlPath points to a .sml file on disk (required). extensionsPath is optional;
// pass "" to omit. The returned reader pipes parts through io.Pipe — closing
// the reader cancels the goroutine. The returned headers contain the
// Content-Type with the auto-generated boundary; merge with any auth headers
// the caller needs (e.g. STTAuthorization is added by Client.Do).
//
// Caller MUST eventually consume or Close the reader to release the goroutine.
func WorkoutMultipart(smlPath, extensionsPath string) (io.ReadCloser, map[string]string, error) {
	if smlPath == "" {
		return nil, nil, fmt.Errorf("multipart: smlPath is required")
	}
	// Validate the SML file exists before we start the goroutine — cheaper failure.
	if _, err := os.Stat(smlPath); err != nil {
		return nil, nil, fmt.Errorf("multipart: stat sml: %w", err)
	}
	if extensionsPath != "" {
		if _, err := os.Stat(extensionsPath); err != nil {
			return nil, nil, fmt.Errorf("multipart: stat extensions: %w", err)
		}
	}

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	headers := map[string]string{"Content-Type": mw.FormDataContentType()}

	go func() {
		// Single error sink: close the writer with whatever went wrong.
		err := writeWorkoutParts(mw, smlPath, extensionsPath)
		_ = mw.Close()
		_ = pw.CloseWithError(err)
	}()

	return pr, headers, nil
}

func writeWorkoutParts(mw *multipart.Writer, smlPath, extensionsPath string) error {
	if err := copyFilePart(mw, "filePart", smlPath, "application/octet-stream"); err != nil {
		return err
	}
	if extensionsPath != "" {
		if err := copyFilePart(mw, "workoutExtensionsPart", extensionsPath, "application/json"); err != nil {
			return err
		}
	}
	return nil
}

func copyFilePart(mw *multipart.Writer, field, path, contentType string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, filepath.Base(path)))
	h.Set("Content-Type", contentType)
	w, err := mw.CreatePart(h)
	if err != nil {
		return fmt.Errorf("multipart: create %s part: %w", field, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("multipart: open %s: %w", field, err)
	}
	defer f.Close()
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("multipart: copy %s: %w", field, err)
	}
	return nil
}
