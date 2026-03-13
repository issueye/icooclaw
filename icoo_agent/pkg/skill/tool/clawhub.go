package tool

import (
	"net/http"
)

type clawHubProvider struct {
	client *http.Client
}
