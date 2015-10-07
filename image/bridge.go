package image

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/henvic/picel/logger"
	"github.com/rakyll/magicmime"
)

const (
	WEBP_QUALITY    = "92"
	IMAGICK_QUALITY = "92"
)

var (
	ErrOutputFormatNotSupported = errors.New("The requested output format is not supported")
	ErrMimeTypeExtension        = errors.New("Internal mime type extension error")
	ErrMimeTypeNotSupported     = errors.New("The loaded file mime type is not supported")
	Verbose                     = false
)

var OutputFormats = map[string]string{
	"jpg":  "Imagick",
	"jpeg": "Imagick",
	"gif":  "Imagick",
	"png":  "Imagick",
	"pdf":  "Imagick",
	"webp": "Webp",
}

var ValidInputMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

func init() {
	if err := magicmime.Open(
		magicmime.MAGIC_MIME_TYPE |
			magicmime.MAGIC_SYMLINK |
			magicmime.MAGIC_ERROR); err != nil {
		log.Fatal(err)
	}
}

func Process(t Transform, input string, output string) (err error) {
	tool, valid := OutputFormats[strings.ToLower(t.Output)]

	if !valid {
		return ErrOutputFormatNotSupported
	}

	mimeType, mimeErr := magicmime.TypeByFile(input)

	if mimeErr != nil {
		return ErrMimeTypeExtension
	}

	if !ValidInputMimeTypes[mimeType] {
		return ErrMimeTypeNotSupported
	}

	if tool == "Imagick" {
		return processImagick(t, input, output)
	}

	return processWebp(t, input, output)
}

func callProgram(name string, params []string) error {
	cmd := exec.Command(name, params...)
	var bOut bytes.Buffer
	var bErr bytes.Buffer
	cmd.Stdout = &bOut
	cmd.Stderr = &bErr

	if Verbose {
		logger.Stdout.Println(fmt.Sprintf("%v %v", name, strings.Join(params, " ")))
	}

	cmdErr := cmd.Run()

	if Verbose {
		logger.Stdout.Println(string(bOut.Bytes()))
		logger.Stderr.Println(string(bErr.Bytes()))
	}

	return cmdErr
}

func processWebp(t Transform, input string, output string) (err error) {
	if t.Extension != "gif" {
		return processCwebp(t, input, output)
	}

	if t.Crop.Width != 0 || t.Crop.Height != 0 || t.Width != 0 || t.Height != 0 {
		t.Output = "gif"
		err = processImagick(t, input, output)
		t.Output = "webp"
		input = output

		if err != nil {
			return err
		}
	}

	return processGif2Webp(input, output)
}

func processGif2Webp(input string, output string) (err error) {
	var params []string

	params = append(params, "-q")
	params = append(params, WEBP_QUALITY)

	if Verbose {
		params = append(params, "-v")
	}

	params = append(params, input)
	params = append(params, "-o")
	params = append(params, output)

	return callProgram("gif2webp", params)
}

func processCwebp(t Transform, input string, output string) (err error) {
	var params []string

	params = append(params, "-q")
	params = append(params, WEBP_QUALITY)

	if t.Crop.Width != 0 && t.Crop.Height != 0 {
		params = append(params, "-crop")
		params = append(params, fmt.Sprintf("%d", t.Crop.X))
		params = append(params, fmt.Sprintf("%d", t.Crop.Y))
		params = append(params, fmt.Sprintf("%d", t.Crop.Width))
		params = append(params, fmt.Sprintf("%d", t.Crop.Height))
	}

	if t.Width != 0 || t.Height != 0 {
		params = append(params, "-resize")
		params = append(params, fmt.Sprintf("%d", t.Width))
		params = append(params, fmt.Sprintf("%d", t.Height))
	}

	if Verbose {
		params = append(params, "-v")
	}

	params = append(params, input)
	params = append(params, "-o")
	params = append(params, output)

	return callProgram("cwebp", params)
}

func processImagick(t Transform, input string, output string) (err error) {
	var params []string

	if Verbose {
		params = append(params, "-verbose")
	}

	params = append(params, "-quality")

	params = append(params, IMAGICK_QUALITY)

	params = append(params, input)

	params = append(params, "-strip")

	c := t.Crop

	if c.Width != 0 && c.Height != 0 {
		crop := fmt.Sprintf("%dx%d+%d+%d", c.Width, c.Height, c.X, c.Y)
		params = append(params, "-crop")
		params = append(params, crop)
		params = append(params, "+repage")
	}

	if t.Width != 0 || t.Height != 0 {
		resize := ""

		if t.Width > 0 {
			resize += fmt.Sprintf("%d", t.Width)
		}

		resize += "x"

		if t.Height > 0 {
			resize += fmt.Sprintf("%d", t.Height)
		}

		params = append(params, "-resize")
		params = append(params, resize)
	}

	output = strings.ToLower(t.Output) + ":" + output

	params = append(params, output)

	return callProgram("convert", params)
}
