package pterodactyl

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	baseURL string
	headers http.Header
}

func NewClient(baseURL, apiKey string) *Client {
	h := http.Header{}
	h.Add("Authorization", "Bearer "+apiKey)
	h.Add("Content-Type", "application/json")
	h.Add("Accept", "Application/vnd.pterodactyl.v1+json")

	return &Client{
		baseURL: baseURL,
		headers: h,
	}
}
func (c *Client) ServerReadFile(serverID string, filePath string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+fmt.Sprintf("/api/client/servers/%s/files/contents", serverID), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header = c.headers
	q := req.URL.Query()
	q.Add("file", filePath)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	return res, nil
}
