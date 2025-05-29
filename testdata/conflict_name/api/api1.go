package api

import (
	_ "github.com/xue-ding-e/swag/testdata/conflict_name/model"
	"net/http"
)

// @Tags Health
// @Description  Check if Health  of service it's OK!
// @ID health
// @Accept  json
// @Produce  json
// @Success 200 {object} model.ErrorsResponse
// @Router /health [get]
func Get1(w http.ResponseWriter, r *http.Request) {

}
