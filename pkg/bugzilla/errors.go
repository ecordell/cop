package bugzilla

import (
	"fmt"
	"net/http"
)

type requestError struct {
	statusCode int
	message    string
}

func (e requestError) Error() string {
	return e.message
}

func IsNotFound(err error) bool {
	reqError, ok := err.(*requestError)
	if !ok {
		return false
	}
	return reqError.statusCode == http.StatusNotFound
}

type identifierNotForPull struct {
	identifier string
}

func (i identifierNotForPull) Error() string {
	return fmt.Sprintf("identifier %q is not for a pull request", i.identifier)
}

func IsIdentifierNotForPullErr(err error) bool {
	_, ok := err.(*identifierNotForPull)
	return ok
}