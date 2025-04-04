package images

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	msql "github.com/discuitnet/discuit/internal/sql"
	"github.com/discuitnet/discuit/internal/uid"
	"golang.org/x/exp/slices"

	// Register jpeg and png decoding for images pkg.
	_ "image/jpeg"
	_ "image/png"

	// Register webp decoding for images pkg.
	_ "golang.org/x/image/webp"
)

var (
	// Global registered stores.
	stores []store

	// HMACKey is the key used to set the HMAC portion of an image's URL.
	HMACKey []byte

	// FullImageURL takes in a partial url (a pathname with a set of query
	// parameters) and it should return a more complete url. This variable may
	// be nil.
	FullImageURL = func(s string) string {
		return "/images/" + s
	}
)

func init() {
	if err := registerStore(newDiskStore()); err != nil {
		panic(err)
	}
}

var (
	ErrImageNotFound          = errors.New("image not found")
	ErrStoreNotRegistered     = errors.New("store not registered")
	ErrBadURL                 = errors.New("bad image request url")
	ErrImageFormatUnsupported = errors.New("image format not supported")
	ErrImageFitUnsupported    = errors.New("invalid image fit")
)

func registerStore(s store) error {
	for _, store := range stores {
		if store.name() == s.name() {
			return fmt.Errorf("a store with the name %v is already registered", s.name())
		}
	}
	stores = append(stores, s)
	return nil
}

func matchStore(name string) store {
	for _, s := range stores {
		if s.name() == name {
			return s
		}
	}
	return nil
}

// A store saves images to a permanent location. Each store is identified by a
// name that must be unique to the running process.
type store interface {
	get(*ImageRecord) ([]byte, error)
	save(r *ImageRecord, image []byte) error
	delete(*ImageRecord) error
	name() string // The identifier of the store.
}

// ImageFormat represents the type of image.
type ImageFormat string

// List of image formats.
const (
	ImageFormatJPEG = ImageFormat("jpeg")
	ImageFormatWEBP = ImageFormat("webp")
	ImageFormatPNG  = ImageFormat("png")
)

// Valid reports whether f is supported by the image package.
func (f ImageFormat) Valid() bool {
	return slices.Contains([]ImageFormat{
		ImageFormatJPEG,
		ImageFormatWEBP,
		ImageFormatPNG,
	}, f)
}

func (f ImageFormat) Extension() string {
	return "." + string(f)
}

// RGB represents color values of range (0, 255). It implements sql.Scanner and
// driver.Valuer interfaces. Use a 12-byte binary database column type to store
// values of this type in SQL databases.
type RGB struct {
	Red   uint32 `json:"r"`
	Green uint32 `json:"g"`
	Blue  uint32 `json:"b"`
}

// String implements fmt.Stringer.
func (c RGB) String() string {
	return "rgb(" + strconv.Itoa(int(c.Red)) + "," + strconv.Itoa(int(c.Green)) + "," + strconv.Itoa(int(c.Blue)) + ")"
}

// MarshalText implements encoding.TextMarshaler.
func (c RGB) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

var errBadRGB = errors.New("unmarshalling images.RGB error")

// UnmarshalText implements encoding.TextUnmarshaler interface.
func (c *RGB) UnmarshalText(b []byte) error {
	s := string(b)
	if len(s) == 0 {
		return nil
	}
	if len(b) < 8 {
		return errBadRGB
	}
	if !(strings.HasPrefix(s, `rgb(`) && strings.HasSuffix(s, `)`)) {
		return errBadRGB
	}

	s = s[4 : len(s)-1]
	v := strings.Split(s, ",")
	for i := range v {
		v[i] = strings.TrimSpace(v[i])
	}

	if len(v) != 3 {
		return errBadRGB
	}
	red, err := strconv.Atoi(v[0])
	if err != nil {
		return errBadRGB
	}
	green, err := strconv.Atoi(v[1])
	if err != nil {
		return errBadRGB
	}
	blue, err := strconv.Atoi(v[2])
	if err != nil {
		return errBadRGB
	}
	if red < 0 || green < 0 || blue < 0 {
		return errBadRGB
	}

	c.Red = uint32(red)
	c.Green = uint32(green)
	c.Blue = uint32(blue)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *RGB) UnmarshalJSON(b []byte) error {
	if len(b) < 3 {
		return errBadRGB
	}
	if string(b) == "null" {
		return nil
	}
	if !(b[0] == '"' && b[len(b)-1] == '"') {
		return errBadRGB
	}
	return c.UnmarshalText(b[1 : len(b)-1])
}

// Scan implements sql.Scanner interface.
func (c *RGB) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok {
		return errors.New("RGB scan source value is not a slice of bytes")
	}
	if len(b) < 12 {
		return errors.New("RGB scan source value is too short")
	}
	c.Red = binary.LittleEndian.Uint32(b)
	c.Green = binary.LittleEndian.Uint32(b[4:])
	c.Blue = binary.LittleEndian.Uint32(b[8:])
	return nil
}

