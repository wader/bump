package dockerv2

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Registry struct {
	Host  string
	Image string
	Token string
}

var defaultRegistry = Registry{
	Host: "index.docker.io",
}

const listTagsURLTemplate = `https://%s/v2/%s/tags/list`

func NewFromImage(image string) (*Registry, error) {
	parts := strings.Split(image, "/")
	r := defaultRegistry
	switch {
	case len(parts) == 0:
		return &r, fmt.Errorf("invalid image")
	case len(parts) == 1:
		// image
		r.Image = "library/" + image
		return &r, nil
	case strings.Contains(parts[0], "."):
		// host.tldr/image
		r.Host = parts[0]
		r.Image = strings.Join(parts[1:], "/")
		return &r, nil
	default:
		// repo/image
		r.Image = image
		return &r, nil
	}
}

// The WWW-Authenticate Response Header Field
// https://www.rfc-editor.org/rfc/rfc6750#section-3
type WWWAuth struct {
	Scheme string
	Params map[string]string
}

// quoteSplit splits but respects quotes and escapes, and can mix quotes
func quoteSplit(s string, sep rune) ([]string, error) {
	r := csv.NewReader(strings.NewReader(s))
	// allows mix quotes and explicit ","
	r.LazyQuotes = true
	r.Comma = rune(sep)
	return r.Read()
}

// WWW-Authenticate: Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:org/image:pull"
func ParseWWWAuth(s string) (WWWAuth, error) {
	var w WWWAuth
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return WWWAuth{}, fmt.Errorf("invalid params")
	}
	w.Scheme = parts[0]

	pairs, pairsErr := quoteSplit(strings.TrimSpace(parts[1]), ',')
	if pairsErr != nil {
		return WWWAuth{}, pairsErr
	}

	w.Params = map[string]string{}
	for _, p := range pairs {
		kv, kvErr := quoteSplit(p, '=')
		if kvErr != nil {
			return WWWAuth{}, kvErr
		}
		if len(kv) != 2 {
			return WWWAuth{}, fmt.Errorf("invalid pair")
		}
		w.Params[kv[0]] = kv[1]
	}

	return w, nil
}

// </v2/library/debian/tags/list?last=oldoldstable-20210208&n=1000>; rel="next"

type LinkHeaderPart struct {
	RawURL string
	Params map[string]string
}

func ParseLinkHeader(s string) ([]LinkHeaderPart, error) {
	var parts []LinkHeaderPart

	rawParts, err := quoteSplit(s, ',')
	if err != nil {
		return nil, err
	}
	for _, rawPart := range rawParts {
		part := LinkHeaderPart{
			Params: map[string]string{},
		}

		params, err := quoteSplit(rawPart, ';')
		if err != nil {
			return nil, err
		}

		for _, param := range params {
			param = strings.TrimSpace(param)

			if strings.HasPrefix(param, "<") && strings.HasSuffix(param, ">") {
				part.RawURL = param[1 : len(param)-1]
			} else {
				keyValue, keyValueErr := quoteSplit(param, '=')
				if keyValueErr != nil {
					return nil, keyValueErr
				}
				if len(keyValue) != 2 {
					continue
				}
				part.Params[keyValue[0]] = keyValue[1]

			}
		}

		parts = append(parts, part)
	}

	return parts, nil
}

type authRespBody struct {
	Token string `json:"token"`
}

type getResp[T any] struct {
	Body       T
	AuthHeader string
	NextRawURL string
}

func get[T any](rawURL string, doAuth bool, authHeader string) (getResp[T], error) {
	var resp getResp[T]

	resp.AuthHeader = authHeader

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return resp, err
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, fmt.Errorf("request failed: %w", err)
	}
	defer r.Body.Close()

	// 4xx some client error
	if r.StatusCode/100 == 4 {
		if doAuth && r.StatusCode == http.StatusUnauthorized {
			wwwAuth := r.Header.Get("WWW-Authenticate")
			if wwwAuth == "" {
				return resp, fmt.Errorf("no WWW-Authenticate found")
			}

			w, wwwAuthErr := ParseWWWAuth(wwwAuth)
			if wwwAuthErr != nil {
				return resp, wwwAuthErr
			}

			authURLValues := url.Values{}
			authURLValues.Set("service", w.Params["service"])
			authURLValues.Set("scope", w.Params["scope"])
			authURL, authURLErr := url.Parse(w.Params["realm"])
			if authURLErr != nil {
				return resp, authURLErr
			}
			authURL.RawQuery = authURLValues.Encode()

			authResp, authTokenErr := get[authRespBody](authURL.String(), false, "")
			if authTokenErr != nil {
				return resp, authTokenErr
			}

			return get[T](rawURL, false, fmt.Sprintf("Bearer %s", authResp.Body.Token))
		}
		return resp, fmt.Errorf(r.Status)
	}

	// not 2xx success
	if r.StatusCode/100 != 2 {
		return resp, fmt.Errorf("error response: %s", r.Status)
	}

	if link := r.Header.Get("Link"); link != "" {
		parts, partsErr := ParseLinkHeader(link)
		if partsErr != nil {
			return resp, partsErr
		}
		for _, part := range parts {
			if v, ok := part.Params["rel"]; ok && v == "next" {
				resp.NextRawURL = part.RawURL
				break
			}
		}
	}

	if err := json.NewDecoder(r.Body).Decode(&resp.Body); err != nil {
		return resp, fmt.Errorf("failed parse response: %w", err)
	}

	return resp, nil
}

type respBody struct {
	Tags []string `json:"tags"`
}

func getPaged[T any](rawURL string, doAuth bool, token string) ([]T, error) {
	var vs []T

	u, uErr := url.Parse(rawURL)
	if uErr != nil {
		return nil, uErr
	}

	authHeader := ""
	const maxNext = 1000

	for i := 0; true; i++ {
		resp, err := get[T](rawURL, true, authHeader)
		if err != nil {
			return nil, err
		}
		vs = append(vs, resp.Body)

		if resp.NextRawURL == "" {
			break
		}

		nextURL, nextURLErr := url.Parse(resp.NextRawURL)
		if nextURLErr != nil {
			return nil, nextURLErr
		}
		rawURL = u.ResolveReference(nextURL).String()
		authHeader = resp.AuthHeader

		if i > maxNext {
			return nil, fmt.Errorf("max next links (%d) reached", maxNext)
		}
	}

	return vs, nil
}

func (r *Registry) Tags() ([]string, error) {
	resps, err := getPaged[respBody](fmt.Sprintf(listTagsURLTemplate, r.Host, r.Image), true, "")
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, resp := range resps {
		tags = append(tags, resp.Tags...)
	}

	return tags, nil
}
