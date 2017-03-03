package util

import (
	"io"
	"net/http"
)

func DoHTTPRequest(method, url string, body io.Reader) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}