// Value implements driver.Valuer interface.
func (c RGB) Value() (driver.Value, error) {
	b := make([]byte, 12)
	binary.LittleEndian.PutUint32(b, c.Red)
	binary.LittleEndian.PutUint32(b[4:], c.Green)
	binary.LittleEndian.PutUint32(b[8:], c.Blue)
	return b, nil
}

// ImageSize represents the size of an image.
type ImageSize struct {
	Width, Height int
}

// Zero returns true if either the width or the height is 0.
func (s ImageSize) Zero() bool {
	return s.Width == 0 || s.Height == 0
}

// String returns, for example, "400" if width and height are both 400px, and
// "400x600" if width is 400px and height is 600px.
func (s ImageSize) String() string {
	if s.Width == s.Height {
		return strconv.Itoa(s.Width)
	}
	return strconv.Itoa(s.Width) + "x" + strconv.Itoa(s.Height)
}

// MarshalText implements encoding.TextMarshaler interface. Output is the same
// as String method.
func (s ImageSize) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

var errInvalidImageSize = errors.New("invalid image size")

// UnmarshalText implements encoding.TextUnmarshaler interface.
func (s *ImageSize) UnmarshalText(text []byte) error {
	str := string(text)
	i := strings.Index(str, "x")
	if i == -1 {
		width, err := strconv.Atoi(str)
		if err != nil {
			return errInvalidImageSize
		}
		s.Width = width
		s.Height = width
		return nil
	}

	if len(str) < i+2 {
		return errInvalidImageSize
	}

	width, err := strconv.Atoi(str[:i])
	if err != nil {
		return errInvalidImageSize
	}
	height, err := strconv.Atoi(str[i+1:])
	if err != nil {
		return errInvalidImageSize
	}

	s.Width, s.Height = width, height
	return nil
}

// ImageFit denotes how an image is to be fitted into a rectangle.
type ImageFit string

// There are the valid ImageFit values.
const (
	// ImageFitCover covers the given container with the image. The resulting
	// image may be shrunken, enlarged, and/or cropped.
	ImageFitCover = ImageFit("cover")

	// ImageFitContain fits the image in the container without either enlarging
	// the image or cropping it.
	ImageFitContain = ImageFit("contain")

	ImageFitDefault = ImageFitContain
)

// Supported returns false if f is not supported by this package.
func (f ImageFit) Supported() bool {
	supported := []ImageFit{
		ImageFitCover,
		ImageFitContain,
	}
	return slices.Contains(supported, f)
}

// ImageContainSize returns the width and height of an image as it would fit
// into a box, while keeping the aspect ratio of the image as it was.
func ImageContainSize(imageWidth, imageHeight, boxWidth, boxHeight int) (int, int) {
	x, y, scale := float64(imageWidth), float64(imageHeight), 1.0
	if imageWidth > boxWidth {
		scale = float64(boxWidth) / float64(imageWidth)
		x = scale * float64(imageWidth)
		y = scale * float64(imageHeight)
	}
	if y > float64(boxHeight) {
		scale = float64(boxHeight) / y
		x = scale * x
		y = scale * y
	}
	return int(x), int(y)
}

// AverageColor returns the average RGB color of img by averaging the colors of
// at most 10^4 pixels.
func AverageColor(img image.Image) RGB {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	xsteps, ysteps := int(math.Floor(float64(width)/100.0)), int(math.Floor(float64(height)/100.0))
	if xsteps <= 0 {
		xsteps = 1
	}
	if ysteps <= 0 {
		ysteps = 1
	}

	var r, g, b float64
	for i := 0; i < width; i += xsteps {
		for j := 0; j < height; j += ysteps {
			c := img.At(i, j)
			r2, g2, b2, _ := c.RGBA()
			r = (r + float64(r2)) / 2
			g = (g + float64(g2)) / 2
			b = (b + float64(b2)) / 2
		}
	}

	var c RGB
	x := r + g + b
	if x > 0 {
		c.Red = uint32(r / x * 255.0)
		c.Green = uint32(g / x * 255.0)
		c.Blue = uint32(b / x * 255.0)
	}
	return c
}

// request is an incoming request for an image.
type request struct {
	id     uid.ID    // ID of the image.
	size   ImageSize // If zero, return the image without altering size.
	fit    ImageFit
	format ImageFormat // Should never be empty.
	hash   []byte      // Incoming request hash value from the URL parameters.
}

