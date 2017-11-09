package resterrors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	errorFormat "github.com/normegil/formats/error"
	timeFormat "github.com/normegil/formats/time"
	urlFormat "github.com/normegil/formats/url"
	"github.com/sirupsen/logrus"
)

const DEFAULT_CODE = 500

type Handler struct {
	Log logrus.FieldLogger
}

func (h Handler) Handle(w http.ResponseWriter, e error) {
	log := h.Log
	stacks := stacks(e)
	if len(stacks) > 0 {
		log = log.WithField("errorStack", stacks[0])
	}
	log.WithError(e).Error("Error while processing request")

	responseBody := toResponse(e)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseBody.HTTPStatus)
	responseBodyJSON, err := json.Marshal(responseBody)
	if nil != err {
		h.Log.WithError(err).Error("An error happened while trying to marshall an other error")
		return
	}
	log.WithField("headers", w.Header()).Debug("Headers of error response")
	fmt.Fprintf(w, string(responseBodyJSON))
}

func toResponse(e error) *ErrorResponse {
	eWithCode := getErrWithCode(e)
	eMarshallable, isMarshable := e.(marshableError)
	if !isMarshable {
		eMarshallable = errorFormat.Error{e.Error()}
	}

	for _, defResp := range predefinedErrors {
		if eWithCode.Code() == defResp.Code {
			return &ErrorResponse{
				Code:       defResp.Code,
				HTTPStatus: defResp.HTTPStatus,
				Message:    defResp.Message,
				MoreInfo:   defResp.MoreInfo,
				Time:       timeFormat.Time(time.Now()),
				Err:        eMarshallable,
			}
		}
	}

	moreInfo, err := url.Parse("http://example.com/5000")
	if nil != err {
		panic(err)
	}
	return &ErrorResponse{
		Code:       50000,
		HTTPStatus: 500,
		Err:        errorFormat.Error{e.Error()},
		MoreInfo:   urlFormat.URL{moreInfo},
		Time:       timeFormat.Time(time.Now()),
		Message:    "Error was not found in the error ressources. Generated a default error.",
	}
}
