package api

import (
	"log"
	"net/http"

	"github.com/xue-ding-e/swag/testdata/alias_import/data"
	"github.com/xue-ding-e/swag/testdata/alias_type/types"
)

// @Summary Get application
// @Description test get application
// @ID get-application
// @Accept  json
// @Produce  json
// @Success 200 {object} data.ApplicationResponse	"ok"
// @Router /testapi/application [get]
func GetApplication(w http.ResponseWriter, r *http.Request) {
	var foo = data.ApplicationResponse{
		Application: types.Application{
			Name: "name",
		},
		ApplicationArray: []types.Application{
			{Name: "name"},
		},
	}
	log.Println(foo)
	//write your code
}