func fromURL(u *url.URL) (_ *request, err error) {
	r := &request{}
	parts := strings.Split(u.Path, "/")
	if len(parts) == 0 {
		return nil, ErrBadURL
	}

	idStr, extension, found := strings.Cut(parts[len(parts)-1], ".")
	if !found {
		return nil, ErrBadURL
	}

	r.id, err = uid.FromString(idStr)
	if err != nil {
		return nil, ErrBadURL
	}

	if r.format = ImageFormat(extension); !r.format.Valid() {
		return nil, ErrImageFormatUnsupported
	}

	query := u.Query()
	if size := query.Get("size"); size != "" {
		if err = r.size.UnmarshalText([]byte(size)); err != nil {
			return nil, ErrBadURL
		}
	}

	r.fit = ImageFit(query.Get("fit"))
	if r.fit != "" && !r.fit.Supported() {
		return nil, ErrImageFitUnsupported
	}
	if !r.size.Zero() && r.fit == "" {
		return nil, errors.New("zero size requires a non-empty image fit")
	}

	r.hash, err = base64.RawURLEncoding.DecodeString(query.Get("sig"))
	if err != nil {
		return nil, ErrBadURL
	}
	return r, nil
}

// valid reports whether r has a valid signature.
func (r *request) valid() bool {
	return hmac.Equal(r.computeHash(), r.hash)
}

// computeHash returns the hash signature of r.
func (r *request) computeHash() []byte {
	hm := hmac.New(sha256.New, HMACKey)
	hm.Write(r.hashData())
	return hm.Sum(nil)
}

func (r *request) hashData() []byte {
	id := r.id.String()
	size := r.size.String() // may be "0"
	fit := ""
	if !r.size.Zero() {
		fit = string(r.fit)
	}
	ext := r.format.Extension()
	return []byte(id + size + fit + ext)
}

// filename returns a string of the format "{FileHash}_300x400_contain.jpeg"
// used for storing images for caching purposes.
func (r *request) filename() string {
	_, s := idToFolder(r.id)
	if !r.size.Zero() {
		s += "_" + r.size.String()
		// ImageFit only makes sense if a size (other than that of the original
		// image) is specified.
		if r.fit == "" {
			s += "_" + string(ImageFitDefault)
		} else {
			s += "_" + string(r.fit)
		}
	}
	s += r.format.Extension()
	return s
}

// url returns a string of the format "{ID}.jpeg?size=300&fit=contain&sig={MAC}".
// If key is nil, the signature query parameter is omitted from the URL.
func (r *request) url() string {
	v := url.Values{}
	if !r.size.Zero() {
		v.Set("size", r.size.String())
		v.Set("fit", string(r.fit))
	}

	if HMACKey != nil {
		v.Set("sig", base64.RawURLEncoding.EncodeToString(r.computeHash()))
	}

	var search string
	if len(v) > 0 {
		search = "?" + v.Encode()
	}
	return r.id.String() + r.format.Extension() + search
}

func cacheFilepath(r *request) string {
	folder, _ := idToFolder(r.id)
	// Cache files are stored in the same directory as diskStore and alongside
	// the original image.
	return path.Join(filesRootFolder, folder, r.filename())
}

func getCachedImage(r *request) (image []byte, err error) {
	return os.ReadFile(cacheFilepath(r))
}

func putToCache(image []byte, r *request) error {
	return os.WriteFile(cacheFilepath(r), image, 0755)
}

func removeFromCache(image uid.ID) error {
	folder, filename := idToFolder(image)
	return filepath.Walk(path.Join(filesRootFolder, folder), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("skipping unwalkable directory: %v", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, filename) && strings.Contains(base, "_") {
			log.Println("deleting cached image: ", path)
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete cached image %s: %w", image, err)
			}
		}
		return nil
	})
}

// ClearCache removes all cached image files.
func ClearCache() error {
	return filepath.Walk(path.Join(filesRootFolder), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("skipping unwalkable directory: %v", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.Contains(filepath.Base(path), "_") {
			log.Println("deleting cached image: ", path)
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete cached image: %w", err)
			}
		}
		return nil
	})
}

// getImage returns an image (after optionally transforming it) as per the
// options in r. Make sure to check whether the request has a valid signature by
// calling r.Valid before calling this function.
func getImage(ctx context.Context, db *sql.DB, r *request, cacheEnabled bool) ([]byte, error) {
	if cacheEnabled {
		if image, err := getCachedImage(r); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("getCachedImage error: %v\n", err)
			}
			// Failed retreiving from cache, proceed.
		} else {
			return image, nil
		}
	}

	record, err := GetImageRecord(ctx, db, r.id)
	if err != nil {
		return nil, err
	}

	store := record.store()
	if store == nil {
		return nil, fmt.Errorf("image store %v is not found", record.StoreName)
	}

	image, err := store.get(record)
	if err != nil {
		return nil, err
	}

	// For now, we'll just return the original image without any processing
	return image, nil
}

// ImageOptions hold optional arguments to SaveImage.
type ImageOptions struct {
	Width, Height int
	Format        ImageFormat
	Fit           ImageFit
}

