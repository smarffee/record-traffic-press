package settings

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// HTTPModifierConfig holds configuration options for built-in traffic modifier
type HTTPModifierConfig struct {
	URLNegativeRegexp      HTTPURLRegexp              `json:"http-disallow-url"`
	URLRegexp              HTTPURLRegexp              `json:"http-allow-url"`
	URLRewrite             URLRewriteMap              `json:"http-rewrite-url"`
	HeaderRewrite          HeaderRewriteMap           `json:"http-rewrite-header"`
	HeaderFilters          HTTPHeaderFilters          `json:"http-allow-header"`
	HeaderNegativeFilters  HTTPHeaderFilters          `json:"http-disallow-header"`
	HeaderBasicAuthFilters HTTPHeaderBasicAuthFilters `json:"http-basic-auth-filter"`
	HeaderHashFilters      HTTPHashFilters            `json:"http-header-limiter"`
	ParamHashFilters       HTTPHashFilters            `json:"http-param-limiter"`
	Params                 HTTPParams                 `json:"http-set-param"`
	Headers                HTTPHeaders                `json:"http-set-header"`
	Methods                HTTPMethods                `json:"http-allow-method"`
}

// HeaderFilter Handling of --http-allow-header, --http-disallow-header options
type HeaderFilter struct {
	Name   []byte
	Regexp *regexp.Regexp
}

// HTTPHeaderFilters holds list of headers and their regexps
type HTTPHeaderFilters []HeaderFilter

func (h *HTTPHeaderFilters) String() string {
	return fmt.Sprint(*h)
}

// Set method to implement flags.Value
func (h *HTTPHeaderFilters) Set(value string) error {
	valArr := strings.SplitN(value, ":", 2)
	if len(valArr) < 2 {
		return errors.New("need both header and value, colon-delimited (ex. user_id:^169$)")
	}
	val := strings.TrimSpace(valArr[1])
	r, err := regexp.Compile(val)
	if err != nil {
		return err
	}

	*h = append(*h, HeaderFilter{Name: []byte(valArr[0]), Regexp: r})

	return nil
}

// BasicAuthFilter Handling of --http-basic-auth-filter option
type BasicAuthFilter struct {
	Regexp *regexp.Regexp
}

// HTTPHeaderBasicAuthFilters holds list of regxp to match basic Auth header values
type HTTPHeaderBasicAuthFilters []BasicAuthFilter

func (h *HTTPHeaderBasicAuthFilters) String() string {
	return fmt.Sprint(*h)
}

// Set method to implement flags.Value
func (h *HTTPHeaderBasicAuthFilters) Set(value string) error {
	r, err := regexp.Compile(value)
	if err != nil {
		return err
	}

	*h = append(*h, BasicAuthFilter{Regexp: r})

	return nil
}

// Handling of --http-allow-header-hash and --http-allow-param-hash options
type HashFilter struct {
	Name    []byte
	Percent uint32
}

// HTTPHashFilters represents a slice of header hash filters
type HTTPHashFilters []HashFilter

func (h *HTTPHashFilters) String() string {
	return fmt.Sprint(*h)
}

// Set method to implement flags.Value
func (h *HTTPHashFilters) Set(value string) error {
	valArr := strings.SplitN(value, ":", 2)
	if len(valArr) < 2 {
		return errors.New("need both header and value, colon-delimited (ex. user_id:50%)")
	}

	f := HashFilter{Name: []byte(valArr[0])}

	val := strings.TrimSpace(valArr[1])

	if strings.Contains(val, "%") {
		p, _ := strconv.ParseInt(val[:len(val)-1], 0, 0)
		f.Percent = uint32(p)
	} else if strings.Contains(val, "/") {
		// DEPRECATED format
		var num, den uint64

		fracArr := strings.Split(val, "/")
		num, _ = strconv.ParseUint(fracArr[0], 10, 64)
		den, _ = strconv.ParseUint(fracArr[1], 10, 64)

		f.Percent = uint32((float64(num) / float64(den)) * 100)
	} else {
		return errors.New("Value should be percent and contain '%'")
	}

	*h = append(*h, f)

	return nil
}

