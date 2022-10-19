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

func get(rawURL string, doAuth bool, token string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer r.Body.Close()

	// 4xx some client error
	if r.StatusCode/100 == 4 {
		if doAuth && r.StatusCode == http.StatusUnauthorized {
			wwwAuth := r.Header.Get("WWW-Authenticate")
			if wwwAuth == "" {
				return fmt.Errorf("no WWW-Authenticate found")
			}

			w, wwwAuthErr := ParseWWWAuth(wwwAuth)
			if wwwAuthErr != nil {
				return wwwAuthErr
			}

			authURLValues := url.Values{}
			authURLValues.Set("service", w.Params["service"])
			authURLValues.Set("scope", w.Params["scope"])
			authURL, authURLErr := url.Parse(w.Params["realm"])
			if authURLErr != nil {
				return authURLErr
			}
			authURL.RawQuery = authURLValues.Encode()

			var authResp struct {
				Token string `json:"token"`
			}
			authTokenErr := get(authURL.String(), false, "", &authResp)
			if authTokenErr != nil {
				return authTokenErr
			}

			return get(rawURL, false, authResp.Token, out)
		}
		return fmt.Errorf(r.Status)
	}

	// not 2xx success
	if r.StatusCode/100 != 2 {
		return fmt.Errorf("error response: %s", r.Status)
	}

	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return fmt.Errorf("failed parse response: %w", err)
	}

	return nil
}

func (r *Registry) Tags() ([]string, error) {
	var resp struct {
		Tags []string `json:"tags"`
	}
	if err := get(fmt.Sprintf(listTagsURLTemplate, r.Host, r.Image), true, "", &resp); err != nil {
		return nil, err
	}
	return resp.Tags, nil
}
