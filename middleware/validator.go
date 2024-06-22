package middleware

import (
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremeres "github.com/globalxtreme/go-core/v2/response"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"
	"time"
)

type Validator struct{}

func (v Validator) Make(r *http.Request, rules interface{}) {
	err := xtremepkg.XtremeValidate.Struct(rules)
	if err != nil {
		var attributes []interface{}
		for _, e := range err.(validator.ValidationErrors) {
			attributes = append(attributes, map[string]interface{}{
				"param":   e.Field(),
				"message": getMessage(e.Error()),
			})
		}

		xtremeres.ErrXtremeValidation(attributes)
	}
}

func (v Validator) RegisterValidation(callback func(validate *validator.Validate)) {
	xtremepkg.XtremeValidate = validator.New()

	_ = xtremepkg.XtremeValidate.RegisterValidation("date_ddmmyyyy", dateDDMMYYYYValidation)
	_ = xtremepkg.XtremeValidate.RegisterValidation("time_hhmm", dateHHMMValidation)
	_ = xtremepkg.XtremeValidate.RegisterValidation("time_hhmmss", dateHHMMSSValidation)

	callback(xtremepkg.XtremeValidate)
}

func getMessage(errMsg string) string {
	splitMsg := strings.Split(errMsg, ":")
	key := 0
	if len(splitMsg) == 3 {
		key = 2
	} else if len(splitMsg) == 2 {
		key = 1
	}

	return splitMsg[key]
}

func dateDDMMYYYYValidation(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if field == "" {
		return true
	}

	_, err := time.Parse("02/01/2006", field)
	return err == nil
}

func dateHHMMValidation(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if field == "" {
		return true
	}

	_, err := time.Parse("15:04", field)
	return err == nil
}

func dateHHMMSSValidation(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if field == "" {
		return true
	}

	_, err := time.Parse("15:04:05", field)
	return err == nil
}