// SaveImage saves the provided image in the image store with the name storeName
// and creates a row in the images table. The argument opts can be nil, in which
// case default values are used.
func SaveImage(ctx context.Context, db *sql.DB, storeName string, file []byte, opts *ImageOptions) (*ImageRecord, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil
	}

	id, err := SaveImageTx(ctx, tx, storeName, file, opts)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			log.Println("images.SaveImage rollback error: ", err)
		}
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return GetImageRecord(ctx, db, id)
}

// If SkipProcessing is set to true, images are saved as is, without compressing
// nor changing their size or format. Warning: This may lead to inadvertently
// storing (and leaking) image metadata.
var SkipProcessing = false

func SaveImageTx(ctx context.Context, tx *sql.Tx, storeName string, file []byte, opts *ImageOptions) (uid.ID, error) {
	if opts == nil {
		opts = &ImageOptions{
			Format: ImageFormatJPEG,
		}
	}

	// For now, we'll just save the original image without any processing
	img := file

	// Get image dimensions
	decodedImg, _, err := image.Decode(bytes.NewBuffer(img))
	if err != nil {
		return uid.ID{}, err
	}
	bounds := decodedImg.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	store := matchStore(storeName)
	if store == nil {
		return uid.ID{}, ErrStoreNotRegistered
	}

	averageColor := AverageColor(decodedImg)

	id := uid.New()
	query, args := msql.BuildInsertQuery("images", []msql.ColumnValue{
		{Name: "id", Value: id},
		{Name: "store_name", Value: storeName},
		{Name: "format", Value: opts.Format},
		{Name: "width", Value: width},
		{Name: "height", Value: height},
		{Name: "size", Value: len(img)},
		{Name: "upload_size", Value: len(file)},
		{Name: "average_color", Value: averageColor},
	})

	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return uid.ID{}, err
	}

	if err = store.save(&ImageRecord{
		ID:        id,
		StoreName: storeName,
		Format:    opts.Format,
	}, img); err != nil {
		return uid.ID{}, fmt.Errorf("error saving image: %v", err)
	}

	return id, nil
}

func DeleteImagesTx(ctx context.Context, tx *sql.Tx, db *sql.DB, images ...uid.ID) error {
	records, err := GetImageRecords(ctx, db, images...)
	if err != nil {
		return err
	}

	for _, record := range records {
		if err := record.store().delete(record); err != nil {
			return err
		}
	}

	// Attempt to remove images from cache. Continue even on failure.
	for _, image := range images {
		if err := removeFromCache(image); err != nil {
			log.Printf("error removing images from cache on image id %v", err)
		}
	}

	args := make([]any, len(images))
	for i := range images {
		args[i] = images[i]
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM images WHERE id IN %s", msql.InClauseQuestionMarks(len(images))), args...)
	return err
}

// ImageProcessor processes images.
type ImageProcessor struct {
	db        *sql.DB
	directory string
}

// NewImageProcessor creates a new ImageProcessor.
func NewImageProcessor(db *sql.DB, directory string) *ImageProcessor {
	return &ImageProcessor{
		db:        db,
		directory: directory,
	}
}

// SaveImage saves an image file with the given options.
func (p *ImageProcessor) SaveImage(file multipart.File, opts ImageOptions) (*Image, error) {
	// Read file into memory
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return nil, err
	}
	image := buf.Bytes()

	// Generate a unique ID for the image
	hash := sha256.Sum256(image)
	id := hex.EncodeToString(hash[:])[:12]
	uid, err := uid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid image ID: %v", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(p.directory, 0755); err != nil {
		return nil, err
	}

	// Save the original image
	filename := fmt.Sprintf("%s.%s", id, opts.Format)
	filepath := path.Join(p.directory, filename)
	if err := os.WriteFile(filepath, image, 0644); err != nil {
		return nil, err
	}

	// Create image record
	img := NewImage()
	*img.ID = uid
	*img.Format = opts.Format
	*img.Width = opts.Width
	*img.Height = opts.Height
	img.PostScan()

	return img, nil
}

// ServeImage serves an image file.
func (p *ImageProcessor) ServeImage(w http.ResponseWriter, r *http.Request, id string) error {
	// Find image files matching the ID
	pattern := path.Join(p.directory, id+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return errors.New("image not found")
	}

	// Serve the first matching file
	filename := matches[0]
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := ""
	switch ext {
	case ".jpeg", ".jpg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	default:
		return errors.New("unsupported image format")
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filename)
	return nil
}

// DeleteImage deletes an image file.
func (p *ImageProcessor) DeleteImage(id string) error {
	pattern := path.Join(p.directory, id+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, filename := range matches {
		if err := os.Remove(filename); err != nil {
			return err
		}
	}
	return nil
}