// Handling of --http-set-header option
type httpHeader struct {
	Name  string
	Value string
}

// HTTPHeaders is a slice of headers that must appended
type HTTPHeaders []httpHeader

func (h *HTTPHeaders) String() string {
	return fmt.Sprint(*h)
}

// Set method to implement flags.Value
func (h *HTTPHeaders) Set(value string) error {
	v := strings.SplitN(value, ":", 2)
	if len(v) != 2 {
		return errors.New("Expected `Key: Value`")
	}

	header := httpHeader{
		strings.TrimSpace(v[0]),
		strings.TrimSpace(v[1]),
	}

	*h = append(*h, header)
	return nil
}

// Handling of --http-set-param option
type httpParam struct {
	Name  []byte
	Value []byte
}

// HTTPParams filters for --http-set-param
type HTTPParams []httpParam

func (h *HTTPParams) String() string {
	return fmt.Sprint(*h)
}

// Set method to implement flags.Value
func (h *HTTPParams) Set(value string) error {
	v := strings.SplitN(value, "=", 2)
	if len(v) != 2 {
		return errors.New("Expected `Key=Value`")
	}

	param := httpParam{
		[]byte(strings.TrimSpace(v[0])),
		[]byte(strings.TrimSpace(v[1])),
	}

	*h = append(*h, param)
	return nil
}

//
// Handling of --http-allow-method option
//

// HTTPMethods holds values for method allowed
type HTTPMethods [][]byte

func (h *HTTPMethods) String() string {
	return fmt.Sprint(*h)
}

// Set method to implement flags.Value
func (h *HTTPMethods) Set(value string) error {
	*h = append(*h, []byte(value))
	return nil
}

// Handling of --http-rewrite-url option
type UrlRewrite struct {
	Src    *regexp.Regexp
	Target []byte
}

// URLRewriteMap holds regexp and data to modify URL
type URLRewriteMap []UrlRewrite

func (r *URLRewriteMap) String() string {
	return fmt.Sprint(*r)
}

// Set method to implement flags.Value
func (r *URLRewriteMap) Set(value string) error {
	valArr := strings.SplitN(value, ":", 2)
	if len(valArr) < 2 {
		return errors.New("need both src and target, colon-delimited (ex. /a:/b)")
	}
	regexp, err := regexp.Compile(valArr[0])
	if err != nil {
		return err
	}
	*r = append(*r, UrlRewrite{Src: regexp, Target: []byte(valArr[1])})
	return nil
}

// Handling of --http-rewrite-header option
type HeaderRewrite struct {
	Header []byte
	Src    *regexp.Regexp
	Target []byte
}

// HeaderRewriteMap holds regexp and data to rewrite headers
type HeaderRewriteMap []HeaderRewrite

func (r *HeaderRewriteMap) String() string {
	return fmt.Sprint(*r)
}

// Set method to implement flags.Value
func (r *HeaderRewriteMap) Set(value string) error {
	headerArr := strings.SplitN(value, ":", 2)
	if len(headerArr) < 2 {
		return errors.New("need both header, regexp and rewrite target, colon-delimited (ex. Header: regexp,target)")
	}

	header := headerArr[0]
	valArr := strings.SplitN(strings.TrimSpace(headerArr[1]), ",", 2)

	if len(valArr) < 2 {
		return errors.New("need both header, regexp and rewrite target, colon-delimited (ex. Header: regexp,target)")
	}

	regexp, err := regexp.Compile(valArr[0])
	if err != nil {
		return err
	}
	*r = append(*r, HeaderRewrite{Header: []byte(header), Src: regexp, Target: []byte(valArr[1])})
	return nil
}

// Handling of --http-allow-url option
type urlRegexp struct {
	Regexp *regexp.Regexp
}

// HTTPURLRegexp a slice of regexp to match URLs
type HTTPURLRegexp []urlRegexp

func (r *HTTPURLRegexp) String() string {
	return fmt.Sprint(*r)
}

// Set method to implement flags.Value
func (r *HTTPURLRegexp) Set(value string) error {
	regexp, err := regexp.Compile(value)

	*r = append(*r, urlRegexp{Regexp: regexp})

	return err
}
