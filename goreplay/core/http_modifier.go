package core

import (
	"bytes"
	"encoding/base64"
	"hash/fnv"
	"record-traffic-press/goreplay/proto"
	"record-traffic-press/goreplay/settings"
	"strings"
)

type HTTPModifier struct {
	config *settings.HTTPModifierConfig
}

func NewHTTPModifier(config *settings.HTTPModifierConfig) *HTTPModifier {
	// Optimization to skip modifier completely if we do not need it
	if len(config.URLRegexp) == 0 &&
		len(config.URLNegativeRegexp) == 0 &&
		len(config.URLRewrite) == 0 &&
		len(config.HeaderRewrite) == 0 &&
		len(config.HeaderFilters) == 0 &&
		len(config.HeaderNegativeFilters) == 0 &&
		len(config.HeaderBasicAuthFilters) == 0 &&
		len(config.HeaderHashFilters) == 0 &&
		len(config.ParamHashFilters) == 0 &&
		len(config.Params) == 0 &&
		len(config.Headers) == 0 &&
		len(config.Methods) == 0 {
		return nil
	}

	return &HTTPModifier{config: config}
}

func (m *HTTPModifier) Rewrite(payload []byte) (response []byte) {
	if !proto.HasRequestTitle(payload) {
		return payload
	}

	if len(m.config.Methods) > 0 {
		method := proto.Method(payload)

		matched := false

		for _, m := range m.config.Methods {
			if bytes.Equal(method, m) {
				matched = true
				break
			}
		}

		if !matched {
			return
		}
	}

	if len(m.config.Headers) > 0 {
		for _, header := range m.config.Headers {
			payload = proto.SetHeader(payload, []byte(header.Name), []byte(header.Value))
		}
	}

	if len(m.config.Params) > 0 {
		for _, param := range m.config.Params {
			payload = proto.SetPathParam(payload, param.Name, param.Value)
		}
	}

	if len(m.config.URLRegexp) > 0 {
		path := proto.Path(payload)

		matched := false

		for _, f := range m.config.URLRegexp {
			if f.Regexp.Match(path) {
				matched = true
				break
			}
		}

		if !matched {
			return
		}
	}

	if len(m.config.URLNegativeRegexp) > 0 {
		path := proto.Path(payload)

		for _, f := range m.config.URLNegativeRegexp {
			if f.Regexp.Match(path) {
				return
			}
		}
	}

	if len(m.config.HeaderFilters) > 0 {
		for _, f := range m.config.HeaderFilters {
			value := proto.Header(payload, f.Name)

			if len(value) == 0 {
				return
			}

			if !f.Regexp.Match(value) {
				return
			}
		}
	}

	if len(m.config.HeaderNegativeFilters) > 0 {
		for _, f := range m.config.HeaderNegativeFilters {
			value := proto.Header(payload, f.Name)

			if len(value) > 0 && f.Regexp.Match(value) {
				return
			}
		}
	}

	if len(m.config.HeaderBasicAuthFilters) > 0 {
		for _, f := range m.config.HeaderBasicAuthFilters {
			value := proto.Header(payload, []byte("Authorization"))

			if len(value) > 0 {
				valueString := string(value)
				trimmedBasicAuthEncoded := strings.TrimPrefix(valueString, "Basic ")
				if strings.Compare(valueString, trimmedBasicAuthEncoded) != 0 {
					decodedAuth, _ := base64.StdEncoding.DecodeString(trimmedBasicAuthEncoded)
					if !f.Regexp.Match(decodedAuth) {
						return
					}
				}
			}
		}
	}

	if len(m.config.HeaderHashFilters) > 0 {
		for _, f := range m.config.HeaderHashFilters {
			value := proto.Header(payload, f.Name)

			if len(value) > 0 {
				hasher := fnv.New32a()
				hasher.Write(value)

				if (hasher.Sum32() % 100) >= f.Percent {
					return
				}
			}
		}
	}

	if len(m.config.ParamHashFilters) > 0 {
		for _, f := range m.config.ParamHashFilters {
			value, s, _ := proto.PathParam(payload, f.Name)

			if s != -1 {
				hasher := fnv.New32a()
				hasher.Write(value)

				if (hasher.Sum32() % 100) >= f.Percent {
					return
				}
			}
		}
	}

	if len(m.config.URLRewrite) > 0 {
		path := proto.Path(payload)

		for _, f := range m.config.URLRewrite {
			if f.Src.Match(path) {
				path = f.Src.ReplaceAll(path, f.Target)
				payload = proto.SetPath(payload, path)

				break
			}
		}
	}

	if len(m.config.HeaderRewrite) > 0 {
		for _, f := range m.config.HeaderRewrite {
			value := proto.Header(payload, f.Header)
			if len(value) == 0 {
				break
			}

			if f.Src.Match(value) {
				newValue := f.Src.ReplaceAll(value, f.Target)
				payload = proto.SetHeader(payload, f.Header, newValue)
			}
		}
	}

	return payload
}
